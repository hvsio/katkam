package connectivity

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type WebRTCReceiver struct {
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel
	videoTrack     *webrtc.TrackRemote
	audioTrack     *webrtc.TrackRemote
	upgrader       websocket.Upgrader
	isConnected    bool
	mutex          sync.RWMutex
	
	// Callback for when video frames are received
	OnVideoFrame func([]byte)
	OnAudioFrame func([]byte)
	OnConnected  func()
	OnDisconnected func()
}

func NewWebRTCReceiver() *WebRTCReceiver {
	return &WebRTCReceiver{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}
}

func (r *WebRTCReceiver) InitializePeerConnection() error {
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
	r.peerConnection = pc

	// Handle incoming tracks
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		fmt.Printf("Received track: %s, codec: %s\n", track.Kind().String(), track.Codec().MimeType)
		
		if track.Kind() == webrtc.RTPCodecTypeVideo {
			r.mutex.Lock()
			r.videoTrack = track
			r.mutex.Unlock()
			go r.handleVideoTrack(track)
		} else if track.Kind() == webrtc.RTPCodecTypeAudio {
			r.mutex.Lock()
			r.audioTrack = track
			r.mutex.Unlock()
			go r.handleAudioTrack(track)
		}
	})

	// Handle connection state changes
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Receiver connection state: %s\n", state.String())
		r.mutex.Lock()
		defer r.mutex.Unlock()
		
		switch state {
		case webrtc.PeerConnectionStateConnected:
			r.isConnected = true
			if r.OnConnected != nil {
				go r.OnConnected()
			}
		case webrtc.PeerConnectionStateDisconnected, webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateClosed:
			r.isConnected = false
			if r.OnDisconnected != nil {
				go r.OnDisconnected()
			}
		}
	})

	// Handle ICE connection state changes
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		fmt.Printf("Receiver ICE connection state: %s\n", state.String())
	})

	return nil
}

func (r *WebRTCReceiver) handleVideoTrack(track *webrtc.TrackRemote) {
	for {
		packet, _, err := track.ReadRTP()
		if err != nil {
			if err == io.EOF {
				fmt.Println("Video track ended")
				return
			}
			fmt.Printf("Error reading video RTP packet: %v\n", err)
			continue
		}

		// Forward video packet data to callback if set
		if r.OnVideoFrame != nil {
			r.OnVideoFrame(packet.Payload)
		}
	}
}

func (r *WebRTCReceiver) handleAudioTrack(track *webrtc.TrackRemote) {
	for {
		packet, _, err := track.ReadRTP()
		if err != nil {
			if err == io.EOF {
				fmt.Println("Audio track ended")
				return
			}
			fmt.Printf("Error reading audio RTP packet: %v\n", err)
			continue
		}

		// Forward audio packet data to callback if set
		if r.OnAudioFrame != nil {
			r.OnAudioFrame(packet.Payload)
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
	if r.peerConnection == nil {
		if err := r.InitializePeerConnection(); err != nil {
			fmt.Printf("Failed to initialize peer connection: %v\n", err)
			return
		}
	}

	// Handle ICE candidates
	r.peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
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

			if err := r.peerConnection.SetRemoteDescription(offer); err != nil {
				fmt.Printf("Error setting remote description: %v\n", err)
				continue
			}

			answer, err := r.peerConnection.CreateAnswer(nil)
			if err != nil {
				fmt.Printf("Error creating answer: %v\n", err)
				continue
			}

			if err := r.peerConnection.SetLocalDescription(answer); err != nil {
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

			if err := r.peerConnection.AddICECandidate(candidate); err != nil {
				fmt.Printf("Error adding ICE candidate: %v\n", err)
			}
		}
	}
}

func (r *WebRTCReceiver) IsConnected() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.isConnected
}

func (r *WebRTCReceiver) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if r.peerConnection != nil {
		return r.peerConnection.Close()
	}
	return nil
}
