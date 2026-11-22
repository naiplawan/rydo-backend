package main

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files/v2"
)

//go:embed swagger/index.html
var swaggerPage []byte

//go:embed swagger/openapi.json
var openAPIDocument []byte

func registerSwagger(router *gin.Engine) {
	docs := router.Group("/swagger")
	docs.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "index.html")
	})
	docs.GET("/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", swaggerPage)
	})
	docs.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", openAPIDocument)
	})

	assets := http.FS(swaggerFiles.FS)
	docs.StaticFileFS("/swagger-ui.css", "swagger-ui.css", assets)
	docs.StaticFileFS("/swagger-ui-bundle.js", "swagger-ui-bundle.js", assets)
}
