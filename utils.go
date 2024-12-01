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
	"math/rand"
	"time"
)

// base40Chars is the character set used for encoding callsigns
const (
	base40Chars = " ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-/."
)

// encodeCallsign encodes a callsign into a 6-byte address
func encodeCallsign(callsign string) ([]byte, error) {
	address := uint64(0)

	for i := len(callsign) - 1; i >= 0; i-- {
		c := callsign[i]
		val := 0
		switch {
		case c == ' ':
			val = 0
		case 'A' <= c && c <= 'Z':
			val = int(c-'A') + 1
		case '0' <= c && c <= '9':
			val = int(c-'0') + 27
		case c == '-':
			val = 37
		case c == '/':
			val = 38
		case c == '.':
			val = 39
		default:
			return nil, fmt.Errorf("invalid character in callsign: %c", c)
		}

		address = address*40 + uint64(val)
	}

	result := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		result[i] = byte(address & 0xFF)
		address >>= 8
	}

	return result, nil
}

// decodeCallsign decodes a 6-byte address into a callsign
func decodeCallsign(encoded []byte) string {
	address := uint64(0)

	for _, b := range encoded {
		address = address*256 + uint64(b)
	}

	callsign := ""
	for address > 0 {
		idx := address % 40
		callsign += string(base40Chars[idx])
		address /= 40
	}

	return callsign
}

// generateRandomCallsign generates a random callsign
func generateRandomCallsign() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 5)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return "LSTN" + string(b)
}
