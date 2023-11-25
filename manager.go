package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin:     checkOrigin,
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}
)

type Manager struct {
	testers TesterList
	sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		testers: make(TesterList),
	}
}

func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {
	log.Println("Got something")

	// Upgrade from http
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Create new client
	tester := NewTester(conn, m)
	m.addTester(tester)

	// Start client process goroutines
	go tester.readMessages()
	go tester.writeMessages()
}

func (m *Manager) addTester(tester *Tester) {
	m.Lock()
	defer m.Unlock()

	m.testers[tester] = true
}

func (m *Manager) removeTester(tester *Tester) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.testers[tester]; ok {
		tester.connection.Close()
		delete(m.testers, tester)
	}
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	switch origin {
	case "test.com":
		return false
	default:
		return true
	}
}
