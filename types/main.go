package types

import (
	"crypto/sha256"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"strings"
)

// Flags

var UploadCompleted = false

// Estruturas Base

type Vector struct {
	Ver int
	Content []string
}


func (v Vector) Hash() (string) {

	// Nota : não utilizar gob para gerar hashes se não temos o caldo entornado

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
