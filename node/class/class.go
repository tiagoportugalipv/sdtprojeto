package main

//Mensagem base
type baseMessage struct {
	From string `"json:from"`
}

//Vector que envia os cids e o embedding
type newVectorMessage struct {
	baseMessage
	Content    []string    `"json:content"`
	embeddings [][]float32 `"json:embeddings"`
}

//Hash que os peers enviam para o lider
type hashACK struct {
	baseMessage
	Content []string `"json:content"`
}

//commit
type commit struct {
	baseMessage
	Content []string `"json:content"`
}
