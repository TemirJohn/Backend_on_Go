package models

type Review struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	UserID  uint   `json:"userId"`
	GameID  uint   `json:"gameId" validate:"required,gte=1"`
	User    User   `gorm:"foreignKey:UserID" json:"user"`
	Rating  int    `json:"rating" validate:"required,gte=1,lte=5"`
	Comment string `json:"comment" validate:"max=1000"`
}

// ReviewCreateInput - for create review
type ReviewCreateInput struct {
	GameID  uint   `json:"gameId" validate:"required,gte=1"`
	Rating  int    `json:"rating" validate:"required,gte=1,lte=5"`
	Comment string `json:"comment" validate:"max=1000"`
}
