package embedding

import (
	"fmt"

	"github.com/knights-analytics/hugot"
)

var ModelDirPath string
var ModelPath string

func SetUpModel() (error,string){

    var err error
    
    downloadOptions := hugot.NewDownloadOptions()
    downloadOptions.OnnxFilePath = "onnx/model.onnx"

    ModelPath,err = hugot.DownloadModel(
        "sentence-transformers/all-MiniLM-L6-v2",
        ModelDirPath,
        downloadOptions,
    ) 

    return err,ModelPath

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


