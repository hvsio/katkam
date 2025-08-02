package receivers

type Receiver struct {
	OnVideoFrame   func([]byte)
	OnAudioFrame   func([]byte)
	OnConnected    func()
	OnDisconnected func()
}
