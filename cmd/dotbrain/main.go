package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{
			"message": "pong",
		})
	})
	err := router.Run()
	if err != nil {
		log.Fatal("failed to start server: ", err)
	}
	log.Println("hello world")
}
