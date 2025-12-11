package filecontroller

import (

	// bibs padrão
	"bufio"
	"io"
	"log"
	"net/http"
	"strings"
        "time"

	// bibs externas
	"github.com/gin-gonic/gin"

	// bibs internas
	"projeto/node"
	"projeto/services/embedding"
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

    fileContent := strings.Builder{}

    scanner := bufio.NewScanner(uploadedFile)


    for scanner.Scan() {
         line := scanner.Text() 
         fileContent.WriteString(line)
    }


    embs,err := embedding.GetEmbeddings(fileContent.String())



    if err != nil {
        log.Printf("Falha ao criar embeddings para ficheiro: %v",err) 
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao criar embeddings para o ficheiro"})
        return
    }


    uploadedFile.Close()

    requestUUID, err := nd.AddFile(fileBytes,embs)


    select {
    case <-nd.RequestQueue[requestUUID].Done:
        // Request completou com sucesso
        var fileCid string

        if nd.RequestQueue[requestUUID].Success {
            fileCid, _ = nd.RequestQueue[requestUUID].Response.(string)

            ctx.JSON(http.StatusOK, gin.H{
                "message":  "File added successfully, CID : " + fileCid,
                "filename": file.Filename,
                "cid":      fileCid,
            })
        } else {
            // Tratamento de falha
            ctx.JSON(http.StatusInternalServerError, gin.H{
                "error": "Request failed",
            })
        }


    case <-time.After(2 * time.Minute):
        // Timeout após 2 minutos
        ctx.JSON(http.StatusRequestTimeout, gin.H{
            "error": "Request timeout after 2 minutes",
        })
        
    }


    delete(nd.RequestQueue, requestUUID)




}
