package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	modelsdto "mishosapi/models/dto"
	"mishosapi/services"
)

type CalendarController struct{}

var calendarService = services.CalendarService{}

// GetCalendarToken returns (and lazily creates) the user's calendar
// subscription token so the frontend can build the subscription URL.
func (cc CalendarController) GetCalendarToken(context *gin.Context) {
	user, err := services.GetUserFromContext(context)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	token, err := calendarService.GetOrCreateToken(user.ID)
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{"token": token})
}

// GetCalendarIcs serves the public subscription feed at /api/calendar/{token}.ics.
// The token protects the feed; Apple Calendar / any CalDAV client can subscribe.
func (cc CalendarController) GetCalendarIcs(context *gin.Context) {
	// gin registers the segment ":token.ics"; the param name therefore
	// contains the dot. Accept both spellings so the token always resolves.
	token := strings.TrimSuffix(context.Param("token"), ".ics")
	if token == "" {
		token = strings.TrimSuffix(context.Param("token.ics"), ".ics")
	}

	if token == "" {
		context.AbortWithStatusJSON(http.StatusNotFound, modelsdto.ErrorDto{Error: "calendar feed not found"})
		return
	}

	user, err := services.GetUserByCalendarToken(token)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusNotFound, modelsdto.ErrorDto{Error: "calendar feed not found"})
		return
	}

	ics, err := calendarService.GetCalendarICS(user.ID)
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.Header("Content-Type", "text/calendar; charset=utf-8")
	context.Header("Cache-Control", "no-cache")
	context.String(http.StatusOK, "%s", ics)
}

// GetShowCalendarIcs downloads a single show's upcoming schedule as an .ics
// file, used by the "Add to calendar" button on the show details page.
func (cc CalendarController) GetShowCalendarIcs(context *gin.Context) {
	if _, err := services.GetUserFromContext(context); err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	showId, err := strconv.ParseUint(context.Param("showId"), 10, 32)
	if err != nil {
		services.SendError(context, "invalid show id")
		return
	}

	ics, err := calendarService.GetShowCalendarICS(uint(showId))
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.Header("Content-Type", "text/calendar; charset=utf-8")
	context.Header("Content-Disposition", "attachment; filename=mishos-show.ics")
	context.String(http.StatusOK, "%s", ics)
}
