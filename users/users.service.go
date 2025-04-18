package users

import (
	"shortener/db"
	"shortener/models"
)

func ValidateUser(userName string, password string) (models.UserLoginResponseDto, error) {
	var user models.User
	if err := db.DBObj.Where("email = ? AND password = ?", userName, password).First(&user).Error; err != nil {
		return models.UserLoginResponseDto{}, err
	}

	userDto := models.UserLoginResponseDto{
		Id:    user.Id,
		Email: user.Email,
		Name:  user.Name,
	}
	return userDto, nil
}

func GetUserById(id uint64) (models.UserDto, error) {
	var user models.User
	if err := db.DBObj.Where("id = ?", id).First(&user).Error; err != nil {
		return models.UserDto{}, err
	}
	userDto := models.UserDto{
		Id:       user.Id,
		Email:    user.Email,
		Verified: user.Verified,
		Name:     user.Name,
	}
	return userDto, nil
}

func CreateUser(dto models.UserCreateDto) (models.UserDto, error) {
	user := models.User{
		Email:    dto.Email,
		Password: dto.Password,
		Name:     dto.Name,
		Verified: false,
	}
	if err := db.DBObj.Create(&user).Error; err != nil {
		return models.UserDto{}, err
	}
	userDto := models.UserDto{
		Id:       user.Id,
		Email:    user.Email,
		Verified: user.Verified,
		Name:     user.Name,
	}
	return userDto, nil
}

func UpdateUser(id uint64, update models.UserUpdateDto) (models.UserDto, error) {
	res := db.DBObj.Model(&models.User{}).Where("id=?", id).Updates(update)
	if res.Error != nil {
		return models.UserDto{}, res.Error
	}

	return GetUserById(id)
}
