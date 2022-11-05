package endpoints

import (
	"net/http"
	"recengine/internal/api/shard/dto"
	"recengine/internal/domain/services"
	"recengine/internal/domain/valueobjects"

	"github.com/gin-gonic/gin"
)

// Controller for the namespace API endpoint.
type NamespaceEndpoint struct {
	nsService *services.NamespaceService
}

// Creates a NamespaceEndpoint.
func NewNamespaceEndpoint(nsService *services.NamespaceService) *NamespaceEndpoint {
	return &NamespaceEndpoint{
		nsService: nsService,
	}
}

// Registers REST API endpoints on a router.
func (endpoint *NamespaceEndpoint) RegisterRoutes(router gin.IRouter) {
	router.GET("/api/v1/namespaces", func(ctx *gin.Context) {
		endpoint.List(ctx)
	})
	router.POST("/api/v1/namespaces", func(ctx *gin.Context) {
		endpoint.Create(ctx)
	})
	router.GET("/api/v1/namespaces/:domain", func(ctx *gin.Context) {
		endpoint.Get(ctx)
	})
	router.PUT("/api/v1/namespaces/:domain", func(ctx *gin.Context) {
		endpoint.Update(ctx)
	})
}

// @Summary      Creates a namespace.
// @Tags         Namespace
// @Accept       json
// @Produce      json
// @Param        body body dto.NamespaceCreateRequest true "NamespaceCreateRequest"
// @Success      200  {object}  dto.NamespaceResponse
// @Failure      400  {object}  dto.ValidationError
// @Router       /api/v1/namespaces [post]
func (endpoint *NamespaceEndpoint) Create(ctx *gin.Context) {
	var req dto.NamespaceCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		AbortWithBindingErrors(ctx, err)
		return
	}
	domainDto, err := req.ToDomain()
	if err != nil {
		AbortWithValidationError(ctx, err)
		return
	}
	ns, err := endpoint.nsService.CreateNamespace(domainDto)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, dto.FromError(err))
		return
	}
	ctx.IndentedJSON(http.StatusCreated, dto.NewNamespaceResponse(ns))
}

// @Summary      Returns a namespace by its name.
// @Tags         Namespace
// @Accept       json
// @Produce      json
// @Param        name path integer true "Namespace name"
// @Success      200  {object}  dto.NamespaceResponse
// @Failure      404  {object}  dto.Error
// @Failure      400  {object}  dto.Error
// @Router       /api/v1/namespaces/{name} [get]
func (endpoint *NamespaceEndpoint) Get(ctx *gin.Context) {
	name, err := valueobjects.ParseNamespaceName(ctx.Param("namespace"))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, dto.FromError(err))
		return
	}
	ns := endpoint.nsService.GetNamespaceByName(name)
	if ns == nil {
		ctx.IndentedJSON(http.StatusNotFound, dto.Error{Message: "namespace not found"})
		return
	}
	ctx.IndentedJSON(http.StatusOK, dto.NewNamespaceResponse(ns))
}

// @Summary      Returns all registered namespaces.
// @Tags         Namespace
// @Accept       json
// @Produce      json
// @Success      200  {array}   dto.NamespaceResponse
// @Router       /api/v1/namespaces [get]
func (endpoint *NamespaceEndpoint) List(ctx *gin.Context) {
	responses := dto.MakeNamespaceResponseArray(endpoint.nsService.GetNamespaces())
	ctx.IndentedJSON(http.StatusOK, responses)
}

// @Summary      Updates a namespaces.
// @Tags         Namespace
// @Accept       json
// @Produce      json
// @Param        name path integer true "Namespace name"
// @Param        body body dto.NamespaceUpdateRequest true "NamespaceUpdateRequest"
// @Success      200  {array}   dto.NamespaceResponse
// @Router       /api/v1/namespaces/{name} [put]
func (endpoint *NamespaceEndpoint) Update(ctx *gin.Context) {
	name, err := valueobjects.ParseNamespaceName(ctx.Param("namespace"))
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, dto.FromError(err))
		return
	}
	ns := endpoint.nsService.GetNamespaceByName(name)
	if ns == nil {
		ctx.IndentedJSON(http.StatusNotFound, dto.Error{Message: "namespace not found"})
		return
	}
	var req dto.NamespaceUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		AbortWithBindingErrors(ctx, err)
		return
	}
	domainDto, err := req.ToDomain()
	if err != nil {
		AbortWithValidationError(ctx, err)
		return
	}
	if ns.GetName().Value() != req.Name &&
		endpoint.nsService.GetNamespaceByName(domainDto.Name) != nil {
		ctx.IndentedJSON(http.StatusNotFound, dto.Error{Message: "namespace name taken"})
		return
	}
	ns, err = endpoint.nsService.UpdateNamespace(name, domainDto)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, dto.FromError(err))
		return
	}
	if err := endpoint.nsService.SaveNamespaces(); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, dto.FromError(err))
		return
	}
	ctx.IndentedJSON(http.StatusOK, dto.NewNamespaceResponse(ns))
}
