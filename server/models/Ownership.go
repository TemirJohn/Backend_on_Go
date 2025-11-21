package models

type Ownership struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"not null" json:"userId"`
	GameID uint   `gorm:"not null" json:"gameId" validate:"required,gte=1"`
	Status string `gorm:"not null" json:"status" validate:"required,oneof=owned wishlisted"`
	Game   Game   `gorm:"foreignKey:GameID" json:"game"`
}

// BuyGameInput - for buy game
type BuyGameInput struct {
	GameID uint `json:"gameId" validate:"required,gte=1"`
}
