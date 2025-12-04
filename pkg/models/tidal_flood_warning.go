package models

import (
	"time"

	"github.com/shadowbane/weather-alert/pkg/helpers"

	"gorm.io/gorm"
)

// TidalFloodWarning stores tidal flood warning data
type TidalFloodWarning struct {
	ID          string    `json:"id" gorm:"type:char(26);primaryKey;autoIncrement:false"`
	GUID        string    `json:"guid" gorm:"uniqueIndex;type:varchar(255)"`
	Title       string    `json:"title" gorm:"type:text"`
	Link        string    `json:"link" gorm:"type:text"`
	Description string    `json:"description" gorm:"type:text"`
	Location    string    `json:"location" gorm:"index;type:varchar(255)"`
	Severity    string    `json:"severity" gorm:"type:varchar(50)"`
	WaterLevel  float64   `json:"water_level"`
	PubDate     time.Time `json:"pub_date"`
	Effective   time.Time `json:"effective"`
	Expires     time.Time `json:"expires"`
	CreatedAt   time.Time `json:"created_at" gorm:"type:timestamp"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:timestamp"`
}

func (t *TidalFloodWarning) TableName() string {
	return "tidal_flood_warnings"
}

// BeforeCreate will set a ULID rather than numeric ID.
func (t *TidalFloodWarning) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = helpers.NewULID()
	}
	return nil
}
