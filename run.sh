#!/bin/bash

echo "🚀 Starting KatKam System..."
echo "════════════════════════════════════════"

# Function to cleanup background processes
cleanup() {
    echo ""
    echo "🛑 Shutting down KatKam..."
    kill $BACKEND_PID $UI_PID 2>/dev/null
    wait $BACKEND_PID $UI_PID 2>/dev/null
    echo "✅ All services stopped"
    exit 0
}

# Set trap to cleanup on exit
trap cleanup SIGINT SIGTERM

# Build the backend
echo "🔨 Building backend..."
go build -o katkam .
if [ $? -ne 0 ]; then
    echo "❌ Backend build failed"
    exit 1
fi

# Build the UI server
#echo "🔨 Building UI server..."
#cd test_ui
#go build -o ui-server server.go
#if [ $? -ne 0 ]; then
#    echo "❌ UI server build failed"
#    exit 1
#fi
# cd ..

echo ""
echo "🎬 Starting backend camera server (port 8080)..."
./katkam &
BACKEND_PID=$!

# Give backend time to start
sleep 2

#echo "🖥️  Starting UI server (port 8081)..."
#cd test_ui
#./ui-server &
#UI_PID=$!
#cd ..

# Give UI server time to start
sleep 1

echo ""
echo "✅ KatKam System Ready!"
echo "────────────────────────────────────────"
echo "📹 Backend (Camera): http://localhost:8080"
echo "🖥️  Frontend (UI):   http://localhost:8081"
echo "📊 Camera Status:    http://localhost:8080/api/camera/status"
echo "🔧 UI Health:        http://localhost:8081/health"
echo ""
echo "Open http://localhost:8081 in your browser to view the camera stream!"
echo "Press Ctrl+C to stop all services"
echo "────────────────────────────────────────"

# Wait for processes
wait $BACKEND_PID $UI_PID
