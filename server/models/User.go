package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	ID       uint   `gorm:"primaryKey" json:"id"`
	Email    string `gorm:"unique;not null" json:"email"`
	Password string `gorm:"not null" json:"password"`
	Name     string `gorm:"not null" json:"name"`
	Role     string `gorm:"not null" json:"role"`
}
