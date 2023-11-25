package main

import (
	"log"
	"math/rand"
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

const (
	letterBytes   = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
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

func getNewToken(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
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

	m.testers[tester] = "@" + getNewToken(64)
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
