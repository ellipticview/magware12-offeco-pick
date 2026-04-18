package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFileCSVExporterWritesHeaderAndRows(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "routigo-orders.csv")
	exporter := NewFileCSVExporter(filename)

	require.NoError(t, exporter.StartRun())
	require.NoError(t, exporter.Append(mapCSVRow(RemoteOrderData{
		ExternalDisplayID: "ORD-1",
		Status:            "ready_for_picking",
		BillingAddress:    strPtr("Hoofdstraat"),
		BillingPhone:      strPtr("0612345678"),
		BillingEmail:      strPtr("test@example.com"),
		ShippingFirstName: strPtr("Ada"),
		ShippingLastName:  strPtr("Lovelace"),
		ShippingCountry:   strPtr("nl"),
		ShippingCity:      strPtr("Amsterdam"),
		CustomerNote:      strPtr("Bel aan"),
	}, distributionDate(time.Date(2026, 4, 18, 9, 0, 0, 0, time.UTC)))))
	require.NoError(t, exporter.Close())

	file, err := os.Open(filename)
	require.NoError(t, err)
	defer file.Close()

	rows, err := csv.NewReader(file).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, csvHeader, rows[0])
	require.Equal(t, []string{
		"ORD-1",
		"2026-04-18",
		"Ada Lovelace ORD-1",
		"Hoofdstraat",
		"",
		"",
		"",
		"Amsterdam",
		"NL",
		"0612345678",
		"test@example.com",
		"",
		"",
		"Bel aan",
	}, rows[1])
}

func TestMapCSVRowFallbacks(t *testing.T) {
	row := mapCSVRow(RemoteOrderData{
		ExternalDisplayID:           "ORD-2",
		Status:                      "ready_for_picking",
		ShippingHouseNumber:         strPtr("10"),
		ShippingHouseNumberAddition: strPtr("A"),
		ShippingZipcode:             strPtr("1234AB"),
		ShippingCity:                strPtr("Utrecht"),
		ShippingCountry:             strPtr("BE"),
	}, "2026-04-18")

	require.Equal(t, "ORD-2", row.Naam)
	require.Equal(t, "10", row.Huisnummer)
	require.Equal(t, "A", row.Huisnummertoevoeging)
	require.Equal(t, "1234AB", row.Postcode)
	require.Equal(t, "Utrecht", row.Plaatsnaam)
	require.Equal(t, "BE", row.CountryCode)
	require.Equal(t, "", row.BezoekenNa)
	require.Equal(t, "", row.BezoekenVoor)
}

func TestMapCSVRowNaamAppendsID(t *testing.T) {
	row := mapCSVRow(RemoteOrderData{
		ExternalDisplayID: "ORD-3",
		Status:            "ready_for_picking",
		ShippingFirstName: strPtr("Grace"),
		ShippingLastName:  strPtr("Hopper"),
	}, "2026-04-18")

	require.Equal(t, "Grace Hopper ORD-3", row.Naam)
}

func TestFileCSVExporterOverwritesFilePerRun(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "routigo-orders.csv")
	exporter := NewFileCSVExporter(filename)

	require.NoError(t, exporter.StartRun())
	require.NoError(t, exporter.Append(CsvRow{ID: "first"}))
	require.NoError(t, exporter.Close())

	require.NoError(t, exporter.StartRun())
	require.NoError(t, exporter.Append(CsvRow{ID: "second"}))
	require.NoError(t, exporter.Close())

	content, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.NotContains(t, string(content), "first")
	require.True(t, strings.Contains(string(content), "second"))
}

func strPtr(value string) *string {
	return &value
}
