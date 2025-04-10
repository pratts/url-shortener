package models

type User struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Verified  bool   `json:"verified"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
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
	Id       string `json:"id"`
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
	Name     string `json:"name"`
}
