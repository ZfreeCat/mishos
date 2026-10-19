package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	modelsdto "mishosapi/models/dto"
	"mishosapi/services"
)

type TmdbController struct{}

var tmdbService = services.TmdbService{}

// ImportShow pulls a movie/tv show into the local library without following it.
func (tc TmdbController) ImportShow(context *gin.Context) {
	if _, err := services.GetUserFromContext(context); err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	var body struct {
		TmdbId    int    `json:"tmdbId" binding:"required"`
		MediaType string `json:"mediaType" binding:"required"`
	}

	if context.BindJSON(&body) != nil {
		services.SendError(context, "tmdbId and mediaType are required")
		return
	}

	showId, err := tmdbService.ImportShow(body.TmdbId, body.MediaType)
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{"showId": showId})
}

// CreateAuthToken starts the TMDB v3 auth flow (step 1).
func (tc TmdbController) CreateAuthToken(context *gin.Context) {
	_, err := services.GetUserFromContext(context)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	requestToken, authUrl, err := tmdbService.CreateRequestToken()
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{"requestToken": requestToken, "authUrl": authUrl})
}

// CompleteAuth finishes the TMDB v3 auth flow (step 3).
func (tc TmdbController) CompleteAuth(context *gin.Context) {
	user, err := services.GetUserFromContext(context)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	var body struct {
		RequestToken string `json:"requestToken" binding:"required"`
	}

	if context.BindJSON(&body) != nil {
		services.SendError(context, "bad request")
		return
	}

	if err := tmdbService.CompleteAuth(user.ID, body.RequestToken); err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "tmdb account connected"})
}

// Disconnect removes the TMDB account binding.
func (tc TmdbController) Disconnect(context *gin.Context) {
	user, err := services.GetUserFromContext(context)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	if err := tmdbService.Disconnect(user.ID); err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "tmdb account disconnected"})
}

// Sync pulls the TMDB watchlist and follows every entry in Mishos.
func (tc TmdbController) Sync(context *gin.Context) {
	user, err := services.GetUserFromContext(context)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	added, followed, removed, err := tmdbService.SyncWatchlist(user.ID)
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{"added": added, "followed": followed, "removed": removed})
}

// Status returns the TMDB binding status.
func (tc TmdbController) Status(context *gin.Context) {
	user, err := services.GetUserFromContext(context)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	connected, username, lastSyncAt, err := tmdbService.GetTmdbStatus(user.ID)
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{
		"connected":  connected,
		"username":   username,
		"lastSyncAt": lastSyncAt,
	})
}

// Ratings returns the user's TMDB rated entries (My Ratings).
// mediaType query param: all | tv | movie (default all).
func (tc TmdbController) Ratings(context *gin.Context) {
	user, err := services.GetUserFromContext(context)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	mediaType := context.Query("mediaType")
	results, err := tmdbService.GetMyRatings(user.ID, mediaType)
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{"results": results})
}

// Recommendations returns new-release recommendations (upcoming/new/trending).
func (tc TmdbController) Recommendations(context *gin.Context) {
	user, err := services.GetUserFromContext(context)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	category := context.Query("category")
	results, err := tmdbService.GetRecommendations(user.ID, category)
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{"results": results})
}

// FollowRecommendation adds a TMDB media item to Mishos and follows it.
func (tc TmdbController) FollowRecommendation(context *gin.Context) {
	user, err := services.GetUserFromContext(context)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusUnauthorized, modelsdto.ErrorDto{Error: err.Error()})
		return
	}

	var body struct {
		TmdbId    int    `json:"tmdbId" binding:"required"`
		MediaType string `json:"mediaType" binding:"required"`
	}

	if context.BindJSON(&body) != nil {
		services.SendError(context, "bad request")
		return
	}

	added, followed, err := tmdbService.EnsureShowFollowed(user.ID, body.TmdbId, body.MediaType)
	if err != nil {
		services.SendError(context, err.Error())
		return
	}

	context.JSON(http.StatusOK, gin.H{"added": added, "followed": followed})
}
