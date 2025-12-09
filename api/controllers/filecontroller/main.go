package filecontroller

import (

	// bibs padrão
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	// bibs externas
	"github.com/gin-gonic/gin"

	// bibs internas
	"projeto/node"
	"projeto/services/embedding"
	"projeto/services/messaging"
	// "projeto/types"
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

    fileCid, err := nd.AddFile(fileBytes)

    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao submeter ficheiro para o ipfs"})
        return
    }

    currentVector := nd.CidVector
    newVersion := nd.CidVector.Ver;

    for k := range(nd.VectorCache){
        if(k > newVersion){
            newVersion = k
        }
    }

    newVersion = newVersion + 1 
    fmt.Printf("Nova versão : %v \n",newVersion)

    newVector := node.Vector {
         Ver: newVersion,
         Content: append(currentVector.Content,fileCid.String()),
    }

    nd.VectorCache[newVersion] = newVector

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
    })
}
