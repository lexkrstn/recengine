package recengine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const defaultServerHost = "localhost"
const defaultServerPort = 8080

type ServerConfig struct {
	Host string
	Port int
}

func MakeServerConfigFromEnv(defaults *ServerConfig) ServerConfig {
	host := os.Getenv("HOST")
	if host == "" {
		if defaults != nil {
			host = defaults.Host
		} else {
			host = defaultServerHost
		}
	}
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil || port < 0 {
		if defaults != nil {
			port = defaults.Port
		} else {
			port = defaultServerPort
		}
	}
	return ServerConfig{
		Host: host,
		Port: port,
	}
}

type Server struct {
	engine  *Engine
	router  *gin.Engine
	httpSrv *http.Server
	config  *ServerConfig
}

func NewServer(engine *Engine, config ServerConfig) *Server {
	router := gin.Default()
	httpSrv := &http.Server{
		Addr:    config.Host + ":" + strconv.Itoa(config.Port),
		Handler: router,
	}
	instance := &Server{engine, router, httpSrv, &config}

	router.GET("/domains", func(ctx *gin.Context) {
		instance.getDomains(ctx)
	})
	router.POST("/domains", func(ctx *gin.Context) {
		instance.postDomain(ctx)
	})
	router.GET("/domains/:domain", func(ctx *gin.Context) {
		instance.getDomain(ctx)
	})
	router.PUT("/domains/:domain", func(ctx *gin.Context) {
		instance.putDomain(ctx)
	})
	// router.DELETE("/domains/:domain", postDomain)
	// router.DELETE("/domains/:domain/users/:user", getDomainByName)
	// router.GET("/domains/:domain/users/:user", getDomainByName)
	// router.PUT("/domains/:domain/users/:user/likes", getDomainByName)
	// router.PUT("/domains/:domain/users/:user/dislikes", getDomainByName)
	// router.DELETE("/domains/:domain/users/:user/items/:item", getDomainByName)
	// router.GET("/domains/:domain/users/:user/similar-users", getDomainByName)
	// router.GET("/domains/:domain/users/:user/recommendations", getDomainByName)

	return instance
}

// Starts the HTTP server in a dedicated Go routine and blocks current thread
// execution until either an error occurs or the OS sends a signal to
// terminate current process.
func (srv *Server) Run() error {
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

// HTTP handler for GET /domains
func (srv *Server) getDomains(ctx *gin.Context) {
	ctx.IndentedJSON(http.StatusOK, srv.engine.domains)
}

// HTTP handler for POST /domains
func (srv *Server) postDomain(ctx *gin.Context) {
	var dto DomainCreateInput
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		abortWithBindingErrors(ctx, err)
		return
	}
	domain, err := srv.engine.AddDomain(&dto)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	ctx.IndentedJSON(http.StatusCreated, domain)
}

// HTTP handler for GET /domains/:domain
func (srv *Server) getDomain(ctx *gin.Context) {
	name := ctx.Param("domain")
	domain := srv.engine.GetDomainByName(name)
	if domain == nil {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "domain not found"})
		return
	}
	ctx.IndentedJSON(http.StatusOK, domain)
}

// HTTP handler for PUT /domains/:domain
func (srv *Server) putDomain(ctx *gin.Context) {
	name := ctx.Param("domain")
	domain := srv.engine.GetDomainByName(name)
	if domain == nil {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "domain not found"})
		return
	}
	var dto DomainUpdateInput
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		abortWithBindingErrors(ctx, err)
		return
	}
	if domain.GetName() != dto.Name && srv.engine.GetDomainByName(dto.Name) != nil {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "domain name taken"})
		return
	}
	if _, err := srv.engine.UpdateDomain(name, &dto); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	if err := srv.engine.SaveDomains(); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	ctx.IndentedJSON(http.StatusOK, domain)
}

// JSON format of validation errors.
type FieldErrorMsg struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Converts validator's FieldError to a message string.
func getFieldErrorMsg(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "This field is required"
	case "lte":
		return "Should be less than " + fieldError.Param()
	case "gte":
		return "Should be greater than " + fieldError.Param()
	}
	return "Unknown error"
}

// Aborts gin handler execution and sends an HTTP response containing the error
// description in JSON format.
func abortWithBindingErrors(ctx *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		errors := make([]FieldErrorMsg, len(ve))
		for i, fe := range ve {
			errors[i] = FieldErrorMsg{fe.Field(), getFieldErrorMsg(fe)}
		}
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error":  "validation",
			"errors": errors,
		})
		return
	}
	ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"message": err.Error(),
	})
}
