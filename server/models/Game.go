package models

type Game struct {
	ID          uint     `gorm:"primaryKey" json:"id"`
	Name        string   `gorm:"not null" json:"name"`
	Price       float64  `gorm:"not null" json:"price"`
	Description string   `json:"description"`
	CategoryID  uint     `json:"categoryId"`
	Category    Category `gorm:"foreignKey:CategoryID" json:"category"`
}
