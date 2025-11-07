package models

type Category struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Name string `gorm:"not null" json:"name" validate:"required,min=2,max=100"`
}

// CategoryInput - для создания/обновления категории
type CategoryInput struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}
