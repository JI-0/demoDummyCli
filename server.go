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

	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/rtcp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
)

var numberOf = 100

const spawnDelay = 1
const autoFinishTime = 20

func server() {
	var clients []client
	// timestampFile, timestampWriter := timestampWriter()
	timestamp()

	for i := 0; i < numberOf; i++ {
		// timestampWriter.WriteString("Client: " + strconv.Itoa(i) + "\n")
		chanel <- []byte("N," + strconv.Itoa(i) + "\n")
		cliOK := make(chan string)
		saver := newWebmSaver()
		_, peerConnection := createWebRTCConn(saver, i, cliOK)
		clients = append(clients, *newClient(saver, peerConnection))

		// Wait for client to finish loading
		<-cliOK
		println("Ready: ", i)
		time.Sleep(time.Second * spawnDelay)
		// if *limitTimestamp {
		// 	timestampFile.Close()
		// 	timestampWriter.Flush()
		// }
	}

	println("All dummy clients ready")
	closed := make(chan os.Signal, 1)
	signal.Notify(closed, os.Interrupt)
	if autoFinishTime != 0 {
		go func() {
			time.Sleep(time.Second * autoFinishTime)
			close(closed)
		}()
	}
	go func() {
		for {
			chanel <- []byte("N,NON\n")
			time.Sleep(time.Second * spawnDelay)
		}
	}()

	<-closed

	for _, c := range clients {
		if err := c.peerCon.Close(); err != nil {
			panic(err)
		}
		c.saver.Close()
		// //Timestamp
		// if !*limitTimestamp {
		// 	c.timestampFile.Close()
		// 	c.timestampWriter.Flush()
		// }
	}
	time.Sleep(time.Second * autoFinishTime)
}

type client struct {
	saver   *webmSaver
	peerCon *webrtc.PeerConnection
}

func newClient(s *webmSaver, c *webrtc.PeerConnection) *client {
	return &client{
		saver:   s,
		peerCon: c,
	}
}

type webmSaver struct {
	audioWriter, videoWriter       webm.BlockWriteCloser
	audioBuilder, videoBuilder     *samplebuilder.SampleBuilder
	audioTimestamp, videoTimestamp time.Duration
}

func newWebmSaver() *webmSaver {
	return &webmSaver{
		audioBuilder: samplebuilder.New(10, &codecs.OpusPacket{}, 48000),
		videoBuilder: samplebuilder.New(10, &codecs.VP8Packet{}, 90000),
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

func (s *webmSaver) Close() {
	fmt.Printf("Finalizing webm...\n")
	if s.audioWriter != nil {
		if err := s.audioWriter.Close(); err != nil {
			panic(err)
		}
	}
	if s.videoWriter != nil {
		if err := s.videoWriter.Close(); err != nil {
			panic(err)
		}
	}
}

func createWebRTCConn(saver *webmSaver, num int, cliOK chan string) (*WebsocketClient, *webrtc.PeerConnection) {
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
		panic(err)
	}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "audio/opus", ClockRate: 48000, Channels: 0, SDPFmtpLine: "", RTCPFeedback: nil},
		PayloadType:        111,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	// Create a new RTCPeerConnection
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		panic(err)
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
				panic(onICECandidateErr)
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
				// panic(readErr)
			}
			switch track.Kind() {
			case webrtc.RTPCodecTypeVideo:
				newTimestamp(num)
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
	ws.egress <- []byte("R\n" + "0")

	return ws, peerConnection
}

func newTimestamp(num int) {
	t := time.Now().UnixMilli()
	chanel <- []byte(strconv.Itoa(num) + "," + strconv.FormatInt(t, 10) + "\n")
}
