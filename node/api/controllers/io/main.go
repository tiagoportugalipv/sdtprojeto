package controllers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"sdt/node/services/messaging"
	"sdt/node/types"

	"github.com/gin-gonic/gin"
	"github.com/ipfs/boxo/files"
	iface "github.com/ipfs/kubo/core/coreiface"
	"github.com/knights-analytics/hugot"
)

// Usamos interface{} porque os tipos exatos do hugot podem variar entre versões
var embeddingPipeline interface{}
var hugotSession interface{}
var pipelineReady = false

// InitEmbeddings deve ser chamada uma única vez no startup
func InitEmbeddings() error {
	if pipelineReady {
		return nil
	}

	session, err := hugot.NewGoSession()
	if err != nil {
		return err
	}
	// NÃO fazer defer - precisamos manter a sessão viva
	hugotSession = session

	downloadOptions := hugot.NewDownloadOptions()
	downloadOptions.OnnxFilePath = "onnx/model.onnx"

	modelPath, err := hugot.DownloadModel(
		"sentence-transformers/all-MiniLM-L6-v2",
		"./models/",
		downloadOptions,
	)
	if err != nil {
		return err
	}

	config := hugot.FeatureExtractionConfig{
		ModelPath: modelPath,
		Name:      "embeddingPipeline",
	}

	pipeline, err := hugot.NewPipeline(session, config)
	if err != nil {
		return err
	}

	embeddingPipeline = pipeline
	pipelineReady = true
	return nil
}

// generateEmbeddings simples - retorna []float32 (média de todos os embeddings)
func generateEmbeddings(fileBytes []byte) ([]float32, error) {
	if !pipelineReady {
		return nil, fmt.Errorf("embeddings not initialized")
	}

	reader := bytes.NewReader(fileBytes)
	scanner := bufio.NewScanner(reader)

	var fileLines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			fileLines = append(fileLines, line)
		}
	}

	if len(fileLines) == 0 {
		return []float32{}, nil
	}

	// Type assertion para chamar RunPipeline
	type PipelineRunner interface {
		RunPipeline([]string) (struct{ Embeddings [][]float32 }, error)
	}

	pipeline, ok := embeddingPipeline.(PipelineRunner)
	if !ok {
		return nil, fmt.Errorf("pipeline type assertion failed")
	}

	result, err := pipeline.RunPipeline(fileLines)
	if err != nil {
		return nil, err
	}

	// Retorna o primeiro embedding ou faz média
	if len(result.Embeddings) == 0 {
		return []float32{}, nil
	}

	// Retornar apenas o primeiro embedding
	return result.Embeddings[0], nil
}

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

	// Na função UploadFile, após adicionar ao IPFS
	newCID := peerCidFile.String()
	cidVector = append(cidVector, newCID)

	// Agora gera os embeddings
	embeddings, err := generateEmbeddings(fileBytes) // chama a função do Hugot
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate embeddings"})
		return
	}

	// Cria a estrutura DocumentUpdate
	update := types.DocumentUpdate{
		VectorVersion: len(cidVector),
		NewCID:        newCID,
		Embeddings:    [][]float32{embeddings},
		UpdatedVector: cidVector,
		Timestamp:     time.Now().Unix(),
	}

	// Publica via PubSub
	jsonUpdate, _ := update.ToJSON()
	if err := pubSubService.PublishMessage(jsonUpdate); err != nil {
		log.Printf("Failed to publish update: %v", err)
	} else {
		log.Printf("Published vector update version %d", len(cidVector))
	}

	//Responde ao cliente HTTP depois de o ficheiro ser enviado com sucesso para o IPFS
	//gin.H -> Cria o ficheiro JSON
	ctx.JSON(http.StatusOK, gin.H{
		"message":  "File added successfully, CID : " + peerCidFile.String(),
		"filename": file.Filename,
		"cid":      peerCidFile.String(), //Identificação única do arquivo
	})
}
