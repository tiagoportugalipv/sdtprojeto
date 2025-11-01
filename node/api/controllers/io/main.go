package controllers

import (
	"context"
	"io"  //ler o conteudo do arquivo
	"log" //Para registar logs
	"net/http"
	"time" //definir tempo limite

	"sdt/node/services/messaging"

	"github.com/gin-gonic/gin"
	"github.com/ipfs/boxo/files"
	iface "github.com/ipfs/kubo/core/coreiface"
)

// UploadFile -> Função que é chamada quando um cliente faz upload do arquivo
// nodeCtx -> contexto do nó, ipfs->interface, pubSubService -> serviço que publica mensagens
func UploadFile(ctx *gin.Context, nodeCtx context.Context, ipfs iface.CoreAPI, pubSubService *messaging.PubSubService, cidVector []string) {

	//Recebe o arquivo
	file, err := ctx.FormFile("file")
	if err != nil || file == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	openedFile, err := file.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer openedFile.Close()

	fileBytes, err := io.ReadAll(openedFile)

	//Se existir erro é enviado um json a dizer que nao é possível abrir o ficheiro
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	//Cria um contexto que expira em 30 s — se o upload demorar mais, é cancelado.
	uploadCtx, cancel := context.WithTimeout(nodeCtx, 30*time.Second)
	defer cancel()

	//Envia o arquivo para o nó IPFS
	peerCidFile, err := ipfs.Unixfs().Add(uploadCtx, files.NewBytesFile(fileBytes))
	if err != nil { //Verifica se existe um erro ao adicionar o arquivo ao IPFS / nil == null
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add file to IPFS: " + err.Error()})
		return
	}
	//Mensagem para comunicar aos outros sistemas que um novo arquivo foi adicionado ao IPFS
	//criar o conteúdo da mensagem que será publicada num canal de Pub/Sub
	//message := "New file uploaded: " + file.Filename + " (CID: " + peerCidFile.String() + ")"

	cidVector = append(cidVector, peerCidFile.String())

	//PublishMessage -> tenta enviar a mensagem para o sistema de Pub/Sub
	if err := pubSubService.PublishMessage(cidVector); err != nil {
		log.Printf("Failed to publish upload notification: %v", err)
	} else {
		log.Printf("Published upload notice to topic")
	}

	//Responde ao cliente HTTP depois de o ficheiro ser enviado com sucesso para o IPFS
	//gin.H -> Cria o ficheiro JSON
	ctx.JSON(http.StatusOK, gin.H{
		"message":  "File added successfully, CID : " + peerCidFile.String(),
		"filename": file.Filename,
		"cid":      peerCidFile.String(), //Identificação única do arquivo
	})
}
