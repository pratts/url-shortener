package users

import (
	"shortener/models"
)

func ValidateUser(userName string, password string) (models.User, error) {
	var user models.User
	if err := models.DBObj.Where("user_name = ? AND password = ?", userName, password).First(&user).Error; err != nil {
		return models.User{}, err
	}
	return user, nil
}

func GetUserById(id string) (models.User, error) {
	var user models.User
	if err := models.DBObj.Where("id = ?", id).First(&user).Error; err != nil {
		return models.User{}, err
	}
	return user, nil
}

func CreateUser(user models.User) (models.User, error) {
	if err := models.DBObj.Create(&user).Error; err != nil {
		return models.User{}, err
	}
	return user, nil
}

func UpdateUser(user models.User) (models.User, error) {
	if err := models.DBObj.Save(&user).Error; err != nil {
		return models.User{}, err
	}
	return user, nil
}
