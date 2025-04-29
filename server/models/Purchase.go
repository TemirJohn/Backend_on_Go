package models

type Purchase struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"not null" json:"userId"`
	GameID uint `gorm:"not null" json:"gameId"`
}
