package api

import (
	// bibs padrão
	"context"
	"fmt"
	"net/http"
	"time"

	// bibs internas
	"projeto/api/routes/fileroute"
	"projeto/node"

	// bibs externas
	"github.com/gin-gonic/gin"
)


type APIInterface struct {
	NodeId string
}

func (api *APIInterface) Initialize(ctx context.Context, nd *node.Node, port int) error {

	gin.SetMode(gin.ReleaseMode)
	
	// Setup do gin http server
	app := gin.New()      
	app.Use(gin.Logger()) 
	app.Use(gin.Recovery())

	// Setup das rotas para gestão de ficheiros
	fileroute.SetUpRoutes(app.Group("/file"), nd)

	// Criar servidor HTTP
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: app,
	}

	// Canal para capturar erros do servidor
	errChan := make(chan error, 1)

	// Iniciar servidor numa goroutine
	go func() {
		fmt.Printf("[API] Servidor iniciado na porta %d\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("erro ao iniciar servidor: %v", err)
		}
	}()

	// Aguardar cancelamento do contexto ou erro
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		fmt.Printf("[API] Encerrando servidor gracefully...\n")
		
		// Contexto com timeout para shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("erro ao encerrar servidor: %v", err)
		}
		
		fmt.Printf("[API] Servidor encerrado\n")
		return nil
	}
}
