package senders

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	ext_webrtc "github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type WebRTCSender struct {
	mutex       sync.RWMutex
	isConnected bool

	// WebSocket
	upgrader            websocket.Upgrader
	websocketConnection *websocket.Conn

	// WebRRTC
	webrtcConnection *ext_webrtc.PeerConnection
	videoTrack       *ext_webrtc.TrackLocalStaticSample
	audioTrack       *ext_webrtc.TrackLocalStaticSample

	// Channels for receiving video and audio data to send
	videoChannel chan []byte
	audioChannel chan []byte
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
	}
}

func (s *WebRTCSender) SetIsConnected(isConnected bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.isConnected = isConnected
}

func (r *WebRTCSender) Start() error {
	err := r.initializePeerConnection()
	if err != nil {
		return err
	}

	err = r.createVideoTrack()
	if err != nil {
		return err
	}

	err = r.createAudioTrack()
	if err != nil {
		return err
	}

	return nil
}

// Sends data from the receiver to the sender's internal video channel. Once the peer connection is established,
// the video data will be sent to the peer connection's video track.
func (s *WebRTCSender) SendVideoFrame(data []byte) {
	select {
	case s.videoChannel <- data:
		// Successfully sent to channel
	default:
		// Channel is full, drop frame
		fmt.Println("⚠️ Video channel full, dropping frame")
	}
}

// Sends data from the receiver to the sender's internal audio channel. Once the peer connection is established,
// the audio data will be sent to the peer connection's audio track.
func (s *WebRTCSender) SendAudioFrame(data []byte) {
	select {
	case s.audioChannel <- data:
		// Successfully sent
	default:
		// Channel is full, drop frame
		fmt.Println("Audio channel full, dropping frame")
	}
}

// Upgrades the WebSocket connection and initializes the peer connection if not already done.
// Adds handlers for ICE candidates and signaling messages.
func (s *WebRTCSender) HandleWebSocketConnection(w http.ResponseWriter, req *http.Request) {
	if s.webrtcConnection == nil {
		panic("Failed to start peer connection")
	}

	upgradedConnection, err := s.upgrader.Upgrade(w, req, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
	}
	defer upgradedConnection.Close()
	s.websocketConnection = upgradedConnection

	// Listen for signaling messages
	for {
		var message map[string]interface{}
		if err := s.websocketConnection.ReadJSON(&message); err != nil {
			break
		}

		switch message["type"] {
		case "offer":
			offer := ext_webrtc.SessionDescription{
				Type: ext_webrtc.SDPTypeOffer,
				SDP:  message["sdp"].(string),
			}

			if err := s.webrtcConnection.SetRemoteDescription(offer); err != nil {
				fmt.Printf("Error setting remote description: %v\n", err)
				continue
			}

			answer, err := s.webrtcConnection.CreateAnswer(nil)
			if err != nil {
				fmt.Printf("Error creating answer: %v\n", err)
				continue
			}

			if err := s.webrtcConnection.SetLocalDescription(answer); err != nil {
				fmt.Printf("Error setting local description: %v\n", err)
				continue
			}

			if err := s.websocketConnection.WriteJSON(map[string]interface{}{
				"type": "answer",
				"sdp":  answer.SDP,
			}); err != nil {
				fmt.Printf("Error sending answer: %v\n", err)
			}

		case "ice-candidate":
			candidateMap := message["candidate"].(map[string]interface{})
			sdpMLineIndex := uint16(candidateMap["sdpMLineIndex"].(float64))
			sdpMid := candidateMap["sdpMid"].(string)

			candidate := ext_webrtc.ICECandidateInit{
				Candidate:     candidateMap["candidate"].(string),
				SDPMLineIndex: &sdpMLineIndex,
				SDPMid:        &sdpMid,
			}

			if err := s.webrtcConnection.AddICECandidate(candidate); err != nil {
				fmt.Printf("Error adding ICE candidate: %v\n", err)
			}
		default:
			fmt.Printf("Unknown message type: %v\n", message["type"])
			return
		}
	}
}

func (s *WebRTCSender) IsConnected() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isConnected
}

func (s *WebRTCSender) Close() error {

	if s.webrtcConnection != nil {
		return s.webrtcConnection.Close()
	}

	if s.websocketConnection != nil {
		return s.websocketConnection.Close()
	}

	return nil
}

