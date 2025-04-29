package models

type Ownership struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"not null" json:"userId"`
	GameID uint   `gorm:"not null" json:"gameId"`
	Status string `gorm:"not null" json:"status"`
	Game   Game   `gorm:"foreignKey:GameID" json:"game"`
}
