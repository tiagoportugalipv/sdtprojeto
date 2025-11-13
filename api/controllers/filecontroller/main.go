package filecontroller

import (

	// bibs padrão
	"bufio"
	"io"
	"log"
	"net/http"

	// bibs externas
	"github.com/gin-gonic/gin"

	// bibs internas
	"projeto/node"
	"projeto/services/embedding"
	"projeto/services/messaging"
)

func UploadFile(ctx *gin.Context, nd *node.Node) {

    // if(nd.State != node.LEADER){
    //     ctx.JSON(http.StatusBadRequest, gin.H{"error": "Nó não lider"})
    //     return
    // }

    file, err := ctx.FormFile("file")
    if err != nil || file == nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "Nenhum ficheiro foi anexado ao request"})
        return
    }

    uploadedFile, err := file.Open()

    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao abrir ficheiro"})
        return
    }


    fileBytes, err := io.ReadAll(uploadedFile)


    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao ler ficheiro"})
        return
    }

    // Geração de embeddings //

    uploadedFile.Seek(0, io.SeekStart)

    fileLines := []string{}
    scanner := bufio.NewScanner(uploadedFile)


    for scanner.Scan() {
         line := scanner.Text() 
         fileLines = append(fileLines,line) 
    }


    embs,err := embedding.GetEmbeddings(fileLines)



    if err != nil {
        log.Printf("Falha ao criar embeddings para ficheiro: %v",err) 
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao criar embeddings para o ficheiro"})
        return
    }


    uploadedFile.Close()

    fileCid, err := nd.AddFile(fileBytes)

    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao submeter ficheiro para o ipfs"})
        return
    }

    currentVector := nd.CidVector
    newVector := node.Vector {
         Ver: currentVector.Ver + 1,
         Content: append(currentVector.Content,fileCid.String()),
    }

    nd.CidVectorStaging = newVector

    msg := messaging.AppendEntryMessage{
            Vector: newVector,
            Embeddings: embs,
    }


    err = messaging.PublishTo(nd.IpfsApi.PubSub(),messaging.AEM,msg)

    if(err != nil){
        log.Printf("Falha ao enviar estrutura: %v",err) 
    }

    ctx.JSON(http.StatusOK, gin.H{
        "message": "File added successfully, CID : "+fileCid.String(),
        "filename": file.Filename,
        "cid": fileCid.String(),
        "vetorHash": nd.CidVectorStaging.Hash(),
        "vetor enviado" : nd.CidVectorStaging.String(),
    })
}
