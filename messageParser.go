package main

import (
	"encoding/json"
	"strings"

	"github.com/pion/webrtc/v4"
)

func parseMessage(c *WebsocketClient, msg string) bool {
	parts := strings.Split(msg, "\n")
	// log.Println(parts)
	switch parts[0] {
	case "O":
		if len(parts) == 3 {
			return processOffer(c, parts[1], parts[2])
		}
	case "C":
		if len(parts) == 3 {
			return processCandidate(c, parts[1], parts[2])
		}
	}
	// No function match
	return false
}

func processOffer(c *WebsocketClient, id string, offer string) bool {
	// Set the remote SessionDescription
	sdp := webrtc.SessionDescription{}
	if err := json.Unmarshal([]byte(offer), &sdp); err != nil {
		panic(err)
	}
	if err := c.peerConnection.SetRemoteDescription(sdp); err != nil {
		panic(err)
	}

	// Create an answer
	answer, err := c.peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	payload, err := json.Marshal(answer)
	if err != nil {
		panic(err)
	}

	c.peerConnection.SetLocalDescription(answer)

	c.peerID = id
	c.egress <- []byte("A\n" + id + "\n" + string(payload))
	return true
}

func processCandidate(c *WebsocketClient, id string, candidate string) bool {
	can := webrtc.ICECandidateInit{}
	if err := json.Unmarshal([]byte(candidate), &can); err != nil {
		panic(err)
	}
	if candidateErr := c.peerConnection.AddICECandidate(can); candidateErr != nil {
		panic(candidateErr)
	}

	return true
}
