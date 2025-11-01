package io

import (
	"context"
	fileController "sdt/node/api/controllers/io"
	"sdt/node/services/messaging"

	"github.com/gin-gonic/gin"
	iface "github.com/ipfs/kubo/core/coreiface"
)

// Configura as rotas
func SetUpRoutes(rg *gin.RouterGroup, nodeCtx context.Context, ipfs iface.CoreAPI, pubSubService *messaging.PubSubService, cidVector []string) {

	//Um cliente envia uma requisição POST para /upload e o Gin corre a função passada
	rg.POST("/upload", func(c *gin.Context) {
		fileController.UploadFile(c, nodeCtx, ipfs, pubSubService, cidVector)
	})

}
