package embedding

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/knights-analytics/hugot"
)

var ModelDirPath string
var ModelPath string

// SetUpModel verifica se o modelo existe localmente; caso contrário, faz download.
func SetUpModel() (error, string) {
    ModelPath = filepath.Join(
        ModelDirPath,
        "sentence-transformers_all-MiniLM-L6-v2",
        "onnx",
        "model.onnx",
    )

    // Verifica se o modelo já existe
    if _, err := os.Stat(ModelPath); err == nil {
        return nil, ModelPath
    }

    // Caso não exista, faz download
    downloadOptions := hugot.NewDownloadOptions()
    downloadOptions.OnnxFilePath = "onnx/model.onnx"

    downloadedPath, err := hugot.DownloadModel(
        "sentence-transformers/all-MiniLM-L6-v2",
        ModelDirPath,
        downloadOptions,
    )
    if err != nil {
        return fmt.Errorf("Erro ao fazer download: %v", err), ""
    }

    ModelPath = downloadedPath
    return nil, ModelPath
}


// GetEmbeddings gera embeddings para um conjunto de strings.
//TODO: Mudar para um array com uma unica string 
func GetEmbeddings(file []string) ([][]float32, error) {

    if len(file) == 0 {
        return nil, fmt.Errorf("input fileLines is empty, cannot generate embeddings")
    }

    session, err := hugot.NewGoSession()
    if err != nil {
        return nil, fmt.Errorf("Falha a obter sessão go para hugot: %v", err)
    }
    defer session.Destroy()

    config := hugot.FeatureExtractionConfig{
        ModelPath: ModelPath,
        Name:      "embeddingPipeline",
    }

    embeddingPipeline, err := hugot.NewPipeline(session, config)
    if err != nil {
        return nil, fmt.Errorf(
            "Falha ao criar embedding pipeline: %v\nCaminho para o modelo: %v",
            err, ModelPath,
        )
    }

    results, err := embeddingPipeline.RunPipeline(file)
    if err != nil {
        return nil, fmt.Errorf("Falha ao gerar embeddings: %v", err)
    }

    return results.Embeddings, nil
}
