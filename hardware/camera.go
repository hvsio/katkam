package hardware

import (
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type Camera struct {
	Device      string
	StreamCmd   *exec.Cmd
	StreamMutex sync.Mutex
	IsStreaming bool

	// WebRTC related fields
	peerConnection *webrtc.PeerConnection
	videoTrack     *webrtc.TrackLocalStaticSample
	audioTrack     *webrtc.TrackLocalStaticSample
	upgrader       websocket.Upgrader
}

func NewCamera() *Camera {
	return &Camera{
		Device: "/dev/video0", // Default video device
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}
}

func (c *Camera) TakePicture(filename string) error {
	// Use ffmpeg to capture a single frame from the camera
	cmd := exec.Command("ffmpeg",
		"-f", "avfoundation",
		"-video_size", "1280x720",
		"-framerate", "30",
		"-i", "0", // Default camera input
		"-frames:v", "1",
		"-y", // Overwrite output file
		filename,
	)

	return cmd.Run()
}

func (c *Camera) StartVideoCapture(filename string, duration time.Duration) error {
	// Use ffmpeg to record video for specified duration
	cmd := exec.Command("ffmpeg",
		"-f", "avfoundation",
		"-video_size", "1280x720",
		"-framerate", "30",
		"-i", "0", // Default camera input
		"-t", fmt.Sprintf("%.0f", duration.Seconds()),
		"-y", // Overwrite output file
		filename,
	)

	return cmd.Run()
}

func (c *Camera) InitializeWebRTC() error {
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
	c.peerConnection = pc

	// Create video track
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8}, "video", "camera")
	if err != nil {
		return fmt.Errorf("failed to create video track: %v", err)
	}
	c.videoTrack = videoTrack

	// Add video track to peer connection
	_, err = pc.AddTrack(videoTrack)
	if err != nil {
		return fmt.Errorf("failed to add video track: %v", err)
	}

	return nil
}

func (c *Camera) StartStreaming() error {
	c.StreamMutex.Lock()
	defer c.StreamMutex.Unlock()

	if c.IsStreaming {
		return fmt.Errorf("streaming already active")
	}

	// Start FFmpeg process to capture frames and output VP8
	cmd := exec.Command("ffmpeg",
		"-f", "avfoundation",
		"-video_size", "640x480",
		"-framerate", "30",
		"-i", "0",
		"-c:v", "libvpx",
		"-b:v", "1M",
		"-crf", "30",
		"-f", "rtp",
		"rtp://127.0.0.1:5004",
	)

	c.StreamCmd = cmd
	c.IsStreaming = true

	go func() {
		err := cmd.Run()
		if err != nil {
			fmt.Printf("FFmpeg error: %v\n", err)
		}
		c.StreamMutex.Lock()
		c.IsStreaming = false
		c.StreamMutex.Unlock()
	}()

	return nil
}

func (c *Camera) StopStreaming() error {
	c.StreamMutex.Lock()
	defer c.StreamMutex.Unlock()

	if !c.IsStreaming {
		return fmt.Errorf("no active stream")
	}

	if c.StreamCmd != nil && c.StreamCmd.Process != nil {
		err := c.StreamCmd.Process.Kill()
		if err != nil {
			return fmt.Errorf("failed to kill stream process: %v", err)
		}
	}

	c.IsStreaming = false
	return nil
}

func (c *Camera) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}
	defer conn.Close()

	// Initialize WebRTC if not already done
	if c.peerConnection == nil {
		if err := c.InitializeWebRTC(); err != nil {
			fmt.Printf("WebRTC initialization error: %v\n", err)
			return
		}
	}

	// Handle ICE candidates
	c.peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
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

	// Handle connection state changes
	c.peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		fmt.Printf("Peer connection state changed: %s\n", state.String())
		if state == webrtc.PeerConnectionStateConnected {
			// Start capturing and streaming video
			go c.captureAndStreamVideo()
		}
	})

	// Listen for messages from client
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

			if err := c.peerConnection.SetRemoteDescription(offer); err != nil {
				fmt.Printf("Error setting remote description: %v\n", err)
				continue
			}

			answer, err := c.peerConnection.CreateAnswer(nil)
			if err != nil {
				fmt.Printf("Error creating answer: %v\n", err)
				continue
			}

			if err := c.peerConnection.SetLocalDescription(answer); err != nil {
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

			if err := c.peerConnection.AddICECandidate(candidate); err != nil {
				fmt.Printf("Error adding ICE candidate: %v\n", err)
			}
		}
	}
}

func (c *Camera) captureAndStreamVideo() {
	// This is a simplified implementation
	// In a real scenario, you'd capture frames from FFmpeg output and encode them
	ticker := time.NewTicker(33 * time.Millisecond) // ~30 FPS
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.peerConnection.ConnectionState() != webrtc.PeerConnectionStateConnected {
				return
			}
			// In a real implementation, you'd read actual video frames here
			// For now, this is a placeholder that would send dummy data
		}
	}
}
