package router

import (
	"github.com/julienschmidt/httprouter"
	"github.com/shadowbane/home-tidal-flood-warning/pkg/application"

	// Import controllers directly from weather-alert
	alertcontroller "github.com/shadowbane/home-tidal-flood-warning/cmd/api/controllers"
)

func Api(app *application.Application) *httprouter.Router {
	mux := httprouter.New()

	// Weather Alerts (from BMKG) - using weather-alert controllers directly
	mux.GET("/api/v1/alerts", alertcontroller.Index(app.Application))

	return mux
}
