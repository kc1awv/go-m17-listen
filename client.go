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
	"context"
	"encoding/binary"
	"fmt"
	"go-m17-listen/codec2"
	"log"
	"net"
	"os"

	"github.com/hajimehoshi/oto"
)

// Packet MAGIC constants
const (
	MagicLSTN = "LSTN"
	MagicACKN = "ACKN"
	MagicNACK = "NACK"
	MagicPING = "PING"
	MagicPONG = "PONG"
	MagicDISC = "DISC"
	MagicM17  = "M17 "
)

// Client represents a M17 client
type Client struct {
	conn         *net.UDPConn
	callsign     string
	relayAddr    *net.UDPAddr
	moduleLetter byte
	codec2       *codec2.Codec2
	player       *oto.Player
	ctx          context.Context
	cancel       context.CancelFunc
	discChan     chan struct{}
}

// NewClient creates a new M17 client
func NewClient(callsign, relayAddr string, moduleLetter byte) (*Client, error) {
	// Resolve relay/reflector address
	addr, err := net.ResolveUDPAddr("udp", relayAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	// Dial UDP connection to relay/reflector
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	// Initialize Codec 2 at 3200 bps
	codec2, err := codec2.New(codec2.MODE_3200)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize codec2: %w", err)
	}

	// Initialize Oto player
	otoCtx, err := oto.NewContext(8000, 1, 2, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to create Oto context: %w", err)
	}
	player := otoCtx.NewPlayer()

	// Create context with cancel function
	ctx, cancel := context.WithCancel(context.Background())

	// Create and return new client
	return &Client{
		conn:         conn,
		callsign:     callsign,
		relayAddr:    addr,
		moduleLetter: moduleLetter,
		codec2:       codec2,
		player:       player,
		ctx:          ctx,
		cancel:       cancel,
		discChan:     make(chan struct{}),
	}, nil
}

// Listen listens for incoming packets
func (c *Client) listen() {
	buf := make([]byte, 64)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			n, addr, err := c.conn.ReadFromUDP(buf)
			if err != nil {
				if ne, ok := err.(*net.OpError); ok && ne.Err.Error() == "use of closed network connection" {
					return
				}
				log.Printf("failed to read from UDP: %v", err)
				updateTUI("Error", fmt.Sprintf("failed to read from UDP: %v", err))
				updateGUI("Error", fmt.Sprintf("failed to read from UDP: %v", err))
				continue
			}

			// Check if the packet is from the connected relay/reflector
			if !addr.IP.Equal(c.relayAddr.IP) || addr.Port != c.relayAddr.Port {
				log.Printf("received packet from unknown source: %v", addr)
				updateTUI("Error", fmt.Sprintf("received packet from unknown source: %v", addr))
				updateGUI("Error", fmt.Sprintf("received packet from unknown source: %v", addr))
				continue
			}

			c.handlePacket(buf[:n])
		}
	}
}

// sendLSTN sends a LSTN packet to the relay/reflector
func (c *Client) sendLSTN() error {
	encodedCallsign, err := encodeCallsign(c.callsign)
	if err != nil {
		return fmt.Errorf("failed to encode callsign: %w", err)
	}

	packet := append([]byte(MagicLSTN), encodedCallsign...)

	// Append module letter if present
	if c.moduleLetter != 0 {
		packet = append(packet, c.moduleLetter)
	}

	_, err = c.conn.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to send LSTN packet: %w", err)
	}

	return nil
}

// sendDISC sends a DISC packet to the relay/reflector
func (c *Client) sendDISC() error {
	encodedCallsign, err := encodeCallsign(c.callsign)
	if err != nil {
		return fmt.Errorf("failed to encode callsign: %w", err)
	}

	packet := append([]byte(MagicDISC), encodedCallsign...)
	_, err = c.conn.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to send DISC packet: %w", err)
	}

	return nil
}

// handlePacket handles incoming packets
func (c *Client) handlePacket(packet []byte) {
	if len(packet) < 4 {
		return
	}

	magic := string(packet[:4])
	switch magic {
	case MagicPING:
		c.handlePing()
	case MagicACKN:
		c.handleACKN()
	case MagicNACK:
		c.handleNACK()
	case MagicDISC:
		c.handleDISC()
	case MagicM17:
		c.handleM17(packet)
	}
}

