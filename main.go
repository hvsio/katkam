package main

import (
	"fmt"
	"katkam/connectivity/receivers"
	"katkam/connectivity/relay"
	"katkam/connectivity/senders"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	var receiver relay.Receiver
	use_direct_camera := os.Getenv("USE_DIRECT_CAMERA")
	if use_direct_camera == "true" {
		receiver = receivers.NewCamera()
	} else {
		receiver = receivers.NewWebRTCReceiver()
	}
	sender := senders.NewWebRTCSender()

	relay := relay.NewWebRTCRelay(receiver, sender)
	relay.Setup()

	// Start HTTP server
	port := ":8080"
	fmt.Printf("Starting camera streaming server on port %s\n", port)
	fmt.Printf("Access camera stream at: http://localhost%s\n", port)
	fmt.Printf("Camera control: http://localhost%s/api/camera/status\n", port)
	fmt.Printf("Camera WebSocket: ws://localhost%s/ws/sender\n", port)

	log.Fatal(http.ListenAndServe(port, nil))
}
