package relay

import "net/http"

type Socket interface {
	IsConnected() bool
	Start() error
	Close() error
	HandleWebSocketConnection(w http.ResponseWriter, req *http.Request)
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
