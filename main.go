package main

import (
	"fmt"
	"katkam/connectivity/receivers"
	"katkam/connectivity/relay"
	"katkam/connectivity/senders"
	"log"
	"net/http"
	"time"
)

func main() {
	camera := receivers.NewCamera()
	sender := senders.NewWebRTCSender()

	relay := relay.NewWebRTCRelay(camera, sender)

	// Start camera streaming (this will start sending frames to the callbacks)
	go func() {
		// Start a 60-second video capture that streams frames
		if err := camera.StartVideoCapture("", 60*60*time.Second); err != nil { // 1 hour duration
			fmt.Printf("❌ Failed to start camera capture: %v\n", err)
		}
	}()

	relay.Setup()

	// Start HTTP server
	port := ":8080"
	fmt.Printf("Starting camera streaming server on port %s\n", port)
	fmt.Printf("Access camera stream at: http://localhost%s\n", port)
	fmt.Printf("Camera control: http://localhost%s/api/camera/status\n", port)
	fmt.Printf("Camera WebSocket: ws://localhost%s/ws/sender\n", port)

	log.Fatal(http.ListenAndServe(port, nil))
}
