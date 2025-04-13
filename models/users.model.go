package models

import "time"

type User struct {
	Id        uint64 `gorm:"primaryKey autoIncrement"`
	Email     string `gorm:"unique;not null;index"`
	Password  string
	Verified  bool
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserLoginDto struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserCreateDto struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type UserUpdateDto struct {
	Password string `json:"password"`
	Name     string `json:"name"`
}

type UserDto struct {
	Id       uint64 `json:"id"`
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
	Name     string `json:"name"`
}

type UserLoginResponseDto struct {
	Id    uint64 `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Token string `json:"token"`
}
