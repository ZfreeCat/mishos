package routes

import (
	"mishosapi/controllers"

	"github.com/gin-gonic/gin"
)

func AddImageRoutes(router *gin.Engine) {
	imageController := controllers.ImageController{}

	// unauthenticated: <img> tags cannot send Authorization headers
	router.GET("/api/images/:name", imageController.GetImage)
}
