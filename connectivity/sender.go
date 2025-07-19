package connectivity

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type WebRTCSender struct {
	peerConnection *webrtc.PeerConnection
	videoTrack     *webrtc.TrackLocalStaticSample
	audioTrack     *webrtc.TrackLocalStaticSample
	upgrader       websocket.Upgrader
	isConnected    bool
	mutex          sync.RWMutex

	// Channel for receiving video/audio data to send
	videoChannel chan []byte
	audioChannel chan []byte
	stopChannel  chan struct{}
}

func NewWebRTCSender() *WebRTCSender {
	return &WebRTCSender{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
		videoChannel: make(chan []byte, 100),
		audioChannel: make(chan []byte, 100),
		stopChannel:  make(chan struct{}),
	}
}

func (s *WebRTCSender) InitializePeerConnection() error {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %v", err)
	}
	s.peerConnection = pc

	// Create video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8},
		"video",
		"relay-video",
	)
	if err != nil {
		return fmt.Errorf("failed to create video track: %v", err)
	}
	s.videoTrack = videoTrack

	// Create audio track
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio",
		"relay-audio",
	)
	if err != nil {
		return fmt.Errorf("failed to create audio track: %v", err)
	}
	s.audioTrack = audioTrack

	// Add tracks to peer connection
	if _, err = pc.AddTrack(videoTrack); err != nil {
		return fmt.Errorf("failed to add video track: %v", err)
	}

	if _, err = pc.AddTrack(audioTrack); err != nil {
		return fmt.Errorf("failed to add audio track: %v", err)
	}

	// Handle connection state changes
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Sender connection state: %s\n", state.String())
		s.mutex.Lock()
		defer s.mutex.Unlock()

		switch state {
		case webrtc.PeerConnectionStateConnected:
			s.isConnected = true
			// Start streaming goroutines
			go s.streamVideo()
			go s.streamAudio()
		case webrtc.PeerConnectionStateDisconnected, webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			s.isConnected = false
		}
	})

	// Handle ICE connection state changes
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		fmt.Printf("Sender ICE connection state: %s\n", state.String())
	})

	return nil
}

func (s *WebRTCSender) streamVideo() {
	ticker := time.NewTicker(33 * time.Millisecond) // ~30 FPS
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChannel:
			return
		case videoData := <-s.videoChannel:
			if s.isConnected && s.videoTrack != nil {
				if err := s.videoTrack.WriteSample(webrtc.Sample{
					Data:     videoData,
					Duration: 33 * time.Millisecond,
				}); err != nil {
					fmt.Printf("Error writing video sample: %v\n", err)
				}
			}
		case <-ticker.C:
			// If no video data is available, continue to maintain frame rate
			if s.isConnected {
				// Could send a dummy frame or skip
			}
		}
	}
}

func (s *WebRTCSender) streamAudio() {
	ticker := time.NewTicker(20 * time.Millisecond) // 50 FPS audio
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChannel:
			return
		case audioData := <-s.audioChannel:
			if s.isConnected && s.audioTrack != nil {
				if err := s.audioTrack.WriteSample(webrtc.Sample{
					Data:     audioData,
					Duration: 20 * time.Millisecond,
				}); err != nil {
					fmt.Printf("Error writing audio sample: %v\n", err)
				}
			}
		case <-ticker.C:
			// If no audio data is available, continue
		}
	}
}

func (s *WebRTCSender) SendVideoFrame(data []byte) {
	select {
	case s.videoChannel <- data:
		// Successfully sent
	default:
		// Channel is full, drop frame
		fmt.Println("Video channel full, dropping frame")
	}
}

func (s *WebRTCSender) SendAudioFrame(data []byte) {
	select {
	case s.audioChannel <- data:
		// Successfully sent
	default:
		// Channel is full, drop frame
		fmt.Println("Audio channel full, dropping frame")
	}
}

func (s *WebRTCSender) HandleWebSocketConnection(w http.ResponseWriter, req *http.Request) {
	conn, err := s.upgrader.Upgrade(w, req, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}
	defer conn.Close()

	// Initialize peer connection if not already done
	if s.peerConnection == nil {
		if err := s.InitializePeerConnection(); err != nil {
			fmt.Printf("Failed to initialize peer connection: %v\n", err)
			return
		}
	}

	// Handle ICE candidates
	s.peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		candidateInit := candidate.ToJSON()
		if err := conn.WriteJSON(map[string]interface{}{
			"type":      "ice-candidate",
			"candidate": candidateInit,
		}); err != nil {
			fmt.Printf("Error sending ICE candidate: %v\n", err)
		}
	})

	// Listen for signaling messages
	for {
		var message map[string]interface{}
		if err := conn.ReadJSON(&message); err != nil {
			fmt.Printf("WebSocket read error: %v\n", err)
			break
		}

		switch message["type"] {
		case "offer":
			offer := webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  message["sdp"].(string),
			}

			if err := s.peerConnection.SetRemoteDescription(offer); err != nil {
				fmt.Printf("Error setting remote description: %v\n", err)
				continue
			}

			answer, err := s.peerConnection.CreateAnswer(nil)
			if err != nil {
				fmt.Printf("Error creating answer: %v\n", err)
				continue
			}

			if err := s.peerConnection.SetLocalDescription(answer); err != nil {
				fmt.Printf("Error setting local description: %v\n", err)
				continue
			}

			if err := conn.WriteJSON(map[string]interface{}{
				"type": "answer",
				"sdp":  answer.SDP,
			}); err != nil {
				fmt.Printf("Error sending answer: %v\n", err)
			}

		case "ice-candidate":
			candidateMap := message["candidate"].(map[string]interface{})
			sdpMLineIndex := uint16(candidateMap["sdpMLineIndex"].(float64))
			sdpMid := candidateMap["sdpMid"].(string)

			candidate := webrtc.ICECandidateInit{
				Candidate:     candidateMap["candidate"].(string),
				SDPMLineIndex: &sdpMLineIndex,
				SDPMid:        &sdpMid,
			}

			if err := s.peerConnection.AddICECandidate(candidate); err != nil {
				fmt.Printf("Error adding ICE candidate: %v\n", err)
			}
		}
	}
}

func (s *WebRTCSender) IsConnected() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isConnected
}

func (s *WebRTCSender) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	close(s.stopChannel)

	if s.peerConnection != nil {
		return s.peerConnection.Close()
	}
	return nil
}
