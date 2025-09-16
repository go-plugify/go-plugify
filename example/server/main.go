package main

import (
	goportal "github.com/chenhg5/go-portal"
	"github.com/gin-gonic/gin"
)

func main() {
	// start a gin server, and return hello world at /
	router := setupRouter()
	router.Run(":8080")
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})
	goportal.InitGin(r, "/api/v1")
	return r
}