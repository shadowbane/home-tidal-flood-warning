package fetcher

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/shadowbane/home-tidal-flood-warning/pkg/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// WorldTidesURL is the URL to scrape tide data from
	WorldTidesURL = "https://www.worldtides.info/tidestations/Sekupang"
	// TideLocation is the location name for the tide data
	TideLocation = "Sekupang"
)

// UTC+7 timezone
var wibTimezone = time.FixedZone("WIB", 7*60*60)

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

// FetchAndStore fetches tide data and stores it in the database using a transaction
func (f *TidalFloodFetcher) FetchAndStore() (int, error) {
	tideData, date, err := f.Fetch()
	if err != nil {
		return 0, err
	}

	if len(tideData) == 0 {
		zap.S().Info("No tide data fetched")
		return 0, nil
	}

	count := 0

	// Use transaction to replace all data for the date
	err = f.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing data for the same date and location
		// Note: date is kept in WIB for correct logical date storage
		if err := tx.Where("location = ? AND date = ?", TideLocation, date).
			Delete(&models.TideData{}).Error; err != nil {
			return fmt.Errorf("failed to delete existing tide data: %w", err)
		}

		zap.S().Infof("Deleted existing tide data for %s on %s", TideLocation, date.Format("2006-01-02"))

		// Insert new data
		for _, data := range tideData {
			if err := tx.Create(&data).Error; err != nil {
				return fmt.Errorf("failed to insert tide data: %w", err)
			}
			count++
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	zap.S().Infof("Synced %d tide data entries for %s on %s", count, TideLocation, date.Format("2006-01-02"))
	return count, nil
}

// Fetch retrieves and parses tide data from worldtides.info
func (f *TidalFloodFetcher) Fetch() ([]models.TideData, time.Time, error) {
	zap.S().Debugf("Fetching tide data from %s", WorldTidesURL)

	resp, err := http.Get(WorldTidesURL)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to fetch tide data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, time.Time{}, fmt.Errorf("worldtides.info returned status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Parse the date from the header div
	// Format: "Tide Times for Sekupang: Thursday December 4, 2025 (WIB)"
	dateText := ""
	doc.Find("div").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.Contains(text, "Tide Times for") && strings.Contains(text, "(WIB)") {
			dateText = text
		}
	})

	if dateText == "" {
		return nil, time.Time{}, fmt.Errorf("could not find tide date header")
	}

	// dateForStorage: UTC midnight with correct Y/M/D (for DB storage)
	// dateWIB: WIB midnight (for combining with tide times)
	dateForStorage, dateWIB, err := parseTideDate(dateText)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to parse tide date: %w", err)
	}

	zap.S().Debugf("Parsing tide data for date: %s", dateForStorage.Format("2006-01-02"))

	tideData := make([]models.TideData, 0)

	// Parse the tide table
	doc.Find("table.table-bordered tr").Each(func(i int, s *goquery.Selection) {
		// Skip header row
		if i == 0 {
			return
		}

		cols := s.Find("td")
		if cols.Length() != 3 {
			return
		}

		tideTypeStr := strings.TrimSpace(cols.Eq(0).Text())
		timeStr := strings.TrimSpace(cols.Eq(1).Text())
		heightStr := strings.TrimSpace(cols.Eq(2).Text())

		// Parse tide type
		var tideType models.TideType
		if strings.Contains(strings.ToLower(tideTypeStr), "high") {
			tideType = models.TideTypeHigh
		} else if strings.Contains(strings.ToLower(tideTypeStr), "low") {
			tideType = models.TideTypeLow
		} else {
			zap.S().Warnf("Unknown tide type: %s", tideTypeStr)
			return
		}

		// Parse time using WIB date (for correct hour/minute combination)
		tideTime, err := parseTimeWIB(dateWIB, timeStr)
		if err != nil {
			zap.S().Warnf("Failed to parse tide time '%s': %v", timeStr, err)
			return
		}

		// Parse height (format: "1.1 m (3.6 ft)")
		heightM, heightFt, err := parseHeight(heightStr)
		if err != nil {
			zap.S().Warnf("Failed to parse tide height '%s': %v", heightStr, err)
			return
		}

		data := models.TideData{
			Location: TideLocation,
			Date:     dateForStorage, // UTC midnight with correct Y/M/D for DB
			TideType: tideType,
			TideTime: tideTime, // Converted to UTC in parseTimeWIB for accurate comparisons
			HeightM:  heightM,
			HeightFt: heightFt,
		}

		tideData = append(tideData, data)
	})

	if len(tideData) == 0 {
		return nil, time.Time{}, fmt.Errorf("no tide data found in the table")
	}

	zap.S().Infof("Fetched %d tide entries for %s", len(tideData), dateForStorage.Format("2006-01-02"))
	return tideData, dateForStorage, nil
}

