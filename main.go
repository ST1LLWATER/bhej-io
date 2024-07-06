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
	r.GET("/room/:id", handlers.RoomHandler)
	r.GET("/ws/:id", handlers.WebsocketHandler)
	r.GET("/room/:id/users", handlers.UsersHandler)

	r.Run(":3000")
}
