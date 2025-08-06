package main

import (
	"fmt"
	"katkam/internal/auth"
	"katkam/internal/config"
	"katkam/internal/handlers"
	"katkam/internal/infrastructure/connectivity"
	"katkam/internal/infrastructure/connectivity/receivers"
	"katkam/internal/infrastructure/connectivity/relay"
	"katkam/internal/infrastructure/connectivity/senders"
	repo "katkam/internal/infrastructure/repository"
	internal_http "katkam/internal/infrastructure/routes/http"
	internal_websocket "katkam/internal/infrastructure/routes/websocket"
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

	// infrastructure
	userRepo := repo.NewUserRepository(config.Users)

	var receiver connectivity.Receiver
	if config.Server.UseDirectCamera {
		receiver = receivers.NewCamera()
	} else {
		receiver = receivers.NewWebRTCReceiver()
	}
	sender := senders.NewWebRTCSender()
	relay := relay.NewWebRTCRelay(receiver, sender)
	// relay.Start()

	// features
	authorizer := auth.NewAuthorizer(config.Auth, userRepo)

	// handlers
	authHandler := handlers.NewAuthHandler(authorizer)
	relayHandler := handlers.NewRelayHandler(relay)

	// routes
	httpRouter := internal_http.NewHttpRouter(authHandler, relayHandler)
	websocketRouter := internal_websocket.NewWebSocketRouter(relayHandler)
	httpRouter.SetupRoutes()
	websocketRouter.SetupRoutes()

	// Start HTTP server
	port := fmt.Sprintf(":%d", config.Server.Port)
	fmt.Printf("Starting camera streaming server on port %s\n", port)
	fmt.Printf("Access camera stream at: http://localhost%s\n", port)
	fmt.Printf("Camera control: http://localhost%s/api/camera/status\n", port)
	fmt.Printf("Camera WebSocket: ws://localhost%s/ws/sender\n", port)

	log.Fatal(http.ListenAndServe(port, nil))
}
