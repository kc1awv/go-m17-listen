package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// guiLabels stores the GUI labels
var guiLabels map[string]*widget.Label

// startGUI starts the GUI
func startGUI() {
	// Create a new application
	a := app.New()
	a.Settings().SetTheme(&customTheme{})
	w := a.NewWindow("M17 Listen Client")

	// Initialize the map of GUI labels
	guiLabels = make(map[string]*widget.Label)

	// Create the GUI content
	content := container.NewVBox(
		widget.NewLabelWithStyle("M17 Listen Client", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
	)

	// Field names and their display names
	fields := map[string]string{
		"Status":                "Status",
		"StreamID":              "Stream ID",
		"FrameNumber":           "Frame Number",
		"DST":                   "Destination",
		"SRC":                   "Source",
		"TYPE":                  "Type",
		"META":                  "Metadata",
		"PacketStreamIndicator": "Packet Stream Indicator",
		"DataTypeIndicator":     "Data Type Indicator",
		"EncryptionType":        "Encryption Type",
		"EncryptionSubtype":     "Encryption Subtype",
		"ChannelAccessNumber":   "Channel Access Number",
		"Payload":               "Payload",
		"Error":                 "Error",
	}

	// Field order
	fieldOrder := []string{
		"Status", "StreamID", "FrameNumber", "DST", "SRC", "TYPE", "META",
		"PacketStreamIndicator", "DataTypeIndicator", "EncryptionType",
		"EncryptionSubtype", "ChannelAccessNumber", "Payload", "Error",
	}

	// Create a grid to display the fields
	grid := container.NewGridWithColumns(2)
	for _, field := range fieldOrder {
		displayName := fields[field]
		label := widget.NewLabelWithStyle(displayName+":", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
		value := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
		if field == "Error" {
			value.SetText("None")
		}
		guiLabels[field] = value
		grid.Add(label)
		grid.Add(value)
	}

	// Add the grid to the content
	content.Add(grid)

	// Set the content and show the window
	w.SetContent(content)
	w.Resize(fyne.NewSize(400, 400))
	w.ShowAndRun()
}

// updateGUI updates the GUI field with the given value
func updateGUI(field, status string) {
	if label, ok := guiLabels[field]; ok {
		if field == "StreamID" || field == "FrameNumber" || field == "TYPE" {
			label.SetText(fmt.Sprintf("0x%s", status))
		} else {
			label.SetText(status)
		}
	}
}