// Initializes WebRTC peer connection. Adds listener for connection state change, ICE connection state change, and ICE gathering state change.
// Starts streaming video and audio in separate goroutines, if connection is established via PeerConnectionStateConnected.
func (s *WebRTCSender) initializePeerConnection() error {
	config := ext_webrtc.Configuration{
		ICEServers: []ext_webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun.relay.metered.ca:80",
					"turn:global.relay.metered.ca:80",
					"turn:global.relay.metered.ca:80?transport=tcp",
					"turn:global.relay.metered.ca:443",
					"turns:global.relay.metered.ca:443?transport=tcp",
				},
				Username:   "3684af77c3b49d3aa21b4fc1",
				Credential: "0ZFmoQdwkmXjk6vh",
			},
		},
		ICECandidatePoolSize: 10,
	}

	pc, err := ext_webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %v", err)
	}
	s.webrtcConnection = pc

	s.webrtcConnection.OnConnectionStateChange(func(state ext_webrtc.PeerConnectionState) {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		switch state {
		case ext_webrtc.PeerConnectionStateConnected:
			s.isConnected = true
			go s.streamVideo()
			go s.streamAudio()
		case ext_webrtc.PeerConnectionStateDisconnected, ext_webrtc.PeerConnectionStateFailed, ext_webrtc.PeerConnectionStateClosed:
			s.Close()
			s.isConnected = false
			// TODO: propagate the disconnection event to the receiver/camera
		}
	})

	// Handle ICE candidates
	s.webrtcConnection.OnICECandidate(func(candidate *ext_webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		candidateInit := candidate.ToJSON()
		if err := s.websocketConnection.WriteJSON(map[string]interface{}{
			"type":      "ice-candidate",
			"candidate": candidateInit,
		}); err != nil {
			fmt.Printf("Error sending ICE candidate: %v\n", err)
		}
	})

	return nil
}

// Creates video track and adds it to the peer connection.
func (s *WebRTCSender) createVideoTrack() error {
	videoTrack, err := ext_webrtc.NewTrackLocalStaticSample(
		ext_webrtc.RTPCodecCapability{MimeType: ext_webrtc.MimeTypeVP8},
		"video",
		"relay-video",
	)
	if err != nil {
		return fmt.Errorf("failed to create video track: %v", err)
	}
	s.videoTrack = videoTrack

	// Add tracks to peer connection
	if _, err = s.webrtcConnection.AddTrack(videoTrack); err != nil {
		return fmt.Errorf("failed to add video track: %v", err)
	}

	return nil
}

// Creates audio track and adds it to the peer connection.
func (s *WebRTCSender) createAudioTrack() error {
	audioTrack, err := ext_webrtc.NewTrackLocalStaticSample(
		ext_webrtc.RTPCodecCapability{MimeType: ext_webrtc.MimeTypeOpus},
		"audio",
		"relay-audio",
	)
	if err != nil {
		return fmt.Errorf("failed to create audio track: %v", err)
	}
	s.audioTrack = audioTrack

	if _, err = s.webrtcConnection.AddTrack(audioTrack); err != nil {
		return fmt.Errorf("failed to add audio track: %v", err)
	}

	return nil
}

// Streams video data frames from the video channel to the peer connection's video track.
func (s *WebRTCSender) streamVideo() {
	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case videoData := <-s.videoChannel:
			if s.isConnected && s.videoTrack != nil {
				if err := s.videoTrack.WriteSample(media.Sample{
					Data:     videoData,
					Duration: 1 * time.Millisecond,
				}); err != nil {
					fmt.Printf("❌ Error writing video sample: %v\n", err)
				}
			} else {
				fmt.Printf("⚠️ Cannot send video: connected=%v, track=%v\n", s.isConnected, s.videoTrack != nil)
			}
		case <-ticker.C:
			// If no video data is available, continue to maintain frame rate
		}
	}
}

// Streams audio data frames from the audio channel to the peer connection's audio track.
func (s *WebRTCSender) streamAudio() {
	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case audioData := <-s.audioChannel:
			if s.isConnected && s.audioTrack != nil {
				if err := s.audioTrack.WriteSample(media.Sample{
					Data:     audioData,
					Duration: 1 * time.Millisecond,
				}); err != nil {
					fmt.Printf("Error writing audio sample: %v\n", err)
				}
			}
		case <-ticker.C:
			// If no audio data is available, continue
		}
	}
}
