package shard

import (
	"net/http"
	"recengine/internal/api/common"
	"recengine/internal/entities"
	"recengine/internal/valueobjects"

	"github.com/gin-gonic/gin"
)

// Controller for the namespace API endpoint.
type NamespaceEndpoint struct {
	nsService *entities.NamespaceService
}

// Creates a NamespaceEndpoint.
func NewNamespaceEndpoint(nsService *entities.NamespaceService) *NamespaceEndpoint {
	return &NamespaceEndpoint{
		nsService: nsService,
	}
}

// Registers REST API endpoints on a router.
func (endpoint *NamespaceEndpoint) RegisterRoutes(router gin.IRouter) {
	router.GET("/namespaces", func(ctx *gin.Context) {
		endpoint.List(ctx)
	})
	router.POST("/namespaces", func(ctx *gin.Context) {
		endpoint.Create(ctx)
	})
	router.GET("/namespaces/:domain", func(ctx *gin.Context) {
		endpoint.Get(ctx)
	})
	router.PUT("/namespaces/:domain", func(ctx *gin.Context) {
		endpoint.Update(ctx)
	})
}

// HTTP handler for POST /namespaces
func (endpoint *NamespaceEndpoint) Create(ctx *gin.Context) {
	var dto common.NamespaceCreateDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		common.AbortWithBindingErrors(ctx, err)
		return
	}
	domainDto, err := dto.ToDomain()
	if err != nil {
		common.AbortWithValidationError(ctx, err)
		return
	}
	domain, err := endpoint.nsService.CreateNamespace(domainDto)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err})
		return
	}
	ctx.IndentedJSON(http.StatusCreated, domain)
}

// HTTP handler for GET /namespaces/:namespace
func (endpoint *NamespaceEndpoint) Get(ctx *gin.Context) {
	name, err := valueobjects.ParseNamespaceName(ctx.Param("namespace"))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err})
		return
	}
	ns := endpoint.nsService.GetNamespaceByName(name)
	if ns == nil {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "namespace not found"})
		return
	}
	ctx.IndentedJSON(http.StatusOK, common.NewNamespaceResponse(ns))
}

// HTTP handler for GET /namespaces
func (endpoint *NamespaceEndpoint) List(ctx *gin.Context) {
	responses := common.MakeNamespaceResponseArray(endpoint.nsService.GetNamespaces())
	ctx.IndentedJSON(http.StatusOK, responses)
}

// HTTP handler for PUT /namespaces/:namespace
func (endpoint *NamespaceEndpoint) Update(ctx *gin.Context) {
	name, err := valueobjects.ParseNamespaceName(ctx.Param("namespace"))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err})
		return
	}
	ns := endpoint.nsService.GetNamespaceByName(name)
	if ns == nil {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "namespace not found"})
		return
	}
	var dto common.NamespaceUpdateDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		common.AbortWithBindingErrors(ctx, err)
		return
	}
	domainDto, err := dto.ToDomain()
	if err != nil {
		common.AbortWithValidationError(ctx, err)
		return
	}
	if ns.GetName().Value() != dto.Name &&
		endpoint.nsService.GetNamespaceByName(domainDto.Name) != nil {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "namespace name taken"})
		return
	}
	ns, err = endpoint.nsService.UpdateNamespace(name, domainDto)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": err})
		return
	}
	if err := endpoint.nsService.SaveNamespaces(); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	ctx.IndentedJSON(http.StatusOK, common.NewNamespaceResponse(ns))
}
