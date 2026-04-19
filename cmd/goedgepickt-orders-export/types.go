package main

import "time"

type DownloadRunStatus string

const (
	DownloadRunStatusIdle       DownloadRunStatus = "idle"
	DownloadRunStatusInProgress DownloadRunStatus = "in_progress"
	DownloadRunStatusCompleted  DownloadRunStatus = "completed"
	DownloadRunStatusFailed     DownloadRunStatus = "failed"
)

type TokenValidationStatus string

const (
	TokenValidationPending   TokenValidationStatus = "pending"
	TokenValidationAvailable TokenValidationStatus = "available"
	TokenValidationInvalid   TokenValidationStatus = "invalid"
)

type ProgressMessageKind string

const (
	ProgressMessageKindProgress  ProgressMessageKind = "progress"
	ProgressMessageKindCompleted ProgressMessageKind = "completed"
	ProgressMessageKindError     ProgressMessageKind = "error"
)

type RemoteOrderData struct {
	ExternalDisplayID           string
	Status                      string
	BillingAddress              *string
	BillingPhone                *string
	BillingEmail                *string
	ShippingFirstName           *string
	ShippingLastName            *string
	ShippingHouseNumber         *string
	ShippingHouseNumberAddition *string
	ShippingAddress2            *string
	ShippingZipcode             *string
	ShippingCity                *string
	ShippingCountry             *string
	CustomerNote                *string
}

type CsvRow struct {
	ID                   string
	DatumDistributiedag  string
	Naam                 string
	Straatnaam           string
	Huisnummer           string
	Huisnummertoevoeging string
	Postcode             string
	Plaatsnaam           string
	CountryCode          string
	TelefoonNummer       string
	Email                string
	BezoekenNa           string
	BezoekenVoor         string
	Locatieinstructie    string
}

type ProgressMessage struct {
	SequenceNumber int
	Text           string
	Kind           ProgressMessageKind
}

type ExportedOrder struct {
	SequenceNumber int
	SourceOrder    RemoteOrderData
	CSVRow         CsvRow
}

type DownloadRunState struct {
	Status                DownloadRunStatus
	StartedAt             *time.Time
	FinishedAt            *time.Time
	HeaderWritten         bool
	ExportedOrderCount    int
	NLExportedOrderCount  int
	BEExportedOrderCount  int
	Messages              []ProgressMessage
	ExportedOrders        []ExportedOrder
	DownloadButtonEnabled bool
}

type AppState struct {
	CSVFilename         string
	DownloadButtonLabel string
	TokenValidation     TokenValidationStatus
	Run                 DownloadRunState
}

func cloneAppState(state AppState) AppState {
	cloned := state
	cloned.Run.Messages = append([]ProgressMessage(nil), state.Run.Messages...)
	cloned.Run.ExportedOrders = append([]ExportedOrder(nil), state.Run.ExportedOrders...)
	return cloned
}
