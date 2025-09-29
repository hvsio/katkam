package relay

import (
	"fmt"
	"sync"

	"katkam/internal/infrastructure/connectivity"
)

type WebRTCRelay struct {
	sender   connectivity.Sender
	receiver connectivity.Receiver
	mutex    sync.RWMutex
	isActive bool
}

func NewWebRTCRelay(receiver connectivity.Receiver, sender connectivity.Sender) *WebRTCRelay {
	relay := &WebRTCRelay{
		receiver: receiver,
		sender:   sender,
	}

	return relay
}

// Starts the receiver and sender components
// Also begins to listen for video and audio frames from the receiver channels and sends them to the sender
func (r *WebRTCRelay) Start() {
	err := r.receiver.Start()
	if err != nil {
		panic(err)
	}
	err = r.sender.Start()
	if err != nil {
		panic(err)
	}

	videoChannel := r.receiver.GetReceiverVideoChannel()
	go func() {
		for videoFrame := range videoChannel {
			if r.sender.IsConnected() {
				r.sender.SendVideoFrame(videoFrame)
			}
		}
	}()

	audioChannel := r.receiver.GetReceiverAudioChannel()
	go func() {
		for audioFrame := range audioChannel {
			if r.sender.IsConnected() {
				r.sender.SendAudioFrame(audioFrame)
			}
		}
	}()

}

func (r *WebRTCRelay) GetReceiver() connectivity.WebSocket {
	return r.receiver
}

func (r *WebRTCRelay) GetSender() connectivity.WebSocket {
	return r.sender
}

func (r *WebRTCRelay) IsActive() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.isActive
}

func (r *WebRTCRelay) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.isActive = false

	var receiverErr, senderErr error
	receiverErr = r.receiver.Close()
	senderErr = r.sender.Close()

	if receiverErr != nil {
		return fmt.Errorf("receiver close error: %v", receiverErr)
	}
	if senderErr != nil {
		return fmt.Errorf("sender close error: %v", senderErr)
	}

	return nil
}
