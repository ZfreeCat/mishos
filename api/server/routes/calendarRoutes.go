package routes

import (
	"mishosapi/controllers"
	"mishosapi/middlewares"

	"github.com/gin-gonic/gin"
)

func AddCalendarRoutes(router *gin.Engine) {
	calendarController := controllers.CalendarController{}

	authRoutes := router.Group("/api/calendar", middlewares.AuthRequired())
	{
		authRoutes.GET("/token", calendarController.GetCalendarToken)
		authRoutes.GET("/show/:showId.ics", calendarController.GetShowCalendarIcs)
	}

	// public subscription feed, protected by the random token itself
	// (param is "token.ics"; the controller handles both spellings)
	router.GET("/api/calendar/:token.ics", calendarController.GetCalendarIcs)
}
