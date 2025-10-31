// Pacote responsável por funcionalidades de Pub/Sub (publicação e subscrição de mensagens)
package messaging

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/ipfs/kubo/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type Message struct {
    From    string `json:"from"`
    Content string `json:"content"`
}

type PubSubService struct {
    ps      *pubsub.PubSub
    topic   *pubsub.Topic
    sub     *pubsub.Subscription
    ctx     context.Context
    peerID  string
    Leader  bool // indica se este nó é considerado "líder" por variável de ambiente
}

type SendMessageReq struct {
    Content string `json:"content"`
}

// NewPubSubService cria e inicializa o serviço de Pub/Sub para um dado tópico
func NewPubSubService(ctx context.Context, node *core.IpfsNode, topicName string) (*PubSubService, error) {

    // Cria uma instância GossipSub associada ao host libp2p do nó
    ps, err := pubsub.NewGossipSub(
        ctx,
        node.PeerHost,
        pubsub.WithPeerExchange(true), // ajuda a descobrir peers do tópico
    )
    if err != nil {
        return nil, err
    }

    // Junta-se ao tópico indicado
    topic, err := ps.Join(topicName)
    if err != nil {
        return nil, err
    }

    // Cria a subscrição para começar a receber mensagens do tópico
    sub, err := topic.Subscribe()
    if err != nil {
        return nil, err
    }

    service := &PubSubService{
        ps:     ps,
        topic:  topic,
        sub:    sub,
        ctx:    ctx,
        peerID: node.Identity.String(),
        Leader: os.Getenv("LEADER") == "1",
    }

    // Se for líder, anuncia a presença no arranque
    if service.Leader {
        _ = service.PublishMessage("Líder online: " + service.peerID)
    }

    // Inicia a rotina de escuta de mensagens
    go service.listenMessages()

    return service, nil
}

// listenMessages corre em background e processa mensagens recebidas do tópico
func (s *PubSubService) listenMessages() {
    for {
        msg, err := s.sub.Next(s.ctx)
        if err != nil {
            // Continua em caso de erro temporário ao receber
            log.Printf("Error receiving message: %v", err)
            continue
        }

        log.Printf("DBG ReceivedFrom=%s self=%s", msg.ReceivedFrom.String(), s.peerID)

        var message Message
        err = json.Unmarshal(msg.Data, &message)
        if err != nil {
            // Ignora mensagens com payload inválido
            log.Printf("Error unmarshaling message: %v", err)
            continue
        }

        // Ignora apenas mensagens cuja origem (autor) seja o próprio peer.
        // Nota: não usamos msg.ReceivedFrom para este filtro, pois indica o hop por onde chegou.
        if message.From == s.peerID {
            continue
        }

        log.Printf("[MESSAGE] From: %s | Content: %s", message.From, message.Content)
    }
}

// PublishMessage serializa e publica uma mensagem no tópico atual
func (s *PubSubService) PublishMessage(content string) error {
    msg := Message{
        From:    s.peerID,
        Content: content,
    }

    msgBytes, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    return s.topic.Publish(s.ctx, msgBytes)
}

// Close encerra a subscrição e fecha o tópico
func (s *PubSubService) Close() error {
    s.sub.Cancel()
    return s.topic.Close()
}

// SendMessageHandler é um handler HTTP (Gin) para publicar mensagens via POST JSON
// Espera um corpo { "content": "..." } e publica no tópico
func (s *PubSubService) SendMessageHandler(c *gin.Context) {
    var req SendMessageReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    err := s.PublishMessage(req.Content)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish message: " + err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Message published successfully",
        "content": req.Content,
    })
}
