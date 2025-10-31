// Pacote responsável por funcionalidades de Pub/Sub (publicação e subscrição de mensagens)
package messaging

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/ipfs/kubo/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type Message struct {
	From    string `json:"from"`
	Content string `json:"content"`
}

type PubSubService struct {
	ps     *pubsub.PubSub
	topic  *pubsub.Topic
	sub    *pubsub.Subscription
	ctx    context.Context
	peerID string
	Leader bool // indica se este nó é considerado "líder" por variável de ambiente
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

        // Primeiro tenta interpretar como JSON {From, Content}
        var message Message
        if err := json.Unmarshal(msg.Data, &message); err == nil && (message.From != "" || message.Content != "") {
            log.Printf("[MESSAGE] From: %s | Content: %s", message.From, message.Content)
            continue
        }

        // Caso contrário, trata como texto bruto
        log.Printf("[MESSAGE] Raw: %s", string(msg.Data))
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


