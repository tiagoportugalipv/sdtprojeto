package io

import (

 "context"
 "github.com/gin-gonic/gin"
 fileController "sdt/node/api/controllers/io"
 iface "github.com/ipfs/kubo/core/coreiface"
 
)

func SetUpRoutes(rg *gin.RouterGroup, nodeCtx context.Context, ipfs iface.CoreAPI){

  rg.POST("/upload", func(c *gin.Context) {
    fileController.UploadFile(c,nodeCtx,ipfs)
  })


}
