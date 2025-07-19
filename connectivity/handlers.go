package connectivity

import (
	"encoding/json"
	"net/http"
)

type RelayServer struct {
	relay *WebRTCRelay
}

func NewRelayServer() *RelayServer {
	return &RelayServer{
		relay: NewWebRTCRelay(),
	}
}

// HandleReceiverSignaling handles WebSocket connections for the input stream
func (s *RelayServer) HandleReceiverSignaling(w http.ResponseWriter, r *http.Request) {
	s.relay.GetReceiver().HandleWebSocketConnection(w, r)
}

// HandleSenderSignaling handles WebSocket connections for the output stream to frontend
func (s *RelayServer) HandleSenderSignaling(w http.ResponseWriter, r *http.Request) {
	s.relay.GetSender().HandleWebSocketConnection(w, r)
}

// HandleStatus returns the current status of the relay
func (s *RelayServer) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	status := s.relay.GetStatus()
	json.NewEncoder(w).Encode(status)
}

// SetupRoutes sets up all the HTTP routes for the relay server
func (s *RelayServer) SetupRoutes() {
	http.HandleFunc("/ws/receiver", s.HandleReceiverSignaling)
	http.HandleFunc("/ws/sender", s.HandleSenderSignaling)
	http.HandleFunc("/api/status", s.HandleStatus)
	
	// Serve a simple HTML page for testing the frontend connection
	http.HandleFunc("/", s.HandleTestPage)
}

// HandleTestPage serves a simple test page for WebRTC frontend connection
func (s *RelayServer) HandleTestPage(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>WebRTC Relay Test</title>
</head>
<body>
    <h1>WebRTC Video Relay</h1>
    <div>
        <h2>Status</h2>
        <div id="status"></div>
    </div>
    <div>
        <h2>Received Video Stream</h2>
        <video id="remoteVideo" autoplay playsinline controls style="width: 640px; height: 480px; border: 1px solid black;"></video>
    </div>
    <div>
        <button id="connectBtn">Connect to Stream</button>
        <button id="disconnectBtn">Disconnect</button>
    </div>

    <script>
        let pc = null;
        let ws = null;
        const remoteVideo = document.getElementById('remoteVideo');
        const statusDiv = document.getElementById('status');
        const connectBtn = document.getElementById('connectBtn');
        const disconnectBtn = document.getElementById('disconnectBtn');

        function updateStatus() {
            fetch('/api/status')
                .then(response => response.json())
                .then(data => {
                    statusDiv.innerHTML = JSON.stringify(data, null, 2);
                })
                .catch(err => console.error('Status fetch error:', err));
        }

        function connect() {
            pc = new RTCPeerConnection({
                iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
            });

            pc.ontrack = (event) => {
                console.log('Received track:', event.track.kind);
                if (event.track.kind === 'video') {
                    remoteVideo.srcObject = event.streams[0];
                }
            };

            pc.onicecandidate = (event) => {
                if (event.candidate) {
                    ws.send(JSON.stringify({
                        type: 'ice-candidate',
                        candidate: event.candidate
                    }));
                }
            };

            ws = new WebSocket('ws://localhost:8080/ws/sender');
            
            ws.onopen = () => {
                console.log('WebSocket connected');
                // Create offer
                pc.createOffer()
                    .then(offer => {
                        return pc.setLocalDescription(offer);
                    })
                    .then(() => {
                        ws.send(JSON.stringify({
                            type: 'offer',
                            sdp: pc.localDescription.sdp
                        }));
                    });
            };

            ws.onmessage = (event) => {
                const message = JSON.parse(event.data);
                if (message.type === 'answer') {
                    pc.setRemoteDescription(new RTCSessionDescription({
                        type: 'answer',
                        sdp: message.sdp
                    }));
                } else if (message.type === 'ice-candidate') {
                    pc.addIceCandidate(new RTCIceCandidate(message.candidate));
                }
            };

            connectBtn.disabled = true;
            disconnectBtn.disabled = false;
        }

        function disconnect() {
            if (pc) {
                pc.close();
                pc = null;
            }
            if (ws) {
                ws.close();
                ws = null;
            }
            remoteVideo.srcObject = null;
            connectBtn.disabled = false;
            disconnectBtn.disabled = true;
        }

        connectBtn.addEventListener('click', connect);
        disconnectBtn.addEventListener('click', disconnect);

        // Update status every 2 seconds
        setInterval(updateStatus, 2000);
        updateStatus();
    </script>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// GetRelay returns the underlying relay instance
func (s *RelayServer) GetRelay() *WebRTCRelay {
	return s.relay
}

// Close closes the relay server
func (s *RelayServer) Close() error {
	return s.relay.Close()
}
