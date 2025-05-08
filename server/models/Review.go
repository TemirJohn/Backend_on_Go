package models

type Review struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	UserID  uint   `json:"userId"`
	GameID  uint   `json:"gameId"`
	User    User   `gorm:"foreignKey:UserID" json:"user"`
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}
