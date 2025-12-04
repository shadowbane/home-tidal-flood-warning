package application

import (
	"github.com/shadowbane/home-tidal-flood-warning/pkg/config"
	"github.com/shadowbane/home-tidal-flood-warning/pkg/fetcher"
	"github.com/shadowbane/home-tidal-flood-warning/pkg/models"
	baseapp "github.com/shadowbane/weather-alert/pkg/application"
	weathermodels "github.com/shadowbane/weather-alert/pkg/models"

	"go.uber.org/zap"
)

type Application struct {
	// Embed the weather-alert Application (gives us DB, BMKGFetcher)
	*baseapp.Application

	// Extended config with tidal-specific settings
	Cfg *config.Config

	// Additional fetchers for this app
	TidalFetcher *fetcher.TidalFloodFetcher
}

func Start() (*Application, error) {
	// Start the base weather-alert application first
	baseApp, err := baseapp.Start()
	if err != nil {
		return nil, err
	}

	// Extend the base config with tidal-specific settings
	cfg := config.Extend(baseApp.Cfg)

	zap.S().Info("Extending with Home Tidal Flood Warning")

	// Replace the base BMKG fetcher with our custom filtered version
	baseApp.Fetcher = fetcher.NewBMKGFetcher(baseApp.DB)

	// Run additional migrations for tidal flood models
	zap.S().Debug("Running additional migrations")
	err = baseApp.DB.AutoMigrate([]interface{}{
		// Ensure weather models are migrated (in case base app changes)
		&weathermodels.WeatherAlert{},
		&weathermodels.AlertDetail{},
		// Tidal flood models (local)
		&models.TideData{},
	}...)
	if err != nil {
		zap.S().Fatalf("Error running auto migration: %v", err)
		panic(err)
	}

	// Initialize tidal flood fetcher
	tidalFetcher := fetcher.NewTidalFloodFetcher(baseApp.DB)

	app := &Application{
		Application:  baseApp,
		Cfg:          cfg,
		TidalFetcher: tidalFetcher,
	}

	return app, nil
}

// StartBackgroundJobs starts all background jobs
func (app *Application) StartBackgroundJobs() {
	// Start base app background jobs (BMKG fetcher)
	app.Application.StartBackgroundJobs()
	// Start tidal flood fetcher with its own interval
	app.TidalFetcher.StartPeriodicFetch(app.Cfg.GetTidalFetchInterval())
}

// StopBackgroundJobs stops all background jobs
func (app *Application) StopBackgroundJobs() {
	// Stop base app background jobs
	app.Application.StopBackgroundJobs()
	// Stop tidal flood fetcher
	app.TidalFetcher.Stop()
}
