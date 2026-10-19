package routes

import (
	"mishosapi/controllers"
	"mishosapi/middlewares"

	"github.com/gin-gonic/gin"
)

func AddTmdbRoutes(router *gin.Engine) {
	tmdbController := controllers.TmdbController{}

	authRoutes := router.Group("/api/tmdb", middlewares.AuthRequired())
	{
		authRoutes.POST("/auth/token", tmdbController.CreateAuthToken)
		authRoutes.POST("/auth/session", tmdbController.CompleteAuth)
		authRoutes.POST("/disconnect", tmdbController.Disconnect)
		authRoutes.POST("/sync", tmdbController.Sync)
		authRoutes.POST("/follow", tmdbController.FollowRecommendation)
		authRoutes.GET("/status", tmdbController.Status)
		authRoutes.GET("/ratings", tmdbController.Ratings)
	}

	authRoutes = router.Group("/api/recommendations", middlewares.AuthRequired())
	{
		authRoutes.GET("", tmdbController.Recommendations)
	}
}
