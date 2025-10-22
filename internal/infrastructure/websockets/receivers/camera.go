package receivers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"katkam/internal/infrastructure/websockets"
	"net/http"
	"os/exec"
	"sync"
	"time"
)

type Camera struct {
	websockets.VideoStreamer

	Device      string
	StreamCmd   *exec.Cmd
	StreamMutex sync.Mutex
	isStreaming bool
}

func NewCamera() *Camera {
	return &Camera{
		Device: "0", // Default camera input for AVFoundation on macOS
		VideoStreamer: websockets.VideoStreamer{
			ReceiverVideoChannel: make(chan []byte),
			ReceiverAudioChannel: make(chan []byte),
		},
	}
}

func (c *Camera) GetReceiverAudioChannel() chan []byte {
	return c.ReceiverAudioChannel
}

func (c *Camera) GetReceiverVideoChannel() chan []byte {
	return c.ReceiverVideoChannel
}

func (c *Camera) Start() error {
	go func() {
		// Start a 60-second video capture that streams frames
		if err := c.StartVideoCapture(60 * 60 * time.Second); err != nil { // 1 hour duration
			fmt.Printf("❌ Failed to start camera capture: %v\n", err)
		}
	}()
	return nil
}

func (c *Camera) StartVideoCapture(duration time.Duration) error {
	c.StreamMutex.Lock()
	defer c.StreamMutex.Unlock()

	if c.isStreaming {
		return fmt.Errorf("camera is already streaming")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use ffmpeg to capture video and output IVF format for VP8 frames
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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	fmt.Printf("📹 Starting FFmpeg command: %s\n", cmd.String())
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %v", err)
	}
	c.StreamCmd = cmd
	c.isStreaming = true

	// Stream frames to callback in fire-and-forget manner
	go c.captureFramesToCallback(stdout, ctx)

	// Wait for command to complete
	err = cmd.Wait()

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
			c.ReceiverVideoChannel <- frameData
		}
	}
}

func (c *Camera) Close() error {
	if !c.isStreaming {
		return fmt.Errorf("camera is not currently streaming")
	}

	if c.StreamCmd != nil && c.StreamCmd.Process != nil {
		if err := c.StreamCmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill capture process: %v", err)
		}
	}

	c.isStreaming = false
	return nil
}

func (c *Camera) HandleWebSocketConnection(w http.ResponseWriter, req *http.Request) {
	panic("Camera is directly connected, it should not handle websocket connection. Make sure you configured the receiver correctly.")
}

func (c *Camera) IsConnected() bool {
	return c.isStreaming
}