// StartPeriodicFetch starts a background goroutine that fetches at 2-hour intervals aligned to UTC+7
func (f *TidalFloodFetcher) StartPeriodicFetch(interval time.Duration) {
	zap.S().Info("Starting periodic tide data fetch (every 2 hours aligned to WIB)")

	// Fetch immediately on start
	go func() {
		if _, err := f.FetchAndStore(); err != nil {
			zap.S().Errorf("Initial tide data fetch failed: %v", err)
		}
	}()

	go func() {
		for {
			// Calculate next 2-hour mark in WIB (00:00, 02:00, 04:00, etc.)
			nextRun := calculateNext2HourMark()
			sleepDuration := time.Until(nextRun)

			zap.S().Infof("Next tide data fetch scheduled at %s (in %v)",
				nextRun.In(wibTimezone).Format("2006-01-02 15:04:05 MST"), sleepDuration)

			select {
			case <-time.After(sleepDuration):
				zap.S().Debug("Running scheduled tide data fetch")
				if _, err := f.FetchAndStore(); err != nil {
					zap.S().Errorf("Scheduled tide data fetch failed: %v", err)
				}
			case <-f.stopChan:
				zap.S().Info("Stopping periodic tide data fetch")
				return
			}
		}
	}()
}

// Stop stops the periodic fetching
func (f *TidalFloodFetcher) Stop() {
	close(f.stopChan)
}

// calculateNext2HourMark calculates the next 2-hour aligned time in WIB
// Returns the time in UTC for use with time.After
func calculateNext2HourMark() time.Time {
	now := time.Now().In(wibTimezone)

	// Get current hour and round up to next 2-hour mark
	currentHour := now.Hour()
	nextHour := ((currentHour / 2) + 1) * 2

	// Create the next run time
	nextRun := time.Date(
		now.Year(), now.Month(), now.Day(),
		nextHour%24, 0, 0, 0,
		wibTimezone,
	)

	// If next hour is >= 24, it's the next day
	if nextHour >= 24 {
		nextRun = nextRun.AddDate(0, 0, 1)
	}

	return nextRun
}

// parseTideDate parses the date from text like "Tide Times for Sekupang: Thursday December 4, 2025 (WIB)"
// Returns two values: dateForStorage (UTC midnight for DB) and dateWIB (for combining with times)
func parseTideDate(text string) (dateForStorage time.Time, dateWIB time.Time, err error) {
	// Extract date portion using regex
	re := regexp.MustCompile(`(\w+)\s+(\w+)\s+(\d+),\s+(\d+)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) < 5 {
		return time.Time{}, time.Time{}, fmt.Errorf("could not extract date from: %s", text)
	}

	// Parse: "Thursday December 4, 2025"
	dateStr := fmt.Sprintf("%s %s, %s", matches[2], matches[3], matches[4])
	date, err := time.ParseInLocation("January 2, 2006", dateStr, wibTimezone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	// dateWIB: midnight in WIB timezone (for combining with tide times)
	dateWIB = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, wibTimezone)

	// dateForStorage: same Y/M/D but in UTC (so MySQL stores correct date)
	dateForStorage = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)

	return dateForStorage, dateWIB, nil
}

// parseTimeWIB parses a time string like "03:12" and combines with the date in WIB,
// then converts to UTC for consistent storage (required for SQLite compatibility)
func parseTimeWIB(date time.Time, timeStr string) (time.Time, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, err
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, err
	}

	// Create time in WIB, then convert to UTC for storage
	wibTime := time.Date(
		date.Year(), date.Month(), date.Day(),
		hour, minute, 0, 0,
		wibTimezone,
	)
	return wibTime.UTC(), nil
}

// parseHeight parses height string like "1.1 m (3.6 ft)" and returns meters and feet
func parseHeight(heightStr string) (float64, float64, error) {
	// Regex to extract: "1.1 m (3.6 ft)" or "-0.1 m (-0.3 ft)"
	re := regexp.MustCompile(`(-?[\d.]+)\s*m\s*\((-?[\d.]+)\s*ft\)`)
	matches := re.FindStringSubmatch(heightStr)
	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("could not parse height: %s", heightStr)
	}

	heightM, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, 0, err
	}

	heightFt, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return 0, 0, err
	}

	return heightM, heightFt, nil
}
