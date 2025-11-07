package types

import "encoding/json"

// DocumentUpdate é a estrutura que será propagada via PubSub
type DocumentUpdate struct {
    VectorVersion int       `json:"version"`
    NewCID        string    `json:"cid"`
    Embeddings    [][]float32 `json:"embeddings"`
    UpdatedVector []string  `json:"updatedVector"`
    Timestamp     int64     `json:"timestamp"`
}

// VectorState mantém o estado do vetor com versionamento
type VectorState struct {
    CurrentVersion int
    CurrentVector  []string
    PendingVector  []string
}

func (d DocumentUpdate) ToJSON() (string, error) {
    data, err := json.Marshal(d)
    return string(data), err
}
