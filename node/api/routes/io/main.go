package io

import (
 "context"
 "github.com/gin-gonic/gin"
 fileController "sdt/node/api/controllers/io"
 iface "github.com/ipfs/kubo/core/coreiface"
 "sdt/node/services/messaging"
)

//Configura as rotas
func SetUpRoutes(rg *gin.RouterGroup, nodeCtx context.Context, ipfs iface.CoreAPI, pubSubService *messaging.PubSubService){

 //Um cliente envia uma requisição POST para /upload e o Gin corre a função passada
  rg.POST("/upload", func(c *gin.Context) {
    fileController.UploadFile(c,nodeCtx,ipfs,pubSubService)
  })


}