// handlePing handles a PING packet
func (c *Client) handlePing() {
	encodedCallsign, err := encodeCallsign(c.callsign)
	if err != nil {
		log.Printf("failed to encode callsign: %v", err)
		updateTUI("Error", fmt.Sprintf("failed to encode callsign: %v", err))
		updateGUI("Error", fmt.Sprintf("failed to encode callsign: %v", err))
		return
	}

	pongPacket := append([]byte(MagicPONG), encodedCallsign...)
	_, err = c.conn.Write(pongPacket)
	if err != nil {
		log.Printf("failed to send PONG packet: %v", err)
		updateTUI("Error", fmt.Sprintf("failed to send PONG packet: %v", err))
		updateGUI("Error", fmt.Sprintf("failed to send PONG packet: %v", err))
	}
}

// handleACKN handles an ACKN packet
func (c *Client) handleACKN() {
	log.Println("Connection accepted by relay/reflector")
	updateTUI("Status", "Connection accepted by relay/reflector")
	updateGUI("Status", "Connection accepted by relay/reflector")
}

// handleNACK handles a NACK packet
func (c *Client) handleNACK() {
	log.Println("Connection not accepted by relay/reflector")
	updateTUI("Status", "Connection not accepted by relay/reflector")
	updateGUI("Status", "Connection not accepted by relay/reflector")
	c.sendDISC()
	c.cancel()
	c.conn.Close()
	c.player.Close()
	os.Exit(1)
}

// handleDISC handles a DISC packet
func (c *Client) handleDISC() {
	log.Println("Received DISC packet")
	updateTUI("Status", "Received DISC packet")
	updateGUI("Status", "Received DISC packet")
	close(c.discChan)
}

