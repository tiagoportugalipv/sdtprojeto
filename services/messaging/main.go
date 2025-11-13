package messaging

import (
	// bibs padrão
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	// bibs internas
	"projeto/types"

	// bibs externas
	"github.com/libp2p/go-libp2p/core/peer"
	iface "github.com/ipfs/kubo/core/coreiface"
)

// https://thesecretlivesofdata.com/raft/#replication

type Topico string
type Vector = types.Vector

const (
    AEM Topico = "aem" // AppendEntryMessage 
    TXT Topico = "txt"
    ACK Topico = "ack"
    COMM Topico = "commit"
)


type TextMessage struct {
    Text string 
}


type AckMessage struct {
    Hash string 
}


type CommitMessage struct {
    Version int 
}

type AppendEntryMessage struct {
    Vector Vector 
    Embeddings [][]float32 
}

func PublishTo(pubsubInt iface.PubSubAPI, topico Topico, msg any)(error){

    var err error
    buf := new(bytes.Buffer)
    encoder := gob.NewEncoder(buf)

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

func ListenTo(pubsubInt iface.PubSubAPI, topico Topico, callback func(sender peer.ID, msg any)) error {

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
        case ACK:
            var ackMsg AckMessage
            if err := decoder.Decode(&ackMsg); err != nil {
                return fmt.Errorf("Erro ao decodificar AckMessage: %v", err)
            }
            msg = ackMsg
        case COMM:
            var commMsg CommitMessage
            if err := decoder.Decode(&commMsg); err != nil {
                return fmt.Errorf("Erro ao decodificar AckMessage: %v", err)
            }
            msg = commMsg
        default:
            return fmt.Errorf("Topico desconhecido: %v", topico)
        }

        callback(pubSubMsg.From(), msg)
    }
}







