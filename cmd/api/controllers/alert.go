package controllers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/shadowbane/home-tidal-flood-warning/pkg/models"
	traits "github.com/shadowbane/home-tidal-flood-warning/pkg/traits/controller-traits"
	"github.com/shadowbane/weather-alert/pkg/application"
	weathermodels "github.com/shadowbane/weather-alert/pkg/models"
	basetraits "github.com/shadowbane/weather-alert/pkg/traits/controller-traits"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// offsetRegex matches UTC offset formats: +08:00, -05:30, +0800, -0530
var offsetRegex = regexp.MustCompile(`^([+-])(\d{2}):?(\d{2})$`)

// parseTimezone converts a UTC offset to IANA timezone string, or returns as-is.
// Examples: "+08:00" -> "Etc/GMT-8", "-05:00" -> "Etc/GMT+5"
// Note: Etc/GMT signs are inverted (Etc/GMT-8 = UTC+08:00)
// For non-offset input, returns the original string unchanged.
func parseTimezone(tz string) string {
	if tz == "" {
		return tz
	}

	matches := offsetRegex.FindStringSubmatch(tz)
	if matches == nil {
		return tz // Not an offset, return as-is (e.g., "Asia/Singapore")
	}

	sign := matches[1]
	hours, _ := strconv.Atoi(matches[2])
	// minutes not used - Etc/GMT only supports whole hours

	// Etc/GMT signs are inverted: +08:00 -> Etc/GMT-8
	if sign == "+" {
		return "Etc/GMT-" + strconv.Itoa(hours)
	}
	return "Etc/GMT+" + strconv.Itoa(hours)
}

// TidalFloodRisk represents the tidal flood risk assessment
type TidalFloodRisk struct {
	HasRisk     bool      `json:"has_risk"`
	RiskLevel   string    `json:"risk_level"`    // "none", "moderate", "high"
	TideType    string    `json:"tide_type"`     // "high" or "low"
	TideTime    time.Time `json:"tide_time"`     // When the high tide occurs
	TideHeightM float64   `json:"tide_height_m"` // Height in meters
	HeavyRain   bool      `json:"heavy_rain"`    // Whether heavy rain is expected
	Message     string    `json:"message"`       // Human-readable risk message
}