// handleM17 handles a M17 packet
func (c *Client) handleM17(packet []byte) {
	if len(packet) < 54 {
		log.Printf("invalid M17 packet length: %d", len(packet))
		updateTUI("Error", fmt.Sprintf("invalid M17 packet length: %d", len(packet)))
		updateGUI("Error", fmt.Sprintf("invalid M17 packet length: %d", len(packet)))
		return
	}

	// Parse M17 packet fields
	streamID := binary.BigEndian.Uint16(packet[4:6])
	lich := packet[6:34]
	frameNumber := binary.BigEndian.Uint16(packet[34:36])
	payload := packet[36:52]
	// reserved := packet[52:54] // Reserved field, not used

	// Parse LICH fields
	dst := decodeCallsign(lich[0:6])
	src := decodeCallsign(lich[6:12])
	typ := binary.BigEndian.Uint16(lich[12:14])
	meta := lich[14:28]

	// Parse Type field
	packetStreamIndicator := typ & 0x0001
	dataTypeIndicator := (typ >> 1) & 0x0003
	encryptionType := (typ >> 3) & 0x0003
	encryptionSubtype := (typ >> 5) & 0x0003
	channelAccessNumber := (typ >> 7) & 0x000F

	// Log packet fields
	log.Printf("Received M17 packet: StreamID=0x%X, FrameNumber=0x%X, DST=%s, SRC=%s, TYPE=0x%X, META=%x", streamID, frameNumber, dst, src, typ, meta)
	log.Printf("Type field breakdown: PacketStreamIndicator=%d, DataTypeIndicator=%d, EncryptionType=%d, EncryptionSubtype=%d, ChannelAccessNumber=%d",
		packetStreamIndicator, dataTypeIndicator, encryptionType, encryptionSubtype, channelAccessNumber)

	// Update TUI fields
	updateTUI("StreamID", fmt.Sprintf("%d", streamID))
	updateTUI("FrameNumber", fmt.Sprintf("%d", frameNumber))
	updateTUI("DST", dst)
	updateTUI("SRC", src)
	updateTUI("TYPE", fmt.Sprintf("%d", typ))
	updateTUI("META", fmt.Sprintf("%x", meta))
	updateTUI("Payload", fmt.Sprintf("%x", payload))
	updateTUI("PacketStreamIndicator", fmt.Sprintf("%d", packetStreamIndicator))
	updateTUI("DataTypeIndicator", fmt.Sprintf("%d", dataTypeIndicator))
	updateTUI("EncryptionType", fmt.Sprintf("%d", encryptionType))
	updateTUI("EncryptionSubtype", fmt.Sprintf("%d", encryptionSubtype))
	updateTUI("ChannelAccessNumber", fmt.Sprintf("%d", channelAccessNumber))

	// Update GUI with packet fields
	updateGUI("StreamID", fmt.Sprintf("%X", streamID))
	updateGUI("FrameNumber", fmt.Sprintf("%X", frameNumber))
	updateGUI("DST", dst)
	updateGUI("SRC", src)
	updateGUI("TYPE", fmt.Sprintf("%X", typ))
	updateGUI("META", fmt.Sprintf("%x", meta))
	updateGUI("PacketStreamIndicator", fmt.Sprintf("%d", packetStreamIndicator))
	updateGUI("DataTypeIndicator", fmt.Sprintf("%d", dataTypeIndicator))
	updateGUI("EncryptionType", fmt.Sprintf("%d", encryptionType))
	updateGUI("EncryptionSubtype", fmt.Sprintf("%d", encryptionSubtype))
	updateGUI("ChannelAccessNumber", fmt.Sprintf("%d", channelAccessNumber))
	updateGUI("Payload", fmt.Sprintf("%x", payload))

	// Filter out packets that are not stream mode or are encrypted
	if packetStreamIndicator == 0 || encryptionType != 0 {
		log.Printf("Ignoring packet mode or encrypted packet: TYPE=%d", typ)
		updateTUI("Status", fmt.Sprintf("Ignoring packet mode or encrypted packet: TYPE=%d", typ))
		updateGUI("Status", fmt.Sprintf("Ignoring packet mode or encrypted packet: TYPE=%d", typ))
		return
	}

	// Filter out packets that are not voice or voice + data
	if dataTypeIndicator != 0b10 && dataTypeIndicator != 0b11 {
		log.Printf("Ignoring non-voice packet: TYPE=%d", typ)
		updateTUI("Status", fmt.Sprintf("Ignoring non-voice packet: TYPE=%d", typ))
		updateGUI("Status", fmt.Sprintf("Ignoring non-voice packet: TYPE=%d", typ))
		return
	}

	// Ensure payload length is correct for Codec 2 at 3200 bps (16 bytes)
	if len(payload) != 16 {
		log.Printf("invalid payload length: %d", len(payload))
		updateTUI("Error", fmt.Sprintf("invalid payload length: %d", len(payload)))
		updateGUI("Error", fmt.Sprintf("invalid payload length: %d", len(payload)))
		return
	}

	// Decode and play the voice stream using Codec 2
	audio1, err := c.codec2.Decode(payload[:8])
	if err != nil {
		log.Printf("failed to decode first voice frame: %v", err)
		updateTUI("Error", fmt.Sprintf("failed to decode first voice frame: %v", err))
		updateGUI("Error", fmt.Sprintf("failed to decode first voice frame: %v", err))
		return
	}

	audio2, err := c.codec2.Decode(payload[8:])
	if err != nil {
		log.Printf("failed to decode second voice frame: %v", err)
		updateTUI("Error", fmt.Sprintf("failed to decode second voice frame: %v", err))
		updateGUI("Error", fmt.Sprintf("failed to decode second voice frame: %v", err))
		return
	}

	// Combine the two audio frames
	audio := append(audio1, audio2...)

	// Play the audio
	c.playAudio(audio)
}

// playAudio plays audio using the Oto player
func (c *Client) playAudio(audio []int16) {
	// Convert int16 audio to byte slice
	buf := make([]byte, len(audio)*2)
	for i, sample := range audio {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(sample))
	}

	// Write audio to player
	_, err := c.player.Write(buf)
	if err != nil {
		log.Printf("failed to play audio: %v", err)
		updateTUI("Error", fmt.Sprintf("failed to play audio: %v", err))
	}
}
