package fileroute

import (

	// bibs internas
	"projeto/node"
	"projeto/api/controllers/filecontroller"

	// bibs externas
	"github.com/gin-gonic/gin"
)

// Configura as rotas
func SetUpRoutes(rg *gin.RouterGroup, nd *node.Node ) {

	//Um cliente envia uma requisição POST para /upload e o Gin corre a função passada
	rg.POST("/upload", func(ctx *gin.Context) {
		filecontroller.UploadFile(ctx, nd)
	})


	rg.GET("/retreive", func(ctx *gin.Context) {
		filecontroller.GetFile(ctx, nd)
	})


	rg.GET("/prompt", func(ctx *gin.Context) {
		filecontroller.GetCidFromPrompt(ctx, nd)
	})

}
