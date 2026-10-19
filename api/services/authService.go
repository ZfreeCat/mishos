package services

import (
	"errors"

	"mishosapi/db"
	modelsdb "mishosapi/models/db"
	modelsdto "mishosapi/models/dto"

	"github.com/gin-gonic/gin"
)

func SanitizeUser(user modelsdb.User) modelsdto.UserDto {
	var sanitizedUser modelsdto.UserDto

	sanitizedUser.ID = user.ID
	sanitizedUser.Username = user.Username
	sanitizedUser.Role = user.UserRole.Name

	return sanitizedUser
}

func GetUserFromContext(context *gin.Context) (*modelsdto.UserDto, error) {
	userFromContext, exists := context.Get("user")
	if !exists {
		return nil, errors.New("context: user not found")
	}

	user, ok := userFromContext.(modelsdto.UserDto)
	if !ok {
		return nil, errors.New("context: invalid user")
	}

	return &user, nil
}

// GetUserByCalendarToken resolves a user from their calendar subscription
// token (used by the public ICS feed endpoint).
func GetUserByCalendarToken(token string) (*modelsdto.UserDto, error) {
	if token == "" {
		return nil, errors.New("calendar: empty token")
	}

	var user modelsdb.User
	err := db.DB.
		Preload("UserRole").
		Joins("JOIN user_configs ON user_configs.user_id = users.id").
		Where("user_configs.calendar_token = ?", token).
		First(&user).Error
	if err != nil {
		return nil, err
	}

	sanitized := SanitizeUser(user)
	return &sanitized, nil
}
