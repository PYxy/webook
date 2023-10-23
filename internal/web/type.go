package web

import "github.com/gin-gonic/gin"

type handler interface {
	RegisterPublicRoutes(server *gin.Engine)
	RegisterPrivateRoutes(server *gin.Engine)
}