// AlertDetailResponse is the response DTO for alert details
// It excludes Polygon and WeatherAlert properties
type AlertDetailResponse struct {
	ID              string          `json:"id"`
	WeatherAlertID  string          `json:"weather_alert_id"`
	Identifier      string          `json:"identifier"`
	Sender          string          `json:"sender"`
	Sent            time.Time       `json:"sent"`
	Status          string          `json:"status"`
	MsgType         string          `json:"msg_type"`
	Scope           string          `json:"scope"`
	Language        string          `json:"language"`
	Category        string          `json:"category"`
	Event           string          `json:"event"`
	Urgency         string          `json:"urgency"`
	Severity        string          `json:"severity"`
	Certainty       string          `json:"certainty"`
	EventCode       string          `json:"event_code"`
	Effective       time.Time       `json:"effective"`
	Expires         time.Time       `json:"expires"`
	SenderName      string          `json:"sender_name"`
	Headline        string          `json:"headline"`
	Description     string          `json:"description"`
	Instruction     string          `json:"instruction"`
	Web             string          `json:"web"`
	Contact         string          `json:"contact"`
	AreaDescription string          `json:"area_description"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	TidalFloodRisk  *TidalFloodRisk `json:"tidal_flood_risk,omitempty"`
}

// toResponse converts AlertDetail to AlertDetailResponse with optional timezone formatting
func toResponse(detail weathermodels.AlertDetail, timezone string, floodRisk *TidalFloodRisk) AlertDetailResponse {
	return AlertDetailResponse{
		ID:              detail.ID,
		WeatherAlertID:  detail.WeatherAlertID,
		Identifier:      detail.Identifier,
		Sender:          detail.Sender,
		Sent:            basetraits.FormatTimeWithTimezone(detail.Sent, timezone),
		Status:          detail.Status,
		MsgType:         detail.MsgType,
		Scope:           detail.Scope,
		Language:        detail.Language,
		Category:        detail.Category,
		Event:           detail.Event,
		Urgency:         detail.Urgency,
		Severity:        detail.Severity,
		Certainty:       detail.Certainty,
		EventCode:       detail.EventCode,
		Effective:       basetraits.FormatTimeWithTimezone(detail.Effective, timezone),
		Expires:         basetraits.FormatTimeWithTimezone(detail.Expires, timezone),
		SenderName:      detail.SenderName,
		Headline:        detail.Headline,
		Description:     detail.Description,
		Instruction:     detail.Instruction,
		Web:             detail.Web,
		Contact:         detail.Contact,
		AreaDescription: detail.AreaDescription,
		CreatedAt:       basetraits.FormatTimeWithTimezone(detail.CreatedAt, timezone),
		UpdatedAt:       basetraits.FormatTimeWithTimezone(detail.UpdatedAt, timezone),
		TidalFloodRisk:  floodRisk,
	}
}

// Buffer time to account for rising sea level before high tide peak
const tideBufferDuration = 2 * time.Hour

// calculateTidalFloodRisk calculates the risk of tidal flooding based on alert and tide data
// Risk conditions: heavy rain + high tide (>2.5m) where tide_time overlaps with alert period
// Sea level rises gradually, so we add a buffer after alert expires to catch rising water scenarios
func calculateTidalFloodRisk(db *gorm.DB, alert weathermodels.AlertDetail, timezone string) *TidalFloodRisk {
	// Check if alert description contains "heavy rain" or "heavy rainfall"
	descLower := strings.ToLower(alert.Description)
	hasHeavyRain := strings.Contains(descLower, "heavy rain")

	if !hasHeavyRain {
		return &TidalFloodRisk{
			HasRisk:   false,
			RiskLevel: "none",
			HeavyRain: false,
			Message:   "No heavy rain expected",
			TideTime:  basetraits.FormatTimeWithTimezone(time.Now().UTC(), timezone),
		}
	}

	// Extend the check window by buffer to account for rising sea level
	// Sea level rises gradually before high tide peak, so if high tide is shortly after
	// the alert expires, there's still risk from rising water during the alert period
	expiresWithBuffer := alert.Expires.Add(tideBufferDuration)

	// Query tide data for high tides (>2.5m) within alert period + buffer
	var tideData []models.TideData
	result := db.Where("tide_type = ? AND height_m > ? AND tide_time >= ? AND tide_time <= ?",
		models.TideTypeHigh, 2.6, alert.Effective, expiresWithBuffer).
		Order("height_m DESC").
		Find(&tideData)

	if result.Error != nil {
		zap.S().Errorf("Failed to query tide data: %v", result.Error)
		return &TidalFloodRisk{
			HasRisk:   false,
			RiskLevel: "unknown",
			HeavyRain: hasHeavyRain,
			Message:   "Unable to determine tidal flood risk",
			TideTime:  basetraits.FormatTimeWithTimezone(time.Now().UTC(), timezone),
		}
	}

	if len(tideData) == 0 {
		// No high tide > 2.6m during the alert period or buffer
		return &TidalFloodRisk{
			HasRisk:   false,
			RiskLevel: "none",
			HeavyRain: hasHeavyRain,
			Message:   "No tidal flood risk: No high tide (>2.6m) during or near alert period",
			TideTime:  basetraits.FormatTimeWithTimezone(time.Now().UTC(), timezone),
		}
	}

	highestTide := tideData[0]

	// Determine risk level based on whether high tide is within alert period or in buffer zone
	if highestTide.TideTime.After(alert.Expires) {
		// High tide is in the buffer zone (after alert expires but within 2 hours)
		// Still risky because sea level is already rising during the alert
		return &TidalFloodRisk{
			HasRisk:     true,
			RiskLevel:   "moderate",
			TideType:    string(highestTide.TideType),
			TideTime:    highestTide.TideTime,
			TideHeightM: highestTide.HeightM,
			HeavyRain:   hasHeavyRain,
			Message:     "MODERATE RISK: Heavy rain with high tide (>2.6m) shortly after - Sea level rising during alert period",
		}
	}

	// High tide > 2.6m during the alert period with heavy rain = high risk
	return &TidalFloodRisk{
		HasRisk:     true,
		RiskLevel:   "high",
		TideType:    string(highestTide.TideType),
		TideTime:    highestTide.TideTime,
		TideHeightM: highestTide.HeightM,
		HeavyRain:   hasHeavyRain,
		Message:     "HIGH RISK: Heavy rain expected during high tide (>2.6m) - Flash flood possible!",
	}
}

func Index(app *application.Application) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// Parse pagination parameters
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		timezone := parseTimezone(r.URL.Query().Get("timezone"))
		activeFilter := r.URL.Query().Get("active")
		locationFilter := r.URL.Query().Get("location")
		asCard := r.URL.Query().Get("as-card")

		// Set defaults
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}

		offset := (page - 1) * limit

		var alertDetails []weathermodels.AlertDetail
		var total int64

		// Build query with home location
		query := app.DB.Model(&weathermodels.AlertDetail{}).
			Where("area_description = ?", "Kep. Riau")

		// Apply active filter if requested
		if activeFilter == "true" {
			now := time.Now().UTC()
			query = query.Where("effective <= ? AND expires >= ?", now, now)
		}

		// Apply location filter
		if locationFilter != "" {
			zap.S().Debugf("Query Location Filter: %s", locationFilter)
			query = query.Where("description LIKE ?", "%"+locationFilter+",%")
		}

		// Check if card format is requested
		isCardMode := asCard == "html" || asCard == "html-dark"

		// Get total count (skip for card mode since we only need 1)
		if !isCardMode {
			query.Count(&total)
		}

		// Get results - limit to 1 for card mode
		queryLimit := limit
		queryOffset := offset
		if isCardMode {
			queryLimit = 1
			queryOffset = 0
		}

		result := query.Order("sent DESC").
			Offset(queryOffset).
			Limit(queryLimit).
			Find(&alertDetails)

		if result.Error != nil {
			basetraits.WriteErrorResponse(w, http.StatusInternalServerError, result.Error.Error())
			return
		}

		// Handle card format response
		if isCardMode {
			if len(alertDetails) == 0 {
				// Render "no alert" card instead of error
				switch asCard {
				case "html":
					traits.WriteHTMLResponse(w, traits.RenderNoAlertCard(locationFilter))
				case "html-dark":
					traits.WriteHTMLResponse(w, traits.RenderNoAlertCardDark(locationFilter))
				}
				return
			}

			// Calculate flood risk for card
			floodRisk := calculateTidalFloodRisk(app.DB, alertDetails[0], timezone)

			// Convert to traits.TidalFloodRisk for card rendering
			var cardFloodRisk *traits.TidalFloodRisk
			if floodRisk != nil && floodRisk.HasRisk {
				cardFloodRisk = &traits.TidalFloodRisk{
					HasRisk:     floodRisk.HasRisk,
					RiskLevel:   floodRisk.RiskLevel,
					TideTime:    floodRisk.TideTime,
					TideHeightM: floodRisk.TideHeightM,
					Message:     floodRisk.Message,
				}
			}

			card := traits.AlertCardData{
				Event:           alertDetails[0].Event,
				Effective:       alertDetails[0].Effective,
				Expires:         alertDetails[0].Expires,
				AreaDescription: alertDetails[0].AreaDescription,
				Description:     alertDetails[0].Description,
				Timezone:        timezone,
				FloodRisk:       cardFloodRisk,
				Location:        locationFilter,
			}

			switch asCard {
			case "html":
				traits.WriteHTMLResponse(w, traits.RenderHTMLCard(card))
			case "html-dark":
				traits.WriteHTMLResponse(w, traits.RenderHTMLCardDark(card))
			}
			return
		}

		// Convert to response DTOs with tidal flood risk calculation
		responses := make([]AlertDetailResponse, len(alertDetails))
		for i, detail := range alertDetails {
			floodRisk := calculateTidalFloodRisk(app.DB, detail, timezone)
			responses[i] = toResponse(detail, timezone, floodRisk)
		}

		// Calculate total pages
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		pagination := basetraits.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		}

		basetraits.WritePaginatedResponse(w, responses, pagination)
	}
}
