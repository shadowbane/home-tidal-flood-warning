package controllertraits

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	basetraits "github.com/shadowbane/weather-alert/pkg/traits/controller-traits"
)

// TidalFloodRisk holds tidal flood risk data for card rendering
type TidalFloodRisk struct {
	HasRisk     bool
	RiskLevel   string // "none", "moderate", "high"
	TideTime    time.Time
	TideHeightM float64
	Message     string
}

// AlertCardData holds the data needed to render an alert card
type AlertCardData struct {
	Event           string
	Effective       time.Time
	Expires         time.Time
	AreaDescription string
	Description     string
	Timezone        string
	FloodRisk       *TidalFloodRisk
	Location        string
}

// GetEventIcon returns an appropriate icon/emoji for the weather event
func GetEventIcon(event string) string {
	eventLower := strings.ToLower(event)

	switch {
	case strings.Contains(eventLower, "thunderstorm"):
		return "⛈️"
	case strings.Contains(eventLower, "thunder") || strings.Contains(eventLower, "lightning"):
		return "⚡"
	case strings.Contains(eventLower, "rain") || strings.Contains(eventLower, "shower"):
		return "🌧️"
	case strings.Contains(eventLower, "wind") || strings.Contains(eventLower, "gale"):
		return "💨"
	case strings.Contains(eventLower, "wave") || strings.Contains(eventLower, "surge"):
		return "🌊"
	case strings.Contains(eventLower, "flood"):
		return "🌊"
	case strings.Contains(eventLower, "heat") || strings.Contains(eventLower, "hot"):
		return "🔥"
	case strings.Contains(eventLower, "fog") || strings.Contains(eventLower, "haze") || strings.Contains(eventLower, "smoke"):
		return "🌫️"
	case strings.Contains(eventLower, "storm") || strings.Contains(eventLower, "extreme") || strings.Contains(eventLower, "severe"):
		return "⛈️"
	case strings.Contains(eventLower, "cyclone") || strings.Contains(eventLower, "typhoon") || strings.Contains(eventLower, "hurricane"):
		return "🌀"
	case strings.Contains(eventLower, "tornado"):
		return "🌪️"
	default:
		return "⚠️"
	}
}

// formatCardTime formats time for card display in Y-m-d H:i format
func formatCardTime(t time.Time, timezone string) string {
	formatted := basetraits.FormatTimeWithTimezone(t, timezone)
	return formatted.Format("2006-01-02 15:04")
}

// renderFloodRiskBadge returns the HTML for the flood risk badge (light mode)
func renderFloodRiskBadge(risk *TidalFloodRisk, timezone string) string {
	if risk == nil || !risk.HasRisk {
		return ""
	}

	var bgColor, borderColor, textColor, icon string
	switch risk.RiskLevel {
	case "high":
		bgColor = "#fef2f2"
		borderColor = "#fca5a5"
		textColor = "#dc2626"
		icon = "🌊"
	case "moderate":
		bgColor = "#fffbeb"
		borderColor = "#fcd34d"
		textColor = "#d97706"
		icon = "⚠️"
	default:
		return ""
	}

	tideTimeStr := formatCardTime(risk.TideTime, timezone)

	return fmt.Sprintf(`
  <div style="margin-top:12px;padding:10px;background:%s;border:1px solid %s;border-radius:8px;">
    <div style="display:flex;align-items:center;gap:6px;">
      <span style="font-size:20px;">%s</span>
      <span style="font-size:12px;font-weight:600;color:%s;text-transform:uppercase;">%s Risk - Tidal Flood</span>
    </div>
    <div style="font-size:11px;color:#64748b;margin-top:6px;">%s</div>
    <div style="display:flex;gap:16px;margin-top:8px;">
      <div>
        <div style="font-size:9px;color:#94a3b8;text-transform:uppercase;">High Tide</div>
        <div style="font-size:11px;font-weight:500;color:#334155;">%s</div>
      </div>
      <div>
        <div style="font-size:9px;color:#94a3b8;text-transform:uppercase;">Height</div>
        <div style="font-size:11px;font-weight:500;color:#334155;">%.1f m</div>
      </div>
    </div>
  </div>`, bgColor, borderColor, icon, textColor, risk.RiskLevel, html.EscapeString(risk.Message), tideTimeStr, risk.TideHeightM)
}

// renderFloodRiskBadgeDark returns the HTML for the flood risk badge (dark mode)
func renderFloodRiskBadgeDark(risk *TidalFloodRisk, timezone string) string {
	if risk == nil || !risk.HasRisk {
		return ""
	}

	var bgColor, borderColor, textColor, icon string
	switch risk.RiskLevel {
	case "high":
		bgColor = "#450a0a"
		borderColor = "#991b1b"
		textColor = "#fca5a5"
		icon = "🌊"
	case "moderate":
		bgColor = "#451a03"
		borderColor = "#92400e"
		textColor = "#fcd34d"
		icon = "⚠️"
	default:
		return ""
	}

	tideTimeStr := formatCardTime(risk.TideTime, timezone)

	return fmt.Sprintf(`
  <div style="margin-top:12px;padding:10px;background:%s;border:1px solid %s;border-radius:8px;">
    <div style="display:flex;align-items:center;gap:6px;">
      <span style="font-size:20px;">%s</span>
      <span style="font-size:12px;font-weight:600;color:%s;text-transform:uppercase;">%s Risk - Tidal Flood</span>
    </div>
    <div style="font-size:11px;color:#94a3b8;margin-top:6px;">%s</div>
    <div style="display:flex;gap:16px;margin-top:8px;">
      <div>
        <div style="font-size:9px;color:#64748b;text-transform:uppercase;">High Tide</div>
        <div style="font-size:11px;font-weight:500;color:#e2e8f0;">%s</div>
      </div>
      <div>
        <div style="font-size:9px;color:#64748b;text-transform:uppercase;">Height</div>
        <div style="font-size:11px;font-weight:500;color:#e2e8f0;">%.1f m</div>
      </div>
    </div>
  </div>`, bgColor, borderColor, icon, textColor, risk.RiskLevel, html.EscapeString(risk.Message), tideTimeStr, risk.TideHeightM)
}

