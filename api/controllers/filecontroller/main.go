package filecontroller

import (

	// bibs padrão
	"bufio"
	"io"
	"log"
	"net/http"
	"time"
        "strings"

	// bibs externas
	"github.com/gin-gonic/gin"

	// bibs internas
	"projeto/node"
	"projeto/services/embedding"
	"projeto/types"
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
                "message":  "File added successfully, CID : " + fileCid[6:],
                "filename": file.Filename,
                "cid":      fileCid[6:],
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

func GetFile(ctx *gin.Context, nd *node.Node) {

    cid := ctx.Query("cid")
    
    if cid == "" {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "cid parameter is required"})
        return
    }

    requestUUID, _ := nd.MakeRequest("/ipfs/"+cid, int(types.GETREQUEST))

    select {
    case <-nd.RequestQueue[requestUUID].Done:
        if nd.RequestQueue[requestUUID].Success {
            fileBytes, _ := nd.RequestQueue[requestUUID].Response.([]byte)
            
            // Send the file bytes as response
            ctx.Data(http.StatusOK, "application/octet-stream", fileBytes)
            
        } else {
            ctx.JSON(http.StatusInternalServerError, gin.H{
                "error": "Request failed",
            })
        }

    case <-time.After(2 * time.Minute):
        ctx.JSON(http.StatusRequestTimeout, gin.H{
            "error": "Request timeout after 2 minutes",
        })
    }

    delete(nd.RequestQueue, requestUUID)
}


type Prompt struct {
    Content string `json:"prompt" binding:"required"`
}

func GetCidFromPrompt(ctx *gin.Context, nd *node.Node) {

    var prt Prompt
    
    if err := ctx.ShouldBindJSON(&prt); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    requestUUID, err := nd.MakeRequest(prt.Content, int(types.PROMPTREQUEST))
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
        return
    }

    select {
    case <-nd.RequestQueue[requestUUID].Done:
        if nd.RequestQueue[requestUUID].Success {
            fileCid, ok := nd.RequestQueue[requestUUID].Response.(string)
            if !ok {
                ctx.JSON(http.StatusInternalServerError, gin.H{
                    "error": "Invalid response type",
                })
                delete(nd.RequestQueue, requestUUID)
                return
            }

            // Safely remove /ipfs/ prefix
            cleanCid := strings.TrimPrefix(fileCid, "/ipfs/")
            
            // Verify we got a valid CID after trimming
            if cleanCid == "" || cleanCid == fileCid {
                ctx.JSON(http.StatusInternalServerError, gin.H{
                    "error": "Invalid CID format",
                })
                delete(nd.RequestQueue, requestUUID)
                return
            }

            ctx.JSON(http.StatusOK, gin.H{
                "message": "File added successfully, CID : " + cleanCid,
                "cid":     cleanCid,
            })
            
        } else {
            ctx.JSON(http.StatusInternalServerError, gin.H{
                "error": "Request failed",
            })
        }

    case <-time.After(2 * time.Minute):
        ctx.JSON(http.StatusRequestTimeout, gin.H{
            "error": "Request timeout after 2 minutes",
        })
    }

    delete(nd.RequestQueue, requestUUID)
}


