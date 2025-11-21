package models

type Game struct {
	ID          uint     `gorm:"primaryKey" json:"id"`
	Name        string   `gorm:"not null" json:"name" validate:"required,min=1,max=200"`
	Price       float64  `gorm:"not null" json:"price" validate:"required,gte=0"`
	Description string   `json:"description" validate:"max=2000"`
	CategoryID  uint     `json:"category_id" validate:"required,gte=1"`
	Category    Category `gorm:"foreignKey:CategoryID" json:"category"`
	Image       string   `json:"image"`
	DeveloperID uint     `json:"developerId" validate:"required,gte=1"`
}

// GameCreateInput - for create game
type GameCreateInput struct {
	Name        string  `form:"name" validate:"required,min=1,max=200"`
	Price       float64 `form:"price" validate:"required,gte=0"`
	Description string  `form:"description" validate:"max=2000"`
	CategoryID  uint    `form:"category_id" validate:"required,gte=1"`
	DeveloperID uint    `form:"developerId" validate:"required,gte=1"`
}

// GameUpdateInput - for update game
type GameUpdateInput struct {
	Name        *string  `form:"name" validate:"omitempty,min=1,max=200"`
	Price       *float64 `form:"price" validate:"omitempty,gte=0"`
	Description *string  `form:"description" validate:"omitempty,max=2000"`
}
