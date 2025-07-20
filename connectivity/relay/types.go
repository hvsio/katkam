package relay

import "net/http"

type Socket interface {
	IsConnected() bool
	Close() error
	HandleWebSocketConnection(w http.ResponseWriter, req *http.Request)
	ExposesEndpoint() bool
}

type Receiver interface {
	Socket
	AssignDisconnectedCallback(func())
	AssignConnectedCallback(func())
	AssignAudioFrameCallback(func(d []byte))
	AssignVideoFrameCallback(func(data []byte))
}

type Sender interface {
	Socket
	SendVideoFrame(data []byte)
	SendAudioFrame(data []byte)
}
