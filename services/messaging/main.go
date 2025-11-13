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
    AEM Topico = "aem" // Topico AppendEntryMessage 
    TXT Topico = "txt" // Topico TextMessage
    ACK Topico = "ack" // Topico Ack
    COMM Topico = "commit" // Topico Commit
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

// Utilizamos gob pois suporta tipos nativos de go e é relativamente mais eficiente que json,
// sacrificando no entanto intercompatibilidade com outras linguagens

// Publicar Mensagens
func PublishTo(pubsubInt iface.PubSubAPI, topico Topico, msg any)(error){

    // Marshaling da mensagem

    var err error
    buf := new(bytes.Buffer)
    encoder := gob.NewEncoder(buf)

    if err = encoder.Encode(msg); err != nil {
        return fmt.Errorf("Falha ao dar marshall da mensagem: %v\n", err)
    }

    // Envio da mensagem


    publishCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err = pubsubInt.Publish(publishCtx,string(topico),buf.Bytes())

    if err != nil {
	return fmt.Errorf("Falha ao enviar mensagem: %v\n", err)
    }

    return nil

}

// Receber mensagens
func ListenTo(pubsubInt iface.PubSubAPI, topico Topico, callback func(sender peer.ID, msg any)) error {

    subscribeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    sub, err := pubsubInt.Subscribe(subscribeCtx, string(topico))
    if err != nil {
        return fmt.Errorf("Falha ao subscrever topico: %v", err)
    }

    for {

        // Receber mensagens
        pubSubMsg, err := sub.Next(context.Background())
        if err != nil {
            return fmt.Errorf("Erro ao ouvir menssagens: %v", err)
        }

        // Unmarshaling da mensagens

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







