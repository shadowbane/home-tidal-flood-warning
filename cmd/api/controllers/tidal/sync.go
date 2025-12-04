package tidal

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/shadowbane/home-tidal-flood-warning/pkg/application"
	traits "github.com/shadowbane/weather-alert/pkg/traits/controller-traits"
)

func Sync(app *application.Application) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		count, err := app.TidalFetcher.FetchAndStore()
		if err != nil {
			traits.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		traits.WriteResponse(w, map[string]interface{}{
			"message": "Sync completed",
			"count":   count,
		})
	}
}
