package io

import (
 "context"
 "github.com/gin-gonic/gin"
 fileController "sdt/node/api/controllers/io"
 iface "github.com/ipfs/kubo/core/coreiface"
 "sdt/node/services/messaging"
)

func SetUpRoutes(rg *gin.RouterGroup, nodeCtx context.Context, ipfs iface.CoreAPI, pubSubService *messaging.PubSubService){

  rg.POST("/upload", func(c *gin.Context) {
    fileController.UploadFile(c,nodeCtx,ipfs,pubSubService)
  })


}
