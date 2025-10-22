package websockets

import "net/http"

type VideoStreamer struct {
	ReceiverVideoChannel chan []byte
	ReceiverAudioChannel chan []byte
}

type WebSocket interface {
	Start() error
	Close() error
	IsConnected() bool
	HandleWebSocketConnection(w http.ResponseWriter, req *http.Request)
}

type Receiver interface {
	WebSocket
	GetReceiverVideoChannel() chan []byte
	GetReceiverAudioChannel() chan []byte
}

type Sender interface {
	WebSocket
	SendVideoFrame(data []byte)
	SendAudioFrame(data []byte)
}
