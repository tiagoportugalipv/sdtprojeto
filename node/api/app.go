package api

import (
	"context"
	fileRoutes "sdt/node/api/routes/io"
	"github.com/gin-gonic/gin"
	iface "github.com/ipfs/kubo/core/coreiface"
  "sdt/node/services/messaging"
)

func Initialize( nodeCtx context.Context, ipfs iface.CoreAPI, pubSubService *messaging.PubSubService ){

  app := gin.New();
  app.Use(gin.Logger());
  app.Use(gin.Recovery());

  fileRoutes.SetUpRoutes(app.Group("/file"), nodeCtx, ipfs, pubSubService);
  app.Run(":9000");
}

