package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

const windowTitle = "Goedgepickt Orders Export"

func main() {
	a := app.New()
	w := a.NewWindow(windowTitle)
	w.SetContent(widget.NewLabel("Hello, world!"))
	w.ShowAndRun()
}
