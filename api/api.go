package api

import (
	// bibs padrão
	"fmt"

	// bibs internas
	"projeto/node"
	"projeto/api/routes/fileroute"

	// bibs externas
	"github.com/gin-gonic/gin"
)


type APIInterface struct {
	NodeId string
}

func (api *APIInterface) Initialize(nd *node.Node,port int) (error){

	gin.SetMode(gin.ReleaseMode)
	

	// Setup do gin http server
	app := gin.New()      
	app.Use(gin.Logger()) 
	app.Use(gin.Recovery())

	// Setup das rotas para gestão de ficheiros
	fileroute.SetUpRoutes(app.Group("/file"), nd)

	return app.Run(fmt.Sprintf(":%d",port))


}
