package messaging

import (
	// bibs padrão
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"time"

	// bibs internas
	"projeto/types"

	// bibs externas
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/libp2p/go-libp2p/core/peer"
)


type Topico string
type Vector = types.Vector

const (
    AEM Topico = "aem" // Topico AppendEntryMessage 
    TXT Topico = "txt" // Topico TextMessage
    ACK Topico = "ack" // Topico Ack
    COMM Topico = "commit" // Topico Commit
    HTB Topico = "heartbeat" // Topico Heartbeat
    RBLQ Topico = "rebuildquery" // Topico RebuildQuery
    RBLR Topico = "rebuildreponse" // Topico RebuildResponse
    CDTP Topico = "candidateproposal" // Topico CandidatePorposal
    VTP  Topico = "votingpool" // Topico VotingPool
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
    Dest peer.ID
}

type CandidatePorposalMessage struct {
    Term int
}

type VoteMessage struct {
    Term int
    Candidate peer.ID
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
            return nil,fmt.Errorf("Erro ao decodificar RebuildQueryResponse: %v", err)
        }
        msg = rebuildResponseMsg

    case CDTP:

        var candidatePorposalMsg CandidatePorposalMessage
        if err := decoder.Decode(&candidatePorposalMsg); err != nil {
        fmt.Println(err)
            return nil,fmt.Errorf("Erro ao decodificar CadidatePorposalMessage: %v", err)
        }
        msg = candidatePorposalMsg


    case VTP:

        var voteMsg VoteMessage
        if err := decoder.Decode(&voteMsg); err != nil {
        fmt.Println(err)
            return nil,fmt.Errorf("Erro ao decodificar VoteMessage: %v", err)
        }
        msg = voteMsg
        

    default:
        return nil,fmt.Errorf("Topico desconhecido: %v", topico)
    }

    return msg,nil


}


// Receber mensagens
func ListenTo(ctx context.Context, pubsubInt iface.PubSubAPI, topico Topico, callback func(sender peer.ID, msg any, stop *bool)) error {

    subscribeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    sub, err := pubsubInt.Subscribe(subscribeCtx, string(topico))
    if err != nil {
        return fmt.Errorf("Falha ao subscrever topico: %v", err)
    }
    defer sub.Close() // Fecha subscription quando terminar

    st := false

    for {
        select {
        case <-ctx.Done():
            // Contexto foi cancelado - terminar listener
            fmt.Printf("[LISTENER] Cancelando listener de %s\n", topico)
            return ctx.Err()
            
        default:
            // Timeout curto para não bloquear indefinidamente
            msgCtx, msgCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
            pubSubMsg, err := sub.Next(msgCtx)
            msgCancel()
            
            if err != nil {
                // Se for timeout, continua o loop para verificar ctx.Done()
                if errors.Is(err, context.DeadlineExceeded) {
                    continue
                }
                return fmt.Errorf("Erro ao ouvir mensagens: %v", err)
            }

            // Unmarshaling da mensagem
            msg, err := decodeMessage(pubSubMsg.Data(), topico)
            if err != nil {
                // Log do erro mas continua a ouvir
                fmt.Printf("[LISTENER] Erro ao decodificar mensagem: %v\n", err)
                continue
            }

            callback(pubSubMsg.From(), msg, &st)

            if st {
                fmt.Printf("[LISTENER] Parando listener de %s (stop flag)\n", topico)
                return nil
            }
        }
    }
}
