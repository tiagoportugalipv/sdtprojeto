package messaging

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
	"os"

    "github.com/gin-gonic/gin"
    pubsub "github.com/libp2p/go-libp2p-pubsub"
    "github.com/ipfs/kubo/core"
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
	Leader  bool
}

type SendMessageReq struct {
    Content string `json:"content"`
}

func NewPubSubService(ctx context.Context, node *core.IpfsNode, topicName string) (*PubSubService, error) {

    ps, err := pubsub.NewGossipSub(ctx, node.PeerHost)
    if err != nil {
        return nil, err
    }

    topic, err := ps.Join(topicName)
    if err != nil {
        return nil, err
    }

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

	if service.Leader {
    	_ = service.PublishMessage("Líder online: " + service.peerID)
	}

    go service.listenMessages()

    return service, nil
}

func (s *PubSubService) listenMessages() {
    for {
        msg, err := s.sub.Next(s.ctx)
        if err != nil {
            log.Printf("Error receiving message: %v", err)
            continue
        }

        log.Printf("DBG ReceivedFrom=%s self=%s", msg.ReceivedFrom.String(), s.peerID)

        var message Message
        err = json.Unmarshal(msg.Data, &message)
        if err != nil {
            log.Printf("Error unmarshaling message: %v", err)
            continue
        }

        // Ignora apenas mensagens que foram publicadas por este próprio peer
        if message.From == s.peerID {
            continue
        }

        log.Printf("[MESSAGE] From: %s | Content: %s", message.From, message.Content)
    }
}

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

func (s *PubSubService) Close() error {
    s.sub.Cancel()
    return s.topic.Close()
}

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