// RenderHTMLCard renders a single alert as an HTML card
func RenderHTMLCard(data AlertCardData) string {
	icon := GetEventIcon(data.Event)
	effective := formatCardTime(data.Effective, data.Timezone)
	expires := formatCardTime(data.Expires, data.Timezone)
	province := html.EscapeString(data.AreaDescription)
	if data.Location != "" {
		// simple title-case: "some area" -> "Some Area"
		titleLocation := strings.Title(strings.ToLower(data.Location)) // deprecated but simple [web:107][web:114]
		province += " - " + html.EscapeString(titleLocation)
	}

	description := html.EscapeString(data.Description)
	event := html.EscapeString(data.Event)
	riskBadge := renderFloodRiskBadge(data.FloodRisk, data.Timezone)

	return fmt.Sprintf(`<div style="width:400px;border:1px solid #e5e7eb;border-radius:12px;padding:16px;font-family:system-ui,-apple-system,sans-serif;background:linear-gradient(135deg,#f8fafc 0%%,#e2e8f0 100%%);box-shadow:0 4px 6px -1px rgba(0,0,0,0.1);">
  <div style="display:flex;align-items:flex-start;gap:12px;">
    <span style="font-size:48px;flex-shrink:0;">%s</span>
    <div style="min-width:0;flex:1;">
      <div style="font-size:18px;font-weight:600;color:#1e293b;">%s</div>
      <div style="font-size:14px;color:#64748b;">%s</div>
    </div>
  </div>
  <div style="font-size:11px;color:#64748b;margin-top:8px;line-height:1.5;">%s</div>%s
  <div style="border-top:1px solid #cbd5e1;margin-top:12px;padding-top:8px;display:flex;justify-content:space-between;">
    <div>
      <div style="font-size:10px;color:#94a3b8;text-transform:uppercase;">Effective</div>
      <div style="font-size:12px;font-weight:500;color:#334155;">%s</div>
    </div>
    <div>
      <div style="font-size:10px;color:#94a3b8;text-transform:uppercase;">Expires</div>
      <div style="font-size:12px;font-weight:500;color:#334155;">%s</div>
    </div>
  </div>
</div>`, icon, event, province, description, riskBadge, effective, expires)
}

// RenderHTMLCardDark renders a single alert as an HTML card in dark mode
func RenderHTMLCardDark(data AlertCardData) string {
	icon := GetEventIcon(data.Event)
	effective := formatCardTime(data.Effective, data.Timezone)
	expires := formatCardTime(data.Expires, data.Timezone)
	province := html.EscapeString(data.AreaDescription)
	if data.Location != "" {
		// simple title-case: "some area" -> "Some Area"
		titleLocation := strings.Title(strings.ToLower(data.Location)) // deprecated but simple [web:107][web:114]
		province += " - " + html.EscapeString(titleLocation)
	}

	description := html.EscapeString(data.Description)
	event := html.EscapeString(data.Event)
	riskBadge := renderFloodRiskBadgeDark(data.FloodRisk, data.Timezone)

	return fmt.Sprintf(`<div style="width:400px;border:1px solid #374151;border-radius:12px;padding:16px;font-family:system-ui,-apple-system,sans-serif;background:linear-gradient(135deg,#1e293b 0%%,#0f172a 100%%);box-shadow:0 4px 6px -1px rgba(0,0,0,0.3);">
  <div style="display:flex;align-items:flex-start;gap:12px;">
    <span style="font-size:48px;flex-shrink:0;">%s</span>
    <div style="min-width:0;flex:1;">
      <div style="font-size:18px;font-weight:600;color:#f1f5f9;">%s</div>
      <div style="font-size:14px;color:#94a3b8;">%s</div>
    </div>
  </div>
  <div style="font-size:11px;color:#94a3b8;margin-top:8px;line-height:1.5;">%s</div>%s
  <div style="border-top:1px solid #475569;margin-top:12px;padding-top:8px;display:flex;justify-content:space-between;">
    <div>
      <div style="font-size:10px;color:#64748b;text-transform:uppercase;">Effective</div>
      <div style="font-size:12px;font-weight:500;color:#e2e8f0;">%s</div>
    </div>
    <div>
      <div style="font-size:10px;color:#64748b;text-transform:uppercase;">Expires</div>
      <div style="font-size:12px;font-weight:500;color:#e2e8f0;">%s</div>
    </div>
  </div>
</div>`, icon, event, province, description, riskBadge, effective, expires)
}

// WriteHTMLResponse writes an HTML response
func WriteHTMLResponse(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(content))
}
