package tidal

import (
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/shadowbane/home-tidal-flood-warning/pkg/application"
	"github.com/shadowbane/home-tidal-flood-warning/pkg/models"
	traits "github.com/shadowbane/weather-alert/pkg/traits/controller-traits"
)

// TidalFloodResponse is the response DTO for tidal flood warnings
type TidalFloodResponse struct {
	ID          string    `json:"id"`
	GUID        string    `json:"guid"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Severity    string    `json:"severity"`
	WaterLevel  float64   `json:"water_level"`
	PubDate     time.Time `json:"pub_date"`
	Effective   time.Time `json:"effective"`
	Expires     time.Time `json:"expires"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// toResponse converts TidalFloodWarning to TidalFloodResponse with optional timezone formatting
func toResponse(warning models.TidalFloodWarning, timezone string) TidalFloodResponse {
	return TidalFloodResponse{
		ID:          warning.ID,
		GUID:        warning.GUID,
		Title:       warning.Title,
		Link:        warning.Link,
		Description: warning.Description,
		Location:    warning.Location,
		Severity:    warning.Severity,
		WaterLevel:  warning.WaterLevel,
		PubDate:     traits.FormatTimeWithTimezone(warning.PubDate, timezone),
		Effective:   traits.FormatTimeWithTimezone(warning.Effective, timezone),
		Expires:     traits.FormatTimeWithTimezone(warning.Expires, timezone),
		CreatedAt:   traits.FormatTimeWithTimezone(warning.CreatedAt, timezone),
		UpdatedAt:   traits.FormatTimeWithTimezone(warning.UpdatedAt, timezone),
	}
}

func Index(app *application.Application) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// Parse pagination parameters
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		timezone := r.URL.Query().Get("timezone")
		activeFilter := r.URL.Query().Get("active")

		// Set defaults
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}

		offset := (page - 1) * limit

		var warnings []models.TidalFloodWarning
		var total int64

		// Build base query
		query := app.DB.Model(&models.TidalFloodWarning{})

		// Apply active filter if requested
		if activeFilter == "true" {
			now := time.Now().UTC()
			query = query.Where("effective <= ? AND expires >= ?", now, now)
		}

		// Get total count
		query.Count(&total)

		// Get paginated results
		result := query.Order("pub_date DESC").Offset(offset).Limit(limit).Find(&warnings)

		if result.Error != nil {
			traits.WriteErrorResponse(w, http.StatusInternalServerError, result.Error.Error())
			return
		}

		// Convert to response DTOs
		responses := make([]TidalFloodResponse, len(warnings))
		for i, warning := range warnings {
			responses[i] = toResponse(warning, timezone)
		}

		// Calculate total pages
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		pagination := traits.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		}

		traits.WritePaginatedResponse(w, responses, pagination)
	}
}

func ByLocation(app *application.Application) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// Parse pagination parameters
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		location := p.ByName("location")
		timezone := r.URL.Query().Get("timezone")
		activeFilter := r.URL.Query().Get("active")

		// Set defaults
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}

		offset := (page - 1) * limit

		var warnings []models.TidalFloodWarning
		var total int64

		// Build query with location filter
		query := app.DB.Model(&models.TidalFloodWarning{}).
			Where("location LIKE ?", "%"+location+"%")

		// Apply active filter if requested
		if activeFilter == "true" {
			now := time.Now().UTC()
			query = query.Where("effective <= ? AND expires >= ?", now, now)
		}

		// Get total count
		query.Count(&total)

		// Get paginated results
		result := query.Order("pub_date DESC").Offset(offset).Limit(limit).Find(&warnings)

		if result.Error != nil {
			traits.WriteErrorResponse(w, http.StatusInternalServerError, result.Error.Error())
			return
		}

		// Convert to response DTOs
		responses := make([]TidalFloodResponse, len(warnings))
		for i, warning := range warnings {
			responses[i] = toResponse(warning, timezone)
		}

		// Calculate total pages
		totalPages := int(total) / limit
		if int(total)%limit > 0 {
			totalPages++
		}

		pagination := traits.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		}

		traits.WritePaginatedResponse(w, responses, pagination)
	}
}
