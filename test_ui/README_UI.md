# 🐱 KatKam - Live Camera Stream System

A WebRTC-based camera streaming system with a clean web UI.

## 🚀 Quick Start

### Option 1: Run Everything (Recommended)
```bash
./run.sh
```

### Option 2: Manual Setup

1. **Start the backend camera server:**
```bash
go build -o katkam .
./katkam
```

2. **Start the UI server:**
```bash
cd ui
go build -o ui-server server.go
./ui-server
cd ..
```

## 🌐 Access Points

- **📱 Camera Stream UI**: http://localhost:8081
- **🎬 Backend Server**: http://localhost:8080  
- **📊 Camera Status**: http://localhost:8080/api/camera/status
- **🔧 UI Health Check**: http://localhost:8081/health

## 🎮 How to Use

1. Open http://localhost:8081 in your web browser
2. Click "Connect to Camera" 
3. Allow camera access if prompted
4. Enjoy your live camera stream! 📹

## 🛠️ Architecture

```
┌─────────────────┐    WebRTC/WebSocket    ┌─────────────────┐
│   Browser UI    │◄──────────────────────►│  Backend Server │
│  (Port 8081)    │                        │  (Port 8080)    │
└─────────────────┘                        └─────────────────┘
                                                     │
                                                     ▼
                                            ┌─────────────────┐
                                            │  macOS Camera   │
                                            │  (AVFoundation) │
                                            └─────────────────┘
```

## 🔧 Technical Details

- **Backend**: Go + WebRTC (Pion) + FFmpeg
- **Frontend**: Vanilla HTML/CSS/JavaScript + WebRTC
- **Video Format**: VP8 codec, 1280x720@30fps
- **Protocol**: WebRTC over WebSocket signaling

## 📋 Features

- ✅ Live camera streaming from macOS
- ✅ WebRTC peer-to-peer connection
- ✅ Clean responsive web UI
- ✅ Real-time connection status
- ✅ Connection info display
- ✅ Auto-reconnection support
- ✅ CORS enabled for development

## 🐛 Troubleshooting

### Camera not working?
- Make sure your camera isn't being used by another app
- Check if FFmpeg is installed: `ffmpeg -version`
- Verify camera permissions in macOS System Preferences

### Connection issues?
- Ensure both servers are running
- Check browser console for WebRTC errors
- Verify backend is accessible: `curl http://localhost:8080/api/camera/status`

### UI not loading?
- Check UI server is running: `curl http://localhost:8081/health`
- Clear browser cache and reload

## 🛑 Stopping the System

Press `Ctrl+C` in the terminal running `./run.sh`, or manually kill processes:

```bash
pkill -f katkam
pkill -f ui-server
```

---

**Enjoy your KatKam streaming! 🎉**
