package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

func (c *Tester) server(streamer string, numberOf int) {
	var clients []client

	for i := 0; i < numberOf; i++ {
		// timestampWriter.WriteString("Client: " + strconv.Itoa(i) + "\n")
		c.egress <- []byte("N\n" + strconv.Itoa(i))
		cliOK := make(chan string)
		_, peerConnection := createWebRTCConn(c, streamer, i, cliOK)
		clients = append(clients, *newClient(peerConnection))

		// Wait for client to finish loading
		<-cliOK
		println("Ready: ", i)
		time.Sleep(time.Second * 1)
		// if *limitTimestamp {
		// 	timestampFile.Close()
		// 	timestampWriter.Flush()
		// }
	}

	println("All dummy clients ready")
	closed := make(chan os.Signal, 1)
	signal.Notify(closed, os.Interrupt)
	if 20 != 0 {
		go func() {
			time.Sleep(time.Second * 20)
			close(closed)
		}()
	}
	go func() {
		for {
			c.egress <- []byte("N\nN")
			time.Sleep(time.Second * 1)
		}
	}()

	<-closed

	for _, cl := range clients {
		if err := cl.peerCon.Close(); err != nil {
			c.egress <- []byte("<CK>Error")
		}
		// //Timestamp
		// if !*limitTimestamp {
		// 	c.timestampFile.Close()
		// 	c.timestampWriter.Flush()
		// }
	}
	time.Sleep(time.Second * 20)
}

type client struct {
	peerCon *webrtc.PeerConnection
}

func newClient(c *webrtc.PeerConnection) *client {
	return &client{
		peerCon: c,
	}
}

// func timestampWriter() (*os.File, *bufio.Writer) {
// 	// Set up a timestamp file
// 	new_file, new_file_err := os.OpenFile("timestamps.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if new_file_err != nil {
// 		log.Fatal(new_file_err)
// 	}
// 	// defer new_file.Close()
// 	writer := bufio.NewWriter(new_file)
// 	// defer writer.Flush()
// 	return new_file, writer
// }

func createWebRTCConn(c *Tester, streamer string, num int, cliOK chan string) (*WebsocketClient, *webrtc.PeerConnection) {
	var candidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	m := &webrtc.MediaEngine{}

	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/VP8", ClockRate: 90000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		c.egress <- []byte("<CK>Error")
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		c.egress <- []byte("<CK>Error")
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	// Create a new RTCPeerConnection
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		c.egress <- []byte("<CK>Error")
	}

	// Create a websocketclient
	ws := NewWebsocketClient(peerConnection)
	go ws.readMessages()
	go ws.writeMessages()

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			pendingCandidates = append(pendingCandidates, c)
		} else {
			payload, onICECandidateErr := json.Marshal(desc)
			if onICECandidateErr != nil {
			}
			ws.egress <- []byte("C\n" + ws.peerID + "\n" + string(payload))
		}
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		go func() {
			ticker := time.NewTicker(time.Second * 3)
			for range ticker.C {
				errSend := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
				if errSend != nil {
					fmt.Println(errSend)
				}
			}
		}()

		fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType(), track.Codec().RTPCodecCapability.MimeType)
		for {
			// Read RTP packets being sent to Pion
			_, _, readErr := track.ReadRTP()
			if readErr != nil {
				if readErr == io.EOF {
					return
				}
			}
			switch track.Kind() {
			case webrtc.RTPCodecTypeVideo:
				t := time.Now().UnixMilli()
				c.egress <- []byte(strconv.Itoa(num) + "\n" + strconv.FormatInt(t, 10))
			}
		}
	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed %s \n", connectionState.String())
	})

	peerConnection.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		if pcs == webrtc.PeerConnectionStateConnected {
			close(cliOK)
		}
	})
	ws.egress <- []byte("R\n" + streamer)

	return ws, peerConnection
}
