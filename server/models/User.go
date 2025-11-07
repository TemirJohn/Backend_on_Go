package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	ID       uint   `gorm:"primaryKey" json:"id"`
	Email    string `gorm:"unique;not null" json:"email" validate:"required,email"`
	Password string `gorm:"not null" json:"password" validate:"required,min=6"`
	Name     string `gorm:"not null" json:"name" validate:"required,min=3,max=50"`
	Role     string `gorm:"not null" json:"role" validate:"required,oneof=user developer admin"`
	Avatar   string `json:"avatar"`
	IsBanned bool   `gorm:"default:false" json:"isBanned"`
}

// LoginInput - используется для валидации логина
type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// RegisterInput - используется для валидации регистрации
type RegisterInput struct {
	Username string `json:"username" form:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" form:"email" validate:"required,email"`
	Password string `json:"password" form:"password" validate:"required,min=6,max=100"`
	Role     string `json:"role" form:"role" validate:"required,oneof=user developer admin"`
}

// UpdateUserInput - используется для обновления пользователя
type UpdateUserInput struct {
	Name     *string `json:"name" form:"name" validate:"omitempty,min=3,max=50"`
	Role     *string `json:"role" form:"role" validate:"omitempty,oneof=user developer admin"`
	IsBanned *bool   `json:"isBanned" form:"isBanned"`
}
