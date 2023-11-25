package main

import (
	"log"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type WebsocketClient struct {
	connection     *websocket.Conn
	peerConnection *webrtc.PeerConnection
	peerID         string
	// Egress to prevent concurrent writes to the websocket conn
	egress chan []byte
}

func NewWebsocketClient(p *webrtc.PeerConnection) *WebsocketClient {
	u := url.URL{Scheme: "wss", Host: "localhost:3000", Path: "/"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	return &WebsocketClient{
		connection:     c,
		peerConnection: p,
		peerID:         "",
		egress:         make(chan []byte),
	}
}

func (c *WebsocketClient) readMessages() {
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
			if !parseMessage(c, message) {
				break
			}
		}
	}
}

func (c *WebsocketClient) writeMessages() {
	defer c.connection.Close()

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
			// log.Println("Message sent: ", string(message))
		}
	}
}
