package main

import (
	"bhej-io/handlers"
	"log"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	tmplPath, err := filepath.Abs("web/templates/*")
	if err != nil {
		log.Fatalf("Error finding template path: %v", err)
	}
	r.LoadHTMLGlob(tmplPath)

	r.GET("/", handlers.IndexHandler)
	r.GET("/ws", handlers.WebsocketHandler)
	r.GET("/users", handlers.UsersHandler)

	r.Run(":8080")
}
