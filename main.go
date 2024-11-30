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
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-m17-listen/codec2"

	"github.com/hajimehoshi/oto"
)

const (
	MagicLSTN   = "LSTN"
	MagicACKN   = "ACKN"
	MagicNACK   = "NACK"
	MagicPING   = "PING"
	MagicPONG   = "PONG"
	MagicDISC   = "DISC"
	MagicM17    = "M17 "
	base40Chars = " ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-/."
)

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

func NewClient(callsign, relayAddr string, moduleLetter byte) (*Client, error) {
	addr, err := net.ResolveUDPAddr("udp", relayAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve relay address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial relay: %w", err)
	}

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

	ctx, cancel := context.WithCancel(context.Background())

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

func (c *Client) sendLSTN() error {
	encodedCallsign, err := EncodeCallsign(c.callsign)
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

func (c *Client) sendDISC() error {
	encodedCallsign, err := EncodeCallsign(c.callsign)
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

func (c *Client) handlePing() {
	encodedCallsign, err := EncodeCallsign(c.callsign)
	if err != nil {
		log.Printf("failed to encode callsign: %v", err)
		return
	}

	pongPacket := append([]byte(MagicPONG), encodedCallsign...)
	_, err = c.conn.Write(pongPacket)
	if err != nil {
		log.Printf("failed to send PONG packet: %v", err)
	}
}

func (c *Client) handleACKN() {
	log.Println("Connection accepted by relay")
}

func (c *Client) handleNACK() {
	log.Println("Connection not accepted by relay")
	c.sendDISC()
	c.cancel()
	c.conn.Close()
	c.player.Close()
	os.Exit(1)
}

func (c *Client) handleDISC() {
	log.Println("Received DISC packet")
	close(c.discChan)
}

func (c *Client) handleM17(packet []byte) {
	if len(packet) < 54 {
		log.Printf("invalid M17 packet length: %d", len(packet))
		return
	}

	// Parse M17 packet fields
	streamID := binary.BigEndian.Uint16(packet[4:6])
	lich := packet[6:34]
	frameNumber := binary.BigEndian.Uint16(packet[34:36])
	payload := packet[36:52]
	// reserved := packet[52:54] // Reserved field, not used

	// Parse LICH fields
	dst := DecodeCallsign(lich[0:6])
	src := DecodeCallsign(lich[6:12])
	typ := binary.BigEndian.Uint16(lich[12:14])
	meta := lich[14:28]

	// Parse Type field
	packetStreamIndicator := typ & 0x0001
	dataTypeIndicator := (typ >> 1) & 0x0003
	encryptionType := (typ >> 3) & 0x0003
	encryptionSubtype := (typ >> 5) & 0x0003
	channelAccessNumber := (typ >> 7) & 0x000F

	log.Printf("Received M17 packet: StreamID=%d, FrameNumber=%d, DST=%s, SRC=%s, TYPE=%d, META=%x", streamID, frameNumber, dst, src, typ, meta)
	log.Printf("Type field breakdown: PacketStreamIndicator=%d, DataTypeIndicator=%d, EncryptionType=%d, EncryptionSubtype=%d, ChannelAccessNumber=%d",
		packetStreamIndicator, dataTypeIndicator, encryptionType, encryptionSubtype, channelAccessNumber)

	// Filter out packets that are not stream mode or are encrypted
	if packetStreamIndicator == 0 || encryptionType != 0 {
		log.Printf("Ignoring packet mode or encrypted packet: TYPE=%d", typ)
		return
	}

	// Filter out packets that are not voice or voice + data
	if dataTypeIndicator != 0b10 && dataTypeIndicator != 0b11 {
		log.Printf("Ignoring non-voice packet: TYPE=%d", typ)
		return
	}

	// Ensure payload length is correct for Codec 2 at 3200 bps (16 bytes)
	if len(payload) != 16 {
		log.Printf("invalid payload length: %d", len(payload))
		return
	}

	// Decode and play the voice stream using Codec 2
	audio1, err := c.codec2.Decode(payload[:8])
	if err != nil {
		log.Printf("failed to decode first voice frame: %v", err)
		return
	}

	audio2, err := c.codec2.Decode(payload[8:])
	if err != nil {
		log.Printf("failed to decode second voice frame: %v", err)
		return
	}

	// Combine the two audio frames
	audio := append(audio1, audio2...)

	// Play the audio
	c.playAudio(audio)
}

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
	}
}

func EncodeCallsign(callsign string) ([]byte, error) {
	address := uint64(0)

	for i := len(callsign) - 1; i >= 0; i-- {
		c := callsign[i]
		val := 0
		switch {
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

func DecodeCallsign(encoded []byte) string {
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

func (c *Client) listen() {
	buf := make([]byte, 1024)
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			n, _, err := c.conn.ReadFromUDP(buf)
			if err != nil {
				if ne, ok := err.(*net.OpError); ok && ne.Err.Error() == "use of closed network connection" {
					return
				}
				log.Printf("failed to read from UDP: %v", err)
				continue
			}

			c.handlePacket(buf[:n])
		}
	}
}

func generateRandomCallsign() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 5)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return "LSTN" + string(b)
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <relay_address> <module_letter>", os.Args[0])
	}

	relayAddr := os.Args[1]
	moduleLetter := os.Args[2][0]
	callsign := generateRandomCallsign()

	client, err := NewClient(callsign, relayAddr, moduleLetter)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	err = client.sendLSTN()
	if err != nil {
		log.Fatalf("failed to send LSTN packet: %v", err)
	}

	go client.listen()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down client...")
	client.sendDISC()
	client.cancel()

	// Wait for DISC packet
	select {
	case <-client.discChan:
		log.Println("Received DISC packet, closing connection")
	case <-time.After(5 * time.Second):
		log.Println("Timeout waiting for DISC packet, closing connection")
	}

	client.conn.Close()
	client.player.Close()
}