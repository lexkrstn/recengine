package shard

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"recengine/internal/entities"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Shard application instantiation parameters.
type ApplicationDto struct {
	Config    *Config
	NsService *entities.NamespaceService
}

// Shard application.
type Application struct {
	router     *gin.Engine
	httpSrv    *http.Server
	config     *Config
	nsEndpoint *NamespaceEndpoint
}

// Instantiates a new Application.
func NewApplication(dto *ApplicationDto) *Application {
	engine := gin.Default()
	httpSrv := &http.Server{
		Addr:    dto.Config.GetHostPort(),
		Handler: engine,
	}
	app := &Application{
		router:     engine,
		httpSrv:    httpSrv,
		config:     dto.Config,
		nsEndpoint: NewNamespaceEndpoint(dto.NsService),
	}
	app.nsEndpoint.RegisterRoutes(engine)
	return app
}

// Starts the HTTP server in a dedicated Go routine and blocks current thread
// execution until either an error occurs or the OS sends a signal to
// terminate current process.
func (srv *Application) Run() error {
	listenError := make(chan error)

	go func() {
		if err := srv.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			listenError <- fmt.Errorf("Server listening failed: %v", err)
		} else {
			listenError <- nil
		}
		log.Println("Server stopped listening")
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("Server shutdown: %v", err)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("Shutdown timeout")
	case err := <-listenError:
		log.Println("Server exiting")
		return err
	}
}
