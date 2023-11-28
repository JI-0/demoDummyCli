package main

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	pongWait     = 10 * time.Second
	pingInterval = (pongWait * 8) / 10
)

// Testers map with Tester: identifier
type TesterList map[*Tester]bool

type Tester struct {
	connection *websocket.Conn
	manager    *Manager
	// Egress to prevent concurrent writes to the websocket conn
	egress chan []byte
}

func NewTester(conn *websocket.Conn, manager *Manager) *Tester {
	return &Tester{
		connection: conn,
		manager:    manager,
		egress:     make(chan []byte),
	}
}

func (c *Tester) readMessages() {
	defer func() {
		// Cleanup connection
		c.manager.removeTester(c)
	}()

	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		// log.Println(err)
		return
	}

	c.connection.SetReadLimit(1024)

	c.connection.SetPongHandler(c.pongHandler)

	for {
		messageType, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// log.Println("Read error ", err)
			}
			break
		}

		// Check if text
		if messageType == 1 || messageType == 2 {
			message := string(payload)
			payloadParts := strings.Split(message, "\n")
			if len(payloadParts) != 3 {
				break
			}
			if payloadParts[0] != "R" {
				break
			}
			numOfDummies, err := strconv.Atoi(payloadParts[2])
			if err != nil {
				break
			}
			c.server(payloadParts[1], numOfDummies)
		}
	}
}

func (c *Tester) writeMessages() {
	ticker := time.NewTicker(pingInterval)

	defer func() {
		ticker.Stop()
		// Cleanup connection
		c.manager.removeTester(c)
	}()

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(3000, "Connection lost unexpectedly")); err != nil {
					// log.Println("Closed connection: ", err)
				}
				return
			}

			if string(message) == "<CK>OK" {
				if err := c.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1000, "")); err != nil {
				}
				return
			} else if strings.HasPrefix(string(message), "<CK>") {
				if err := c.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(3000, strings.TrimPrefix(string(message), "<CK>"))); err != nil {
				}
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Failed to send message: %v", err)
			}

		case <-ticker.C:
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Tester) pongHandler(pongMsg string) error {
	return c.connection.SetReadDeadline(time.Now().Add(pongWait))
}
