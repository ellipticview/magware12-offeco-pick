package main

import (
	"log"
	"net/http"
	"time"

	"fyne.io/fyne/v2/app"
)

const (
	windowTitle       = "Goedgepickt Orders Export"
	defaultAPIBaseURL = "https://account.goedgepickt.nl/api/v1/orders"
	defaultCSVFile    = "routigo-orders.csv"
	defaultTokenFile  = "token.txt"
	defaultButtonText = "Downloaden"
)

func main() {
	a := app.New()

	controller := NewController(
		NewTokenFileSource(defaultTokenFile),
		NewHTTPOrdersClient(defaultAPIBaseURL, &http.Client{Timeout: 30 * time.Second}),
		NewFileCSVExporter(defaultCSVFile),
		ControllerOptions{
			CSVFilename:         defaultCSVFile,
			DownloadButtonLabel: defaultButtonText,
			Now:                 time.Now,
		},
	)

	if err := controller.Initialize(); err != nil {
		log.Printf("startup initialisation returned an error: %v", err)
	}

	window := buildMainWindow(a, controller)
	window.ShowAndRun()
}
