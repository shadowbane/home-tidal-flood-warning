package fetcher

import (
	"strings"
	"time"

	"github.com/shadowbane/weather-alert/pkg/models"

	basefetcher "github.com/shadowbane/weather-alert/pkg/fetcher"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProvinceFilter is the province filter for alerts
const ProvinceFilter = "Kep. Riau"

// BMKGFetcher wraps the base BMKGFetcher with province filtering
type BMKGFetcher struct {
	*basefetcher.BMKGFetcher
	db       *gorm.DB
	stopChan chan struct{}
}

// NewBMKGFetcher creates a new BMKGFetcher with province filtering
func NewBMKGFetcher(db *gorm.DB) *BMKGFetcher {
	return &BMKGFetcher{
		BMKGFetcher: basefetcher.NewBMKGFetcher(db),
		db:          db,
		stopChan:    make(chan struct{}),
	}
}

// FetchAndStore fetches alerts from BMKG, filters by province, and stores them
func (f *BMKGFetcher) FetchAndStore() (int, error) {
	// Use the base Fetch() to get all alerts
	alerts, err := f.BMKGFetcher.Fetch()
	if err != nil {
		return 0, err
	}

	// Filter alerts to only those containing the province filter
	filteredAlerts := make([]models.WeatherAlert, 0)
	for _, alert := range alerts {
		if strings.Contains(alert.Province, ProvinceFilter) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	zap.S().Infof("Filtered %d alerts to %d alerts for province containing '%s'",
		len(alerts), len(filteredAlerts), ProvinceFilter)

	count := 0
	storedAlerts := make([]models.WeatherAlert, 0, len(filteredAlerts))

	for _, alert := range filteredAlerts {
		// Use GUID as unique identifier to avoid duplicates
		var existing models.WeatherAlert
		result := f.db.Where("guid = ?", alert.GUID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			// Insert new record
			if err := f.db.Create(&alert).Error; err != nil {
				zap.S().Errorf("Failed to insert alert: %v", err)
				continue
			}
			count++
			storedAlerts = append(storedAlerts, alert)
		} else if result.Error == nil {
			// Update existing record - preserve ID and CreatedAt
			alert.ID = existing.ID
			alert.CreatedAt = existing.CreatedAt
			if err := f.db.Save(&alert).Error; err != nil {
				zap.S().Errorf("Failed to update alert: %v", err)
				continue
			}
			storedAlerts = append(storedAlerts, alert)
		}
	}

	zap.S().Infof("Synced %d new alerts from BMKG (filtered for %s)", count, ProvinceFilter)

	// Fetch alert details concurrently (max 5 concurrent requests)
	if len(storedAlerts) > 0 {
		go func() {
			zap.S().Infof("Fetching details for %d alerts concurrently", len(storedAlerts))
			results := f.BMKGFetcher.FetchAlertDetailsConcurrently(storedAlerts, 5)
			detailCount := f.BMKGFetcher.StoreAlertDetails(results)
			zap.S().Infof("Stored %d alert details", detailCount)
		}()
	}

	return count, nil
}

// StartPeriodicFetch starts a background goroutine that fetches alerts periodically
func (f *BMKGFetcher) StartPeriodicFetch(interval time.Duration) {
	zap.S().Infof("Starting periodic BMKG fetch (filtered for %s) every %v", ProvinceFilter, interval)

	// Fetch immediately on start
	go func() {
		if _, err := f.FetchAndStore(); err != nil {
			zap.S().Errorf("Initial BMKG fetch failed: %v", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				zap.S().Debug("Running scheduled BMKG fetch")
				if _, err := f.FetchAndStore(); err != nil {
					zap.S().Errorf("Scheduled BMKG fetch failed: %v", err)
				}
			case <-f.stopChan:
				zap.S().Info("Stopping periodic BMKG fetch")
				return
			}
		}
	}()
}

// Stop stops the periodic fetching
func (f *BMKGFetcher) Stop() {
	close(f.stopChan)
}
