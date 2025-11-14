package embedding

import (
    "fmt"
    "os"
    "path/filepath"
    "github.com/knights-analytics/hugot"
)

var ModelDirPath string
var ModelPath string

func SetUpModel() (error, string) {
    var err error
    
    localModelPath := filepath.Join(ModelDirPath, "all-MiniLM-L6-v2")
    ModelPath = filepath.Join(localModelPath, "onnx", "model.onnx")
    
    if _, err := os.Stat(ModelPath); err == nil {
        fmt.Printf("✓ Modelo encontrado localmente: %s\n", ModelPath)
        return nil, ModelPath  
    }
    
    fmt.Println("⚠ Modelo não encontrado. A fazer download...")
    downloadOptions := hugot.NewDownloadOptions()
    downloadOptions.OnnxFilePath = "onnx/model.onnx"
    
    ModelPath, err = hugot.DownloadModel(
        "sentence-transformers/all-MiniLM-L6-v2",
        ModelDirPath,
        downloadOptions,
    )
    
    if err != nil {
        return fmt.Errorf("Erro ao fazer download: %v", err), ""
    }
    
    return nil, ModelPath
}




func GetEmbeddings(fileLines []string)([][]float32,error){

    if len(fileLines) == 0 {
        return nil, fmt.Errorf("input fileLines is empty, cannot generate embeddings")
    }

    session, err := hugot.NewGoSession()

    if err != nil {
        err = fmt.Errorf("Falha a obter sessão go para hugot : %v\n",err) 
        return nil,err
    }


    config := hugot.FeatureExtractionConfig{
        ModelPath: ModelPath,
        Name:      "embeddingPipeline",
    }

    embeddingPipeline, err := hugot.NewPipeline(session, config)

    if(err != nil){
        err = fmt.Errorf("Falha ao criar embedding pipeline : %v\n",err) 
        return nil,err
    }

    pipelineResults,err := embeddingPipeline.RunPipeline(fileLines)


    if(err != nil){
        err = fmt.Errorf("Falha ao gerar embeddings : %v\n",err) 
        return nil,err
    }

    embeddings := pipelineResults.Embeddings

    session.Destroy()

    return embeddings, nil

} 