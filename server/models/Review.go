package models

type Review struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	UserID  uint   `json:"userId"`
	GameID  uint   `json:"gameId"`
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}
