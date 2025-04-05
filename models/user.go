package models

type User struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type UserOTP struct {
	UserId     string `json:"user_id"`
	Otp        string `json:"otp"`
	ValidUntil int64  `json:"valid_until"`
}
