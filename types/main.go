package types

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"strings"
)


// Estruturas Base

type Vector struct {
	Ver int // Versão
	Content []string // Array de CIDs
}

type CidVectorEmbsEntry struct {
	InIndex bool
	Content []float32
}


type ResquestType int


const (
    ADD ResquestType = 1
    PROMPT ResquestType = 2
    GET ResquestType = 3
)



func (v Vector) Hash() (string) {

    // Nota : não utilizar gob para gerar hashes se não temos o caldo entornado
    // pelo que percebi o gob bytes ao conteudo original em prol da eficiencia de unmarshaling
    // no que resulta hash diferentes para dados iguais

    jsonBytes, err := json.Marshal(v) 
    if err != nil {
        panic(err)
    }

    sum := sha256.Sum256(jsonBytes)
    return hex.EncodeToString(sum[:])

}

func (v Vector) String() (string) {

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Version : %d\n[\n",v.Ver))

	for index,element := range v.Content{
		builder.WriteString(fmt.Sprintf("\tCID%d | %s\n",index,element))
	}

	builder.WriteString("]")

	return builder.String()

}
