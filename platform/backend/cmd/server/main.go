package main

import (
	"context"
	"log"

	"wfagent/internal/base/bootstrap"
)

// @title WFAgent Base API
// @version 0.1.0
// @description WFAgent technology finance system base API.
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer " followed by a valid access token.
func main() {
	app, err := bootstrap.New(".")
	if err != nil {
		log.Fatalf("bootstrap backend: %v", err)
	}

	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("run backend: %v", err)
	}
}
