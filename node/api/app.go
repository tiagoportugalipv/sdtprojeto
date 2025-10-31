package api

import (
	"context"
	fileRoutes "sdt/node/api/routes/io"
	"sdt/node/services/messaging"

	"github.com/gin-gonic/gin"
	iface "github.com/ipfs/kubo/core/coreiface"
)

func Initialize( nodeCtx context.Context, ipfs iface.CoreAPI, pubSubService *messaging.PubSubService ){

  app := gin.New(); //a app está a criar instancia de um http server
  app.Use(gin.Logger()); //é um middleware que vai dar logs das mensagens
  app.Use(gin.Recovery()); 

  //Vai utilizar as rotas e vai implementar no grupo, passa o context, ipfs e pubsubservice
  fileRoutes.SetUpRoutes(app.Group("/file"), nodeCtx, ipfs, pubSubService);
  app.Run(":9000"); //A app vai correr na porta 9000
}

