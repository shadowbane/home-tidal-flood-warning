package models

import (
	"time"

	"github.com/shadowbane/weather-alert/pkg/helpers"

	"gorm.io/gorm"
)

// TideType represents the type of tide (high or low)
type TideType string

const (
	TideTypeHigh TideType = "high"
	TideTypeLow  TideType = "low"
)

// TideData stores tide level data scraped from worldtides.info
type TideData struct {
	ID        string    `json:"id" gorm:"type:char(26);primaryKey;autoIncrement:false"`
	Location  string    `json:"location" gorm:"index;type:varchar(255)"`
	Date      time.Time `json:"date" gorm:"index;type:date"`
	TideType  TideType  `json:"tide_type" gorm:"type:varchar(10)"`
	TideTime  time.Time `json:"tide_time" gorm:"type:timestamp"`
	HeightM   float64   `json:"height_m"`
	HeightFt  float64   `json:"height_ft"`
	CreatedAt time.Time `json:"created_at" gorm:"type:timestamp"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamp"`
}

func (t *TideData) TableName() string {
	return "tide_data"
}

// BeforeCreate will set a ULID rather than numeric ID.
func (t *TideData) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = helpers.NewULID()
	}
	return nil
}
