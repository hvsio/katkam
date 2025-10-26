package relay

import (
	"context"
	"fmt"
	"sync"

	connectivity "katkam/internal/infrastructure/websockets"
)

type WebRTCRelay struct {
	sender   connectivity.Sender
	receiver connectivity.Receiver
	mutex    sync.RWMutex
	isActive bool

	wg sync.WaitGroup
}

func NewWebRTCRelay(receiver connectivity.Receiver, sender connectivity.Sender) *WebRTCRelay {
	relay := &WebRTCRelay{
		receiver: receiver,
		sender:   sender,
		wg:       sync.WaitGroup{},
	}

	return relay
}

// Starts the receiver and sender components
// Also begins to listen for video and audio frames from the receiver channels and sends them to the sender
func (r *WebRTCRelay) Start(ctx context.Context) {
	r.mutex.Lock()
	r.isActive = true
	r.mutex.Unlock()
	streamingContext, streamingCancel := context.WithCancel(context.WithoutCancel(ctx))

	err := r.receiver.Start(streamingContext)
	if err != nil {
		fmt.Printf("error starting receiver: %v", err)
	}
	err = r.sender.Start(streamingContext)
	if err != nil {
		fmt.Printf("error starting sender: %v", err)
	}

	videoChannel := r.receiver.GetReceiverVideoChannel()
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-streamingContext.Done():
				for len(videoChannel) > 0 {
					<-videoChannel
				}
				return
			case videoFrame, ok := <-videoChannel:
				if !ok || !r.sender.IsConnected() {
					fmt.Println("No connection, canceling streaming")
					streamingCancel()
				} else {
					r.sender.SendVideoFrame(videoFrame)
				}
			}
		}
	}()

	// Start audio relay goroutine
	audioChannel := r.receiver.GetReceiverAudioChannel()
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case <-streamingContext.Done():
				// Context cancelled, drain remaining frames and exit
				for len(audioChannel) > 0 {
					<-audioChannel
				}
				return
			case audioFrame, ok := <-audioChannel:
				if !ok || !r.sender.IsConnected() {
					streamingCancel()
				} else {
					r.sender.SendAudioFrame(audioFrame)
				}
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
	if !r.isActive {
		return nil
	}

	r.isActive = false

	// Close receiver and sender
	var receiverErr, senderErr error
	receiverErr = r.receiver.Close()
	senderErr = r.sender.Close()

	if receiverErr != nil {
		return fmt.Errorf("receiver close error: %v", receiverErr)
	}
	if senderErr != nil {
		return fmt.Errorf("sender close error: %v", senderErr)
	}

	fmt.Println("Relay closed successfully")
	return nil
}
