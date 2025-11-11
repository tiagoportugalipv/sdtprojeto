package messaging

import (
	// bibs padrão
	"fmt"
	"context"
    "bytes"
    "encoding/gob"
	"time"

	// bibs internas
	"projeto/node"

)

// https://thesecretlivesofdata.com/raft/#replication

type Topico string

const (
	AEM Topico = "aem" // AppendEntryMessage 
	TXT Topico = "txt"
)

type Message struct {
    From string 
}

type TextMessage struct {
	Message
	Text string 
}

type AppendEntryMessage struct {
	Message
	Vector node.Vector 
	Embeddings [][]float32 
}

func PublishTo(nd *node.Node, topico Topico, msg any)(error){

    var err error
    buf := new(bytes.Buffer)
    encoder := gob.NewEncoder(buf)
	pubsubInt := nd.IpfsApi.PubSub()

    if err = encoder.Encode(msg); err != nil {
        return fmt.Errorf("Falha a codificar mensagem: %v\n", err)
    }


	publishCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = pubsubInt.Publish(publishCtx,string(topico),buf.Bytes())

	if err != nil {
		return fmt.Errorf("Falha ao enviar mensagem: %v\n", err)
	}

	return nil

}
