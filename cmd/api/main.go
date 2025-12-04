package main

import (
	"fmt"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/shadowbane/home-tidal-flood-warning/cmd/api/router"
	"github.com/shadowbane/weather-alert/pkg/exithandler"
	"github.com/shadowbane/weather-alert/pkg/server"
	"go.uber.org/zap"

	"github.com/shadowbane/home-tidal-flood-warning/pkg/application"
)

func main() {
	var cpuCount = runtime.NumCPU()
	if cpuCount > 1 {
		runtime.GOMAXPROCS(cpuCount)
	}

	// load .env
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		fmt.Println("Please ensure you load correct environment variables")
	}

	// start application
	app, err := application.Start()
	if err != nil {
		zap.S().Fatal(err.Error())
	}

	srv := server.
		Get().
		WithAddr(app.Cfg.GetAPIPort()).
		WithRouter(router.Api(app)).
		WithErrLogger(zap.S())

	// Start background jobs (periodic fetch)
	app.StartBackgroundJobs()

	// start the api server
	go func() {
		zap.S().Info("starting api server at ", app.Cfg.GetAPIPort())

		if err := srv.Start(); err != nil {
			zap.S().Warn(err.Error())
		}
	}()

	exithandler.Init(func() {
		zap.S().Info("Closing Application")
		zap.S().Info("Waiting for all the processes to finish")

		// Stop background jobs
		app.StopBackgroundJobs()

		if err := srv.Close(); err != nil {
			zap.S().Error(err.Error())
		}

		zap.S().Info("Application Closed")
	})

	zap.S().Info("Bye!")
}
