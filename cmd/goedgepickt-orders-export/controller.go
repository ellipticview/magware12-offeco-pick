package main

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrDownloadAlreadyRunning = errors.New("download is al bezig")
	ErrTokenUnavailable       = errors.New("startup-token is niet beschikbaar")
)

type ControllerOptions struct {
	CSVFilename         string
	DownloadButtonLabel string
	Now                 func() time.Time
}

type Controller struct {
	mu          sync.Mutex
	tokenSource TokenSource
	orders      OrdersClient
	exporter    CsvExporter
	now         func() time.Time

	state        AppState
	startupToken string
	initialized  bool
	subscribers  []func(AppState)
}

func NewController(tokenSource TokenSource, orders OrdersClient, exporter CsvExporter, options ControllerOptions) *Controller {
	nowFn := options.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	csvFilename := options.CSVFilename
	if csvFilename == "" {
		csvFilename = defaultCSVFile
	}

	buttonLabel := options.DownloadButtonLabel
	if buttonLabel == "" {
		buttonLabel = defaultButtonText
	}

	controller := &Controller{
		tokenSource: tokenSource,
		orders:      orders,
		exporter:    exporter,
		now:         nowFn,
		state: AppState{
			CSVFilename:         csvFilename,
			DownloadButtonLabel: buttonLabel,
			TokenValidation:     TokenValidationPending,
			Run: DownloadRunState{
				Status: DownloadRunStatusIdle,
			},
		},
	}
	controller.recomputeButtonEnabledLocked()

	return controller
}

func (c *Controller) Initialize() error {
	c.mu.Lock()
	if c.initialized {
		c.mu.Unlock()
		return nil
	}
	c.initialized = true
	c.mu.Unlock()

	token, err := c.tokenSource.Load()
	if err != nil {
		c.mu.Lock()
		c.state.TokenValidation = TokenValidationInvalid
		c.appendMessageLocked(tokenValidationErrorMessage(err), ProgressMessageKindError)
		c.recomputeButtonEnabledLocked()
		state, subscribers := c.snapshotLocked()
		c.mu.Unlock()
		notifySubscribers(subscribers, state)
		return err
	}

	c.mu.Lock()
	c.startupToken = token
	c.state.TokenValidation = TokenValidationAvailable
	c.recomputeButtonEnabledLocked()
	state, subscribers := c.snapshotLocked()
	c.mu.Unlock()
	notifySubscribers(subscribers, state)
	return nil
}

func (c *Controller) Subscribe(fn func(AppState)) {
	c.mu.Lock()
	c.subscribers = append(c.subscribers, fn)
	state := cloneAppState(c.state)
	c.mu.Unlock()

	fn(state)
}

func (c *Controller) State() AppState {
	c.mu.Lock()
	defer c.mu.Unlock()

	return cloneAppState(c.state)
}

func (c *Controller) StartDownload() error {
	c.mu.Lock()
	if c.state.Run.Status == DownloadRunStatusInProgress {
		c.mu.Unlock()
		return ErrDownloadAlreadyRunning
	}
	if c.state.TokenValidation != TokenValidationAvailable {
		c.mu.Unlock()
		return ErrTokenUnavailable
	}

	c.resetRunLocked()
	state, subscribers := c.snapshotLocked()
	token := c.startupToken
	c.mu.Unlock()
	notifySubscribers(subscribers, state)

	if err := c.exporter.StartRun(); err != nil {
		c.failRun(err, false)
		return nil
	}

	startedAt := c.now()
	c.mu.Lock()
	c.state.Run.Status = DownloadRunStatusInProgress
	c.state.Run.StartedAt = &startedAt
	c.state.Run.FinishedAt = nil
	c.state.Run.HeaderWritten = true
	c.recomputeButtonEnabledLocked()
	state, subscribers = c.snapshotLocked()
	c.mu.Unlock()
	notifySubscribers(subscribers, state)

	go c.runDownload(token)
	return nil
}

