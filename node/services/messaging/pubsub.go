// Pacote responsável por funcionalidades de Pub/Sub (publicação e subscrição de mensagens)
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"sdt/node/types"

	"github.com/ipfs/kubo/core"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type Message struct {
	From    string   `json:"from"`
	Content []string `json:"content"`
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
	/*if service.Leader {
		_ = service.PublishMessage("Líder online: " + service.peerID)
	}*/

	// Inicia a rotina de escuta de mensagens
	go service.listenMessages()

	return service, nil
}

// listenMessages corre em background e processa mensagens recebidas do tópico
func (s *PubSubService) listenMessages() {
	for {
		msg, err := s.sub.Next(s.ctx)
		if err != nil {
			log.Printf("Error receiving message: %v", err)
			return
		}

		var update types.DocumentUpdate
		if err := json.Unmarshal(msg.Data, &update); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Aqui os peers recebem e armazenam
		log.Printf("[PEER %s] Received CID: %s (v%d), Embeddings dim: %d",
			s.peerID, update.NewCID, update.VectorVersion, len(update.Embeddings))
	}
}

// PublishMessage serializa e publica uma mensagem no tópico atual
func (s *PubSubService) PublishMessage(messageJSON string) error {
	if s.topic == nil {
		return fmt.Errorf("topic not initialized")
	}

	if err := s.topic.Publish(s.ctx, []byte(messageJSON)); err != nil {
		return err
	}

	log.Printf("[%s] Published: %s", s.peerID, messageJSON)
	return nil
}
