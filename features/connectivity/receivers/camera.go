package receivers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type Camera struct {
	Receiver

	Device      string
	StreamCmd   *exec.Cmd
	StreamMutex sync.Mutex
	IsStreaming bool
}

func NewCamera() *Camera {
	return &Camera{
		Device: "0", // Default camera input for AVFoundation on macOS
	}
}

func (c *Camera) Start() error {
	go func() {
		// Start a 60-second video capture that streams frames
		if err := c.StartVideoCapture("", 60*60*time.Second); err != nil { // 1 hour duration
			fmt.Printf("❌ Failed to start camera capture: %v\n", err)
		}
	}()
	return nil
}

func (r *Camera) AssignVideoFrameCallback(fn func([]byte)) {
	r.OnVideoFrame = fn
}

func (r *Camera) AssignAudioFrameCallback(fn func([]byte)) {
	r.OnAudioFrame = fn
}

func (r *Camera) AssignConnectedCallback(fn func()) {
	r.OnConnected = fn
}

func (r *Camera) AssignDisconnectedCallback(fn func()) {
	r.OnDisconnected = fn
}

func (c *Camera) StartVideoCapture(filename string, duration time.Duration) error {
	fmt.Printf("🎬 Starting video capture for %.0f seconds...\n", duration.Seconds())
	c.StreamMutex.Lock()
	defer c.StreamMutex.Unlock()

	if c.IsStreaming {
		return fmt.Errorf("camera is already streaming")
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use ffmpeg to capture video and output IVF format for VP8 frames
	// Note: macOS requires camera permission for Terminal/process
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "avfoundation",
		"-video_size", "640x480", // Resolution supported by camera
		"-framerate", "30", // Use 30fps as it's better supported
		"-i", c.Device, // Camera device
		"-t", fmt.Sprintf("%.0f", duration.Seconds()),
		"-c:v", "libvpx",
		"-b:v", "500k", // Lower bitrate
		"-crf", "40", // Higher CRF for smaller files
		"-f", "ivf", // IVF format contains individual VP8 frames
		"-", // Output to stdout for streaming
	)

	// Set up pipes for streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// Also capture stderr for debugging
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	fmt.Printf("📹 Starting FFmpeg command: %s\n", cmd.String())
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %v", err)
	}
	fmt.Println("✅ FFmpeg started successfully")

	// Log stderr in background
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, err := stderr.Read(buffer)
			if err != nil {
				break
			}
			if n > 0 {
				fmt.Printf("FFmpeg stderr: %s", string(buffer[:n]))
			}
		}
	}()

	c.StreamCmd = cmd
	c.IsStreaming = true

	// Stream frames to callback in fire-and-forget manner
	go c.captureFramesToCallback(stdout, ctx)

	// Wait for command to complete
	err = cmd.Wait()

	c.StreamMutex.Lock()
	c.IsStreaming = false
	c.StreamMutex.Unlock()

	return err
}

func (c *Camera) captureFramesToCallback(reader io.Reader, ctx context.Context) {
	// Skip IVF header (32 bytes)
	header := make([]byte, 32)
	_, err := io.ReadFull(reader, header)
	if err != nil {
		fmt.Printf("Failed to read IVF header: %v\n", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Read IVF frame header (12 bytes)
			frameHeader := make([]byte, 12)
			_, err := io.ReadFull(reader, frameHeader)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Camera stream ended: %v\n", err)
				}
				return
			}

			// Extract frame size from header (little-endian uint32 at offset 0)
			frameSize := binary.LittleEndian.Uint32(frameHeader[0:4])
			if frameSize == 0 || frameSize > 1024*1024 { // Sanity check
				fmt.Printf("Invalid frame size: %d\n", frameSize)
				continue
			}

			// Read the actual VP8 frame data
			frameData := make([]byte, frameSize)
			_, err = io.ReadFull(reader, frameData)
			if err != nil {
				fmt.Printf("Failed to read frame data: %v\n", err)
				return
			}

			// Send the VP8 frame to WebRTC
			if c.OnVideoFrame != nil {
				c.OnVideoFrame(frameData)
			}
		}
	}
}

func (c *Camera) StopVideoCapture() error {
	c.StreamMutex.Lock()
	defer c.StreamMutex.Unlock()

	if !c.IsStreaming {
		return fmt.Errorf("camera is not currently streaming")
	}

	if c.StreamCmd != nil && c.StreamCmd.Process != nil {
		if err := c.StreamCmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill capture process: %v", err)
		}
	}

	c.IsStreaming = false
	return nil
}

func (c *Camera) Close() error {
	if c.IsStreaming {
		return c.StopVideoCapture()
	}
	return nil
}

func (c *Camera) HandleWebSocketConnection(w http.ResponseWriter, req *http.Request) {
	panic("Camera is directly connected, it should not handle websocket connection. Make sure you configured the receiver correctly.")
}

func (c *Camera) IsConnected() bool {
	return c.IsStreaming
}