func (c *Controller) resetRunLocked() {
	c.state.Run.Status = DownloadRunStatusIdle
	c.state.Run.StartedAt = nil
	c.state.Run.FinishedAt = nil
	c.state.Run.HeaderWritten = false
	c.state.Run.ExportedOrderCount = 0
	c.state.Run.NLExportedOrderCount = 0
	c.state.Run.BEExportedOrderCount = 0
	c.state.Run.Messages = nil
	c.state.Run.ExportedOrders = nil
	c.recomputeButtonEnabledLocked()
}

func (c *Controller) runDownload(token string) {
	err := c.orders.FetchReadyForPicking(context.Background(), token, func(order RemoteOrderData, sequenceNumber int) error {
		if !isQualifyingOrder(order) {
			return nil
		}

		row := mapCSVRow(order, distributionDate(c.now()))
		if err := c.exporter.Append(row); err != nil {
			return err
		}

		c.mu.Lock()
		c.state.Run.ExportedOrders = append(c.state.Run.ExportedOrders, ExportedOrder{
			SequenceNumber: sequenceNumber,
			SourceOrder:    order,
			CSVRow:         row,
		})
		c.state.Run.ExportedOrderCount++

		switch normalizedCountry(order.ShippingCountry) {
		case "NL":
			c.state.Run.NLExportedOrderCount++
		case "BE":
			c.state.Run.BEExportedOrderCount++
		}

		c.appendMessageLocked(exportProgressMessage(row.CountryCode, row.Plaatsnaam), ProgressMessageKindProgress)
		state, subscribers := c.snapshotLocked()
		c.mu.Unlock()
		notifySubscribers(subscribers, state)
		return nil
	})

	if closeErr := c.exporter.Close(); err == nil && closeErr != nil {
		err = closeErr
	}

	if err != nil {
		c.failRun(err, true)
		return
	}

	finishedAt := c.now()
	c.mu.Lock()
	c.state.Run.Status = DownloadRunStatusCompleted
	c.state.Run.FinishedAt = &finishedAt
	c.appendMessageLocked(
		completionSummaryMessage(
			c.state.Run.NLExportedOrderCount,
			c.state.Run.BEExportedOrderCount,
			c.state.Run.ExportedOrderCount,
		),
		ProgressMessageKindCompleted,
	)
	c.recomputeButtonEnabledLocked()
	state, subscribers := c.snapshotLocked()
	c.mu.Unlock()
	notifySubscribers(subscribers, state)
}

func (c *Controller) failRun(err error, started bool) {
	finishedAt := c.now()

	c.mu.Lock()
	if !started && c.state.Run.StartedAt == nil {
		startedAt := finishedAt
		c.state.Run.StartedAt = &startedAt
	}
	c.state.Run.Status = DownloadRunStatusFailed
	c.state.Run.FinishedAt = &finishedAt
	c.appendMessageLocked(downloadErrorMessage(err), ProgressMessageKindError)
	c.recomputeButtonEnabledLocked()
	state, subscribers := c.snapshotLocked()
	c.mu.Unlock()
	notifySubscribers(subscribers, state)
}

func (c *Controller) appendMessageLocked(text string, kind ProgressMessageKind) {
	c.state.Run.Messages = append(c.state.Run.Messages, ProgressMessage{
		SequenceNumber: len(c.state.Run.Messages) + 1,
		Text:           text,
		Kind:           kind,
	})
}

func (c *Controller) recomputeButtonEnabledLocked() {
	c.state.Run.DownloadButtonEnabled = c.state.Run.Status != DownloadRunStatusInProgress &&
		c.state.TokenValidation == TokenValidationAvailable
}

func (c *Controller) snapshotLocked() (AppState, []func(AppState)) {
	return cloneAppState(c.state), append([]func(AppState){}, c.subscribers...)
}

func notifySubscribers(subscribers []func(AppState), state AppState) {
	for _, subscriber := range subscribers {
		subscriber(state)
	}
}
