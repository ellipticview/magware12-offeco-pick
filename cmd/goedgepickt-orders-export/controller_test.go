package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestControllerInitializeAndStartupOnlyTokenLoad(t *testing.T) {
	tokenSource := &stubTokenSource{token: "startup-token"}
	orders := &stubOrdersClient{}
	exporter := &stubCSVExporter{}
	now := fixedNow()

	controller := NewController(tokenSource, orders, exporter, ControllerOptions{
		CSVFilename:         defaultCSVFile,
		DownloadButtonLabel: defaultButtonText,
		Now:                 now,
	})

	err := controller.Initialize()
	require.NoError(t, err)

	tokenSource.token = "changed-token"
	err = controller.StartDownload()
	require.NoError(t, err)
	orders.wait()

	state := controller.State()
	require.Equal(t, 1, tokenSource.calls)
	require.Equal(t, "startup-token", orders.receivedToken)
	require.Equal(t, TokenValidationAvailable, state.TokenValidation)
	require.True(t, state.Run.DownloadButtonEnabled)
}

func TestControllerInitializeFailureDisablesButton(t *testing.T) {
	tokenSource := &stubTokenSource{err: errors.New("bestand ontbreekt")}
	controller := NewController(tokenSource, &stubOrdersClient{}, &stubCSVExporter{}, ControllerOptions{
		Now: fixedNow(),
	})

	err := controller.Initialize()

	require.Error(t, err)
	state := controller.State()
	require.Equal(t, TokenValidationInvalid, state.TokenValidation)
	require.False(t, state.Run.DownloadButtonEnabled)
	require.Len(t, state.Run.Messages, 1)
	require.Contains(t, state.Run.Messages[0].Text, "bestand ontbreekt")
}

func TestControllerSuccessfulRunResetsAndCompletes(t *testing.T) {
	tokenSource := &stubTokenSource{token: "startup-token"}
	orders := &stubOrdersClient{
		orders: []RemoteOrderData{
			{
				ExternalDisplayID: "ORD-1",
				Status:            "ready_for_picking",
				ShippingCountry:   strPtr("NL"),
				ShippingCity:      strPtr("Amsterdam"),
			},
			{
				ExternalDisplayID: "ORD-2",
				Status:            "pending",
				ShippingCountry:   strPtr("NL"),
				ShippingCity:      strPtr("Rotterdam"),
			},
			{
				ExternalDisplayID: "ORD-3",
				Status:            "ready_for_picking",
				ShippingCountry:   strPtr("BE"),
				ShippingCity:      strPtr("Antwerpen"),
			},
		},
	}
	exporter := &stubCSVExporter{}
	controller := NewController(tokenSource, orders, exporter, ControllerOptions{
		Now: fixedNow(),
	})
	require.NoError(t, controller.Initialize())

	controller.mu.Lock()
	controller.state.Run.Messages = []ProgressMessage{{SequenceNumber: 1, Text: "oude melding", Kind: ProgressMessageKindProgress}}
	controller.state.Run.ExportedOrders = []ExportedOrder{{SequenceNumber: 99}}
	controller.state.Run.ExportedOrderCount = 7
	controller.mu.Unlock()

	err := controller.StartDownload()
	require.NoError(t, err)
	orders.wait()

	state := controller.State()
	require.Equal(t, DownloadRunStatusCompleted, state.Run.Status)
	require.True(t, state.Run.HeaderWritten)
	require.NotNil(t, state.Run.StartedAt)
	require.NotNil(t, state.Run.FinishedAt)
	require.True(t, state.Run.DownloadButtonEnabled)
	require.Equal(t, 2, state.Run.ExportedOrderCount)
	require.Equal(t, 1, state.Run.NLExportedOrderCount)
	require.Equal(t, 1, state.Run.BEExportedOrderCount)
	require.Len(t, exporter.rows, 2)
	require.Equal(t, "ORD-1", exporter.rows[0].ID)
	require.Equal(t, "ORD-3", exporter.rows[1].ID)
	require.Equal(t, []int{1, 3}, []int{state.Run.ExportedOrders[0].SequenceNumber, state.Run.ExportedOrders[1].SequenceNumber})
	require.Len(t, state.Run.Messages, 3)
	require.Equal(t, ProgressMessageKindCompleted, state.Run.Messages[2].Kind)
	require.Contains(t, state.Run.Messages[2].Text, "2 totaal")
}

