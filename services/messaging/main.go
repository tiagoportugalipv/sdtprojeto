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


type Topico string
type Vector = types.Vector

const (
    AEM Topico = "aem" // Topico AppendEntryMessage 
    TXT Topico = "txt" // Topico TextMessage
    ACK Topico = "ack" // Topico Ack
    COMM Topico = "commit" // Topico Commit
    HTB Topico = "heartbeat" // Topico Heartbeat
    RBLQ Topico = "rebuildquery" // Topico Rebuild
    RBLR Topico = "rebuildreponse" // Topico Rebuild
)


type TextMessage struct {
    Text string 
}


type AckMessage struct {
    Version int
    Hash string 
}


type CommitMessage struct {
    Version int 
}

type AppendEntryMessage struct {
    Vector Vector 
    Embeddings []float32 
}

type HeartBeatMessage struct {
    Npeers int
}

type RebuildQueryMessage struct {
    Info []string 
    Dest peer.ID
}


type RebuildResponseMessage struct {
    Response map[string][]float32
    Total bool
    Dest peer.ID
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
        fmt.Println(err)
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

func decodeMessage(data []byte,topico Topico) (any,error){

    buf := bytes.NewBuffer(data)
    decoder := gob.NewDecoder(buf)


    var msg any

    switch topico {
    case TXT:
        var txtMsg TextMessage
        if err := decoder.Decode(&txtMsg); err != nil {
        fmt.Println(err)
            return nil,fmt.Errorf("Erro ao decodificar TextMessage: %v", err)
        }
        msg = txtMsg
    case AEM:
        var aemMsg AppendEntryMessage
        if err := decoder.Decode(&aemMsg); err != nil {
        fmt.Println(err)
            return nil,fmt.Errorf("Erro ao decodificar AppendEntryMessage: %v", err)
        }
        msg = aemMsg
    case ACK:
        var ackMsg AckMessage
        if err := decoder.Decode(&ackMsg); err != nil {
        fmt.Println(err)
            return nil,fmt.Errorf("Erro ao decodificar AckMessage: %v", err)
        }
        msg = ackMsg
    case COMM:
        var commMsg CommitMessage
        if err := decoder.Decode(&commMsg); err != nil {
        fmt.Println(err)
            return nil,fmt.Errorf("Erro ao decodificar AckMessage: %v", err)
        }
        msg = commMsg
    case HTB:
        var heartbeatMsg HeartBeatMessage
        if err := decoder.Decode(&heartbeatMsg); err != nil {
        fmt.Println(err)
            return nil,fmt.Errorf("Erro ao decodificar HeartBeatMessage: %v", err)
        }
        msg = heartbeatMsg

    case RBLQ:
        var rebuildQueryMsg RebuildQueryMessage
        if err := decoder.Decode(&rebuildQueryMsg); err != nil {
        fmt.Println(err)
            return nil,fmt.Errorf("Erro ao decodificar RebuildQueryMessage: %v", err)
        }
        msg = rebuildQueryMsg

    case RBLR:

        var rebuildResponseMsg RebuildResponseMessage
        if err := decoder.Decode(&rebuildResponseMsg); err != nil {
        fmt.Println(err)
            return nil,fmt.Errorf("Erro ao decodificar JoinMessage: %v", err)
        }
        msg = rebuildResponseMsg

    default:
        return nil,fmt.Errorf("Topico desconhecido: %v", topico)
    }

    return msg,nil


}


// Receber mensagens
func ListenTo(pubsubInt iface.PubSubAPI, topico Topico, callback func(sender peer.ID, msg any, stop *bool)) error {

    subscribeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    sub, err := pubsubInt.Subscribe(subscribeCtx, string(topico))
    if err != nil {
        return fmt.Errorf("Falha ao subscrever topico: %v", err)
    }

    st := false

    for {


        pubSubMsg, err := sub.Next(context.Background())
        if err != nil {
            return fmt.Errorf("Erro ao ouvir menssagens: %v", err)
        }

        // Unmarshaling da mensagens

        msg,err := decodeMessage(pubSubMsg.Data(),topico)

        if(err != nil){
            return err
        }

        callback(pubSubMsg.From(), msg, &st)

        if(st){
            break
        }

    }

    return nil
}







