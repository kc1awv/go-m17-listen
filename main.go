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
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nsf/termbox-go"
)

// main is the entry point of the program
func main() {
	// Parse command line arguments
	var useTUI bool
	var useGUI bool
	flag.BoolVar(&useTUI, "tui", false, "Enable TUI")
	flag.BoolVar(&useGUI, "gui", false, "Enable GUI")
	flag.Parse()

	if len(flag.Args()) < 1 || len(flag.Args()) > 2 {
		log.Fatalf("Usage: %s [--tui] [--gui] <address> [module_letter]", os.Args[0])
	}

	relayAddr := flag.Arg(0)
	var moduleLetter byte
	if len(flag.Args()) == 2 {
		moduleLetter = flag.Arg(1)[0]
	} else {
		moduleLetter = ' ' // Default to space character
	}

	// Generate random callsign
	callsign := generateRandomCallsign()

	// Initialize TUI
	if useTUI {
		err := termbox.Init()
		if err != nil {
			log.Fatalf("failed to initialize termbox: %v", err)
		}
		defer termbox.Close()

		// Redirect log output to io.Discard to disable logging to stdout
		log.SetOutput(io.Discard)

		go func() {
			for {
				switch ev := termbox.PollEvent(); ev.Type {
				case termbox.EventKey:
					if ev.Key == termbox.KeyCtrlC {
						return
					}
				case termbox.EventError:
					log.Printf("termbox error: %v", ev.Err)
				}
			}
		}()
	}

	if useGUI {
		go func() {
			// Redirect log output to io.Discard to disable logging to stdout
			log.SetOutput(io.Discard)

			client, err := NewClient(callsign, relayAddr, moduleLetter)
			if err != nil {
				log.Fatalf("failed to create client: %v", err)
			}
			err = client.sendLSTN()
			if err != nil {
				log.Fatalf("failed to send LSTN packet: %v", err)
			}
			go client.listen()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			select {
			case <-sigChan:
				log.Println("Shutting down client...")
			case <-func() chan struct{} {
				done := make(chan struct{})
				go func() {
					termbox.PollEvent()
					close(done)
				}()
				return done
			}():
				log.Println("GUI closed, shutting down client...")
			}
			client.sendDISC()
			client.cancel()
			select {
			case <-client.discChan:
				log.Println("Received DISC packet from relay, exiting...")
			case <-time.After(5 * time.Second):
				log.Println("Timeout waiting for DISC packet, exiting...")
			}
		}()
		startGUI()
	} else {
		client, err := NewClient(callsign, relayAddr, moduleLetter)
		if err != nil {
			log.Fatalf("failed to create client: %v", err)
		}
		err = client.sendLSTN()
		if err != nil {
			log.Fatalf("failed to send LSTN packet: %v", err)
		}
		go client.listen()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-sigChan:
			log.Println("Shutting down client...")
		case <-func() chan struct{} {
			done := make(chan struct{})
			go func() {
				termbox.PollEvent()
				close(done)
			}()
			return done
		}():
			log.Println("TUI closed, shutting down client...")
		}
		client.sendDISC()
		client.cancel()
		select {
		case <-client.discChan:
			log.Println("Received DISC packet from relay, exiting...")
		case <-time.After(5 * time.Second):
			log.Println("Timeout waiting for DISC packet, exiting...")
		}
	}
}
