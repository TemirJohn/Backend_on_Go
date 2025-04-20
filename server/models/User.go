package models

type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Email    string `gorm:"unique;not null" json:"email"`
	Password string `gorm:"not null" json:"password"`
	Name     string `gorm:"not null" json:"name"`
	Role     string `gorm:"not null" json:"role"`
}
