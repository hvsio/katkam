package receivers

import (
	"context"
	"fmt"
	"io"
	"katkam/internal/infrastructure/websockets"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	ext_webrtc "github.com/pion/webrtc/v3"
)

type WebRTCReceiver struct {
	// WebSocket
	websockets.VideoStreamer
	upgrader websocket.Upgrader

	// WebRTC
	videoTrack *ext_webrtc.TrackRemote
	audioTrack *ext_webrtc.TrackRemote
	pc         *ext_webrtc.PeerConnection
	pcMutex    sync.RWMutex

	isStreaming bool
	mutex       sync.RWMutex
}

func NewWebRTCReceiver() *WebRTCReceiver {
	return &WebRTCReceiver{
		VideoStreamer: websockets.VideoStreamer{
			ReceiverVideoChannel: make(chan []byte, 100),
			ReceiverAudioChannel: make(chan []byte, 100),
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo TODO
			},
		},
	}
}

func (r *WebRTCReceiver) Start(ctx context.Context) error {
	return nil
}

func (r *WebRTCReceiver) GetReceiverAudioChannel() chan []byte {
	return r.ReceiverAudioChannel
}

func (r *WebRTCReceiver) GetReceiverVideoChannel() chan []byte {
	return r.ReceiverVideoChannel
}

func (r *WebRTCReceiver) InitializePeerConnection() error {
	config := ext_webrtc.Configuration{
		ICEServers: []ext_webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	pc, err := ext_webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %v", err)
	}
	r.pc = pc

	// Handle incoming tracks
	pc.OnTrack(func(track *ext_webrtc.TrackRemote, receiver *ext_webrtc.RTPReceiver) {
		if track.Kind() == ext_webrtc.RTPCodecTypeVideo {
			r.pcMutex.Lock()
			r.videoTrack = track
			r.pcMutex.Unlock()
			go r.handleVideoTrack(track)
		} else if track.Kind() == ext_webrtc.RTPCodecTypeAudio {
			r.pcMutex.Lock()
			r.audioTrack = track
			r.pcMutex.Unlock()
			go r.handleAudioTrack(track)
		}
	})

	// Handle connection state changes
	pc.OnConnectionStateChange(func(state ext_webrtc.PeerConnectionState) {
		r.pcMutex.Lock()
		defer r.pcMutex.Unlock()

		switch state {
		case ext_webrtc.PeerConnectionStateConnected:
			r.isStreaming = true
			//TODO
		case ext_webrtc.PeerConnectionStateDisconnected, ext_webrtc.PeerConnectionStateFailed, ext_webrtc.PeerConnectionStateClosed:
			r.isStreaming = false
			//TODO
		}
	})

	// Handle ICE connection state changes
	pc.OnICEConnectionStateChange(func(state ext_webrtc.ICEConnectionState) {
		fmt.Printf("Receiver ICE connection state: %s\n", state.String())
	})

	return nil
}

func (r *WebRTCReceiver) handleVideoTrack(track *ext_webrtc.TrackRemote) {
	for {
		_, _, err := track.ReadRTP()
		if err != nil {
			if err == io.EOF {
				fmt.Println("Video track ended")
				return
			}
			fmt.Printf("Error reading video RTP packet: %v\n", err)
			continue
		}
	}
}

func (r *WebRTCReceiver) handleAudioTrack(track *ext_webrtc.TrackRemote) {
	for {
		_, _, err := track.ReadRTP()
		if err != nil {
			if err == io.EOF {
				fmt.Println("Audio track ended")
				return
			}
			fmt.Printf("Error reading audio RTP packet: %v\n", err)
			continue
		}

	}
}

func (r *WebRTCReceiver) HandleWebSocketConnection(w http.ResponseWriter, req *http.Request) {
	conn, err := r.upgrader.Upgrade(w, req, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}
	defer conn.Close()

	// Initialize peer connection if not already done
	if r.pc == nil {
		if err := r.InitializePeerConnection(); err != nil {
			fmt.Printf("Failed to initialize peer connection: %v\n", err)
			return
		}
	}

	// Handle ICE candidates
	r.pc.OnICECandidate(func(candidate *ext_webrtc.ICECandidate) {
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
			offer := ext_webrtc.SessionDescription{
				Type: ext_webrtc.SDPTypeOffer,
				SDP:  message["sdp"].(string),
			}

			if err := r.pc.SetRemoteDescription(offer); err != nil {
				fmt.Printf("Error setting remote description: %v\n", err)
				continue
			}

			answer, err := r.pc.CreateAnswer(nil)
			if err != nil {
				fmt.Printf("Error creating answer: %v\n", err)
				continue
			}

			if err := r.pc.SetLocalDescription(answer); err != nil {
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

			candidate := ext_webrtc.ICECandidateInit{
				Candidate:     candidateMap["candidate"].(string),
				SDPMLineIndex: &sdpMLineIndex,
				SDPMid:        &sdpMid,
			}

			if err := r.pc.AddICECandidate(candidate); err != nil {
				fmt.Printf("Error adding ICE candidate: %v\n", err)
			}
		}
	}
}

func (r *WebRTCReceiver) IsConnected() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.isStreaming
}

func (r *WebRTCReceiver) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.pc != nil {
		return r.pc.Close()
	}
	return nil
}
