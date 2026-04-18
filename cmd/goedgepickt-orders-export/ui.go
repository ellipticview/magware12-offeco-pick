package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func buildMainWindow(a fyne.App, controller *Controller) fyne.Window {
	window := a.NewWindow(windowTitle)
	window.Resize(fyne.NewSize(720, 480))

	progressLog := widget.NewLabel("")
	progressLog.Wrapping = fyne.TextWrapWord
	progressScroll := container.NewScroll(progressLog)

	downloadButton := widget.NewButton(defaultButtonText, func() {
		_ = controller.StartDownload()
	})

	controller.Subscribe(func(state AppState) {
		fyne.Do(func() {
			downloadButton.SetText(state.DownloadButtonLabel)
			if state.Run.DownloadButtonEnabled {
				downloadButton.Enable()
			} else {
				downloadButton.Disable()
			}
			progressLog.SetText(orderedProgressText(state.Run.Messages))
		})
	})

	window.SetContent(container.NewPadded(container.NewBorder(
		container.NewVBox(downloadButton),
		nil,
		nil,
		nil,
		progressScroll,
	)))

	return window
}
