package router

import (
	"github.com/julienschmidt/httprouter"
	"github.com/shadowbane/home-tidal-flood-warning/cmd/api/controllers/tidal"
	"github.com/shadowbane/home-tidal-flood-warning/pkg/application"

	// Import controllers directly from weather-alert
	alertcontroller "github.com/shadowbane/weather-alert/cmd/api/controllers/alert"
	alertByProvince "github.com/shadowbane/weather-alert/cmd/api/controllers/alert/province"
)

func Api(app *application.Application) *httprouter.Router {
	mux := httprouter.New()

	// Weather Alerts (from BMKG) - using weather-alert controllers directly
	mux.GET("/api/v1/alerts", alertcontroller.Index(app.Application))
	mux.GET("/api/v1/alerts/:province", alertByProvince.Index(app.Application))
	mux.POST("/api/v1/alerts/sync", alertcontroller.Sync(app.Application))

	// Tidal Flood Warnings
	mux.GET("/api/v1/tidal-floods", tidal.Index(app))
	mux.GET("/api/v1/tidal-floods/:location", tidal.ByLocation(app))
	mux.POST("/api/v1/tidal-floods/sync", tidal.Sync(app))

	return mux
}