func TestControllerFailureKeepsPartialCSV(t *testing.T) {
	tokenSource := &stubTokenSource{token: "startup-token"}
	orders := &stubOrdersClient{
		orders: []RemoteOrderData{
			{
				ExternalDisplayID: "ORD-1",
				Status:            "ready_for_picking",
				ShippingCountry:   strPtr("NL"),
				ShippingCity:      strPtr("Amsterdam"),
			},
		},
		err: errors.New("api kapot"),
	}
	exporter := &stubCSVExporter{}
	controller := NewController(tokenSource, orders, exporter, ControllerOptions{
		Now: fixedNow(),
	})
	require.NoError(t, controller.Initialize())

	err := controller.StartDownload()
	require.NoError(t, err)
	orders.wait()

	state := controller.State()
	require.Equal(t, DownloadRunStatusFailed, state.Run.Status)
	require.Equal(t, 1, state.Run.ExportedOrderCount)
	require.Len(t, exporter.rows, 1)
	require.Len(t, state.Run.Messages, 2)
	require.Equal(t, ProgressMessageKindError, state.Run.Messages[1].Kind)
	require.Contains(t, state.Run.Messages[1].Text, "api kapot")
}

func TestControllerStartRunFailure(t *testing.T) {
	tokenSource := &stubTokenSource{token: "startup-token"}
	exporter := &stubCSVExporter{startErr: errors.New("kan bestand niet maken")}
	controller := NewController(tokenSource, &stubOrdersClient{}, exporter, ControllerOptions{
		Now: fixedNow(),
	})
	require.NoError(t, controller.Initialize())

	err := controller.StartDownload()

	require.NoError(t, err)
	state := controller.State()
	require.Equal(t, DownloadRunStatusFailed, state.Run.Status)
	require.False(t, state.Run.HeaderWritten)
	require.Contains(t, state.Run.Messages[0].Text, "kan bestand niet maken")
}

type stubTokenSource struct {
	mu    sync.Mutex
	token string
	err   error
	calls int
}

func (s *stubTokenSource) Load() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return "", s.err
	}

	return s.token, nil
}

type stubOrdersClient struct {
	orders []RemoteOrderData
	err    error

	mu            sync.Mutex
	receivedToken string
	done          chan struct{}
}

func (s *stubOrdersClient) FetchReadyForPicking(ctx context.Context, token string, onOrder func(RemoteOrderData, int) error) error {
	s.mu.Lock()
	s.receivedToken = token
	if s.done == nil {
		s.done = make(chan struct{})
	}
	done := s.done
	s.mu.Unlock()
	defer close(done)

	for index, order := range s.orders {
		if err := onOrder(order, index+1); err != nil {
			return err
		}
	}

	if s.err != nil {
		return s.err
	}

	return nil
}

func (s *stubOrdersClient) wait() {
	for {
		s.mu.Lock()
		done := s.done
		s.mu.Unlock()
		if done != nil {
			<-done
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
}

type stubCSVExporter struct {
	mu        sync.Mutex
	started   bool
	closed    bool
	startErr  error
	appendErr error
	closeErr  error
	rows      []CsvRow
}

func (s *stubCSVExporter) StartRun() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return s.startErr
	}

	s.started = true
	s.closed = false
	s.rows = nil
	return nil
}

func (s *stubCSVExporter) Append(row CsvRow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.appendErr != nil {
		return s.appendErr
	}

	s.rows = append(s.rows, row)
	return nil
}

func (s *stubCSVExporter) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return s.closeErr
}

func fixedNow() func() time.Time {
	return func() time.Time {
		return time.Date(2026, 4, 18, 9, 0, 0, 0, time.UTC)
	}
}
