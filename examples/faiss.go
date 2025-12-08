package main

// Substitui isto pelo main para testar

import (
    "bufio"
    "fmt"
    "os"
    "strings"

    "projeto/services/embedding"
    faiss "github.com/DataIntelligenceCrew/go-faiss"
)

func main() {

    // 1) Configurar diretório do modelo e garantir que está disponível
    embedding.ModelDirPath = "./models"
    if err, path := embedding.SetUpModel(); err != nil {
        panic(err)
    } else {
        fmt.Printf("Modelo em: %s\n", path)
    }

    // 2) Ler ficheiro linha a linha
    filePath := "files/sample.md"
    f, err := os.Open(filePath)
    if err != nil {
        panic(err)
    }
    defer f.Close()

    var lines []string
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line != "" {
            lines = append(lines, line)
        }
    }

    if err := scanner.Err(); err != nil {
        panic(err)
    }
    if len(lines) == 0 {
        fmt.Println("Ficheiro vazio depois de filtrar linhas.")
        return
    }

    // 3) Gerar embeddings
    embs, err := embedding.GetEmbeddings(lines)
    if err != nil {
        panic(err)
    }
    dim := len(embs[0])
    fmt.Printf("Linhas: %d | Dimensão: %d\n", len(embs), dim)

    // 4) Criar índice FAISS (Flat L2)
    index, err := faiss.NewIndexFlatL2(dim)
    if err != nil {
        panic(err)
    }
    defer index.Delete()

    // Adicionar embeddings ao índice (concatenar todos numa slice)
    var flat []float32
    for _, v := range embs {
        flat = append(flat, v...)
    }
    if err := index.Add(flat); err != nil {
        panic(err)
    }
    fmt.Println("Índice FAISS criado.")

    // 5) Perguntar query ao utilizador
    reader := bufio.NewReader(os.Stdin)
    fmt.Print("\nPesquisar frase: ")
    qText, _ := reader.ReadString('\n')
    qText = strings.TrimSpace(qText)
    if qText == "" {
        fmt.Println("Query vazia, a sair.")
        return
    }

    qEmb, err := embedding.GetEmbeddings([]string{qText})
    if err != nil {
        panic(err)
    }
    qVec := qEmb[0]

    // 6) Pesquisa FAISS
    var k int64 = 5
    dists, idxs, err := index.Search(qVec, k)
    if err != nil {
        panic(err)
    }

    // 7) Mostrar resultados
    fmt.Printf("\nTop %d resultados:\n", k)
    for i := int64(0); i < k && i < int64(len(idxs)); i++ {
        idx := idxs[i]
        if idx < 0 || idx >= int64(len(lines)) {
            continue
        }
        fmt.Printf("\n#%d (dist: %.4f)\n", i+1, dists[i])
        fmt.Println(lines[idx])
    }
}
