/*
Copyright (C) 2024 Steve Miller KC1AWV

This program is free software: you can redistribute it and/or modify it
under the terms of the GNU General Public License as published by the Free
Software Foundation, either version 3 of the License, or (at your option)
any later version.

This program is distributed in the hope that it will be useful, but WITHOUT
ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for
more details.

You should have received a copy of the GNU General Public License along with
this program. If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"fmt"
	"strconv"

	"github.com/nsf/termbox-go"
)

// tuiData stores the data to be displayed in the TUI
var tuiData = map[string]string{
	"StreamID":              "",
	"FrameNumber":           "",
	"DST":                   "",
	"SRC":                   "",
	"TYPE":                  "",
	"META":                  "",
	"PacketStreamIndicator": "",
	"DataTypeIndicator":     "",
	"EncryptionType":        "",
	"EncryptionSubtype":     "",
	"ChannelAccessNumber":   "",
	"Payload":               "",
	"Status":                "",
	"Error":                 "",
}

// updateTUI updates the TUI field with the given value
func updateTUI(field, value string) {
	switch field {
	case "StreamID", "FrameNumber", "TYPE":
		// Convert the value to hexadecimal
		if intValue, err := strconv.Atoi(value); err == nil {
			tuiData[field] = fmt.Sprintf("0x%X", intValue)
		} else {
			tuiData[field] = value // Fallback to the original value if conversion fails
		}
	default:
		tuiData[field] = value
	}
	drawTUI()
}

// fieldDisplayNames maps field names to their display names
var fieldDisplayNames = map[string]string{
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
	"Status":                "Status",
	"Error":                 "Error",
}

// drawTUI draws the TUI
func drawTUI() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	tbprint(0, 0, termbox.ColorDefault, termbox.ColorDefault, "M17 Listen Client")
	tbprint(0, 1, termbox.ColorDefault, termbox.ColorDefault, "") // Blank line
	y := 2
	for _, key := range []string{
		"StreamID", "FrameNumber", "DST", "SRC", "TYPE", "META",
		"PacketStreamIndicator", "DataTypeIndicator", "EncryptionType",
		"EncryptionSubtype", "ChannelAccessNumber", "Payload",
		"Status", "Error",
	} {
		displayName := fieldDisplayNames[key]
		tbprint(0, y, termbox.ColorDefault, termbox.ColorDefault, displayName+":")
		tbprint(26, y, termbox.ColorDefault, termbox.ColorDefault, tuiData[key])
		y++
	}
	termbox.Flush()
}

// tbprint prints a message to the TUI at the given coordinates
func tbprint(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x++
	}
}
