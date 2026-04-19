package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
)

var csvHeader = []string{
	"ID",
	"Datumdistributiedag",
	"Naam",
	"Straatnaam",
	"Huisnummer",
	"Huisnummertoevoeging",
	"Postcode",
	"Plaatsnaam",
	"CountryCode",
	"TelefoonNummer",
	"Email",
	"BezoekenNa",
	"BezoekenVoor",
	"Locatieinstructie",
}

type CsvExporter interface {
	StartRun() error
	Append(CsvRow) error
	Close() error
}

type FileCSVExporter struct {
	filename string

	mu     sync.Mutex
	file   *os.File
	writer *csv.Writer
}

func NewFileCSVExporter(filename string) *FileCSVExporter {
	return &FileCSVExporter{filename: filename}
}

func (e *FileCSVExporter) StartRun() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.closeLocked(); err != nil {
		return err
	}

	file, err := os.Create(e.filename)
	if err != nil {
		return fmt.Errorf("kan %s niet aanmaken: %w", e.filename, err)
	}

	writer := csv.NewWriter(file)
	if err := writer.Write(csvHeader); err != nil {
		_ = file.Close()
		return fmt.Errorf("kan CSV-kop niet schrijven: %w", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = file.Close()
		return fmt.Errorf("kan CSV-kop niet wegschrijven: %w", err)
	}

	e.file = file
	e.writer = writer
	return nil
}

func (e *FileCSVExporter) Append(row CsvRow) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.writer == nil {
		return fmt.Errorf("CSV-export is niet gestart")
	}

	record := []string{
		row.ID,
		row.DatumDistributiedag,
		row.Naam,
		row.Straatnaam,
		row.Huisnummer,
		row.Huisnummertoevoeging,
		row.Postcode,
		row.Plaatsnaam,
		row.CountryCode,
		row.TelefoonNummer,
		row.Email,
		row.BezoekenNa,
		row.BezoekenVoor,
		row.Locatieinstructie,
	}

	if err := e.writer.Write(record); err != nil {
		return fmt.Errorf("kan CSV-rij niet schrijven: %w", err)
	}
	e.writer.Flush()
	if err := e.writer.Error(); err != nil {
		return fmt.Errorf("kan CSV-rij niet wegschrijven: %w", err)
	}

	return nil
}

func (e *FileCSVExporter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.closeLocked()
}

func (e *FileCSVExporter) closeLocked() error {
	var err error
	if e.writer != nil {
		e.writer.Flush()
		if writerErr := e.writer.Error(); writerErr != nil {
			err = fmt.Errorf("kan CSV-buffer niet flushen: %w", writerErr)
		}
		e.writer = nil
	}
	if e.file != nil {
		if closeErr := e.file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("kan CSV-bestand niet sluiten: %w", closeErr)
		}
		e.file = nil
	}

	return err
}
