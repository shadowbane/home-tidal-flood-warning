package fetcher

import (
	"time"

	"github.com/shadowbane/home-tidal-flood-warning/pkg/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TidalFloodFetcher handles fetching and parsing tidal flood warnings
// Implements the fetcher.Fetcher interface from weather-alert
type TidalFloodFetcher struct {
	db       *gorm.DB
	stopChan chan struct{}
}

// NewTidalFloodFetcher creates a new TidalFloodFetcher instance
func NewTidalFloodFetcher(db *gorm.DB) *TidalFloodFetcher {
	return &TidalFloodFetcher{
		db:       db,
		stopChan: make(chan struct{}),
	}
}

// FetchAndStore fetches tidal flood warnings and stores them in the database
func (f *TidalFloodFetcher) FetchAndStore() (int, error) {
	warnings, err := f.Fetch()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, warning := range warnings {
		// Use GUID as unique identifier to avoid duplicates
		var existing models.TidalFloodWarning
		result := f.db.Where("guid = ?", warning.GUID).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			// Insert new record
			if err := f.db.Create(&warning).Error; err != nil {
				zap.S().Errorf("Failed to insert tidal flood warning: %v", err)
				continue
			}
			count++
		} else if result.Error == nil {
			// Update existing record - preserve ID and CreatedAt
			warning.ID = existing.ID
			warning.CreatedAt = existing.CreatedAt
			if err := f.db.Save(&warning).Error; err != nil {
				zap.S().Errorf("Failed to update tidal flood warning: %v", err)
				continue
			}
		}
	}

	zap.S().Infof("Synced %d new tidal flood warnings", count)
	return count, nil
}

// Fetch retrieves and parses tidal flood data from the data source
// TODO: Implement actual data source fetching
func (f *TidalFloodFetcher) Fetch() ([]models.TidalFloodWarning, error) {
	// TODO: Replace with actual data source URL and parsing logic
	// This is a placeholder implementation
	zap.S().Debug("Fetching tidal flood data...")

	warnings := make([]models.TidalFloodWarning, 0)

	// Example placeholder - replace with actual API/RSS fetch
	// resp, err := http.Get("https://example.com/tidal-flood-api")
	// if err != nil {
	//     return nil, fmt.Errorf("failed to fetch tidal flood data: %w", err)
	// }
	// defer resp.Body.Close()
	// ... parse response ...

	return warnings, nil
}

// StartPeriodicFetch starts a background goroutine that fetches warnings periodically
func (f *TidalFloodFetcher) StartPeriodicFetch(interval time.Duration) {
	zap.S().Infof("Starting periodic tidal flood fetch every %v", interval)

	// Fetch immediately on start
	go func() {
		if _, err := f.FetchAndStore(); err != nil {
			zap.S().Errorf("Initial tidal flood fetch failed: %v", err)
		}
	}()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				zap.S().Debug("Running scheduled tidal flood fetch")
				if _, err := f.FetchAndStore(); err != nil {
					zap.S().Errorf("Scheduled tidal flood fetch failed: %v", err)
				}
			case <-f.stopChan:
				zap.S().Info("Stopping periodic tidal flood fetch")
				return
			}
		}
	}()
}

// Stop stops the periodic fetching
func (f *TidalFloodFetcher) Stop() {
	close(f.stopChan)
}
