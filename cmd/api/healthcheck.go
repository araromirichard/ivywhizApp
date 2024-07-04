package main

import "net/http"

func (app *application) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	//define expected app health response data
	resEnv := envelope{
		"status": "online",
		"system_info": map[string]string{
			"environment": app.config.env,
			"version":     version,
		},
	}

	//pass data to our write json helper
	err := app.writeJSON(w, http.StatusOK, resEnv, nil)
	if err != nil {
		app.logger.PrintError(err, nil)

		//send back a server error response to the client
		app.serverErrorResponse(w, r, err)
		return
	}
}
