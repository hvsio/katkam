package main

import (
	"fmt"
	"katkam/config"
	"katkam/features/connectivity/receivers"
	"katkam/features/connectivity/relay"
	"katkam/features/connectivity/senders"
	"katkam/handlers"
	internal_http "katkam/routes/http"
	"katkam/routes/websocket"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	config, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	authController := handlers.NewAuthController(config.AuthConfig, nil)
	httpRouter := internal_http.NewHttpRouter(authController)
	websocketRouter := websocket.NewWebSocketRouter(nil)

	httpRouter.SetupRoutes()
	websocketRouter.SetupRoutes()

	var receiver relay.Receiver
	if config.Server.UseDirectCamera {
		receiver = receivers.NewCamera()
	} else {
		receiver = receivers.NewWebRTCReceiver()
	}

	sender := senders.NewWebRTCSender()
	relay := relay.NewWebRTCRelay(receiver, sender)
	relay.Start(http.ResponseWriter(nil), nil)

	// Start HTTP server
	port := fmt.Sprintf(":%d", config.Server.Port)
	fmt.Printf("Starting camera streaming server on port %s\n", port)
	fmt.Printf("Access camera stream at: http://localhost%s\n", port)
	fmt.Printf("Camera control: http://localhost%s/api/camera/status\n", port)
	fmt.Printf("Camera WebSocket: ws://localhost%s/ws/sender\n", port)

	log.Fatal(http.ListenAndServe(port, nil))
}
