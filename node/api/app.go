package api

import (
	"context"
	fileRoutes "sdt/node/api/routes/io"
	"github.com/gin-gonic/gin"
	iface "github.com/ipfs/kubo/core/coreiface"
)

func Initialize( nodeCtx context.Context, ipfs iface.CoreAPI ){


  app := gin.New();
  app.Use(gin.Logger());
  app.Use(gin.Recovery());


  fileRoutes.SetUpRoutes(app.Group("/file"), nodeCtx, ipfs);
  app.Run(":9000");

}

