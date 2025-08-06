package connectivity

import "net/http"

type VideoStreamer struct {
	OnVideoFrame   func([]byte)
	OnAudioFrame   func([]byte)
	OnConnected    func()
	OnDisconnected func()
}

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
