package messaging

import (
	// bibs padrão
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	// bibs internas
	"projeto/node"

	// bibs externas
	"github.com/libp2p/go-libp2p/core/peer"
)

// https://thesecretlivesofdata.com/raft/#replication

type Topico string

const (
    AEM Topico = "aem" // AppendEntryMessage 
    TXT Topico = "txt"
)


type TextMessage struct {
    Text string 
}

type AppendEntryMessage struct {
    Vector node.Vector 
    Embeddings [][]float32 
}

func PublishTo(nd *node.Node, topico Topico, msg any)(error){

    var err error
    buf := new(bytes.Buffer)
    encoder := gob.NewEncoder(buf)
    pubsubInt := nd.IpfsApi.PubSub()

    if err = encoder.Encode(msg); err != nil {
        return fmt.Errorf("Falha ao dar marshall da mensagem: %v\n", err)
    }


    publishCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err = pubsubInt.Publish(publishCtx,string(topico),buf.Bytes())

    if err != nil {
	return fmt.Errorf("Falha ao enviar mensagem: %v\n", err)
    }

    return nil

}

func ListenTo(nd *node.Node, topico Topico, callback func(sender peer.ID, msg any)) error {

    pubsubInt := nd.IpfsApi.PubSub()

    subscribeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    sub, err := pubsubInt.Subscribe(subscribeCtx, string(topico))
    if err != nil {
        return fmt.Errorf("Falha ao subscrever topico: %v", err)
    }

    for {
        pubSubMsg, err := sub.Next(context.Background())
        if err != nil {
            return fmt.Errorf("Erro ao ouvir menssagens: %v", err)
        }

        buf := bytes.NewBuffer(pubSubMsg.Data())
        decoder := gob.NewDecoder(buf)

        var msg any
        switch topico {
        case TXT:
            var txtMsg TextMessage
            if err := decoder.Decode(&txtMsg); err != nil {
                return fmt.Errorf("Erro ao decodificar TextMessage: %v", err)
            }
            msg = txtMsg
        case AEM:
            var aemMsg AppendEntryMessage
            if err := decoder.Decode(&aemMsg); err != nil {
                return fmt.Errorf("Erro ao decodificar AppendEntryMessage: %v", err)
            }
            msg = aemMsg
        default:
            return fmt.Errorf("Topico desconhecido: %v", topico)
        }

        callback(pubSubMsg.From(), msg)
    }
}







