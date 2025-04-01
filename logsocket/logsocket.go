package logsocket

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/LajnaLegenden/transpiler4/helpers"
	"github.com/gorilla/websocket"
)

var (
	// Upgrader is used to upgrade HTTP connections to WebSocket connections
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins
		},
	}

	// Clients holds all connected WebSocket clients
	clients    = make(map[*websocket.Conn]bool)
	clientsMux sync.Mutex

	// Server variables
	server     *http.Server
	serverMux  sync.Mutex
	isRunning  bool
	serverPort int
)

// StartServer starts a web server that serves a Vue.js app with Tailwind CSS
// and also hosts a WebSocket server for streaming build logs
func StartServer() (int, error) {
	serverMux.Lock()
	defer serverMux.Unlock()

	if isRunning {
		return serverPort, nil
	}

	// Generate our port number
	serverPort = helpers.GeneratePortNumber()

	// Create a new HTTP server mux
	mux := http.NewServeMux()

	// Serve the static HTML file
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		serveHTML(w)
	})

	// Handle WebSocket connections
	mux.HandleFunc("/ws", handleWebSocket)

	// Create a new server
	server = &http.Server{
		Addr:    ":" + strconv.Itoa(serverPort),
		Handler: mux,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting web server on http://localhost:%d", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	isRunning = true
	return serverPort, nil
}

// StopServer stops the web server if it's running
func StopServer() error {
	serverMux.Lock()
	defer serverMux.Unlock()

	if !isRunning || server == nil {
		return nil
	}

	err := server.Close()
	if err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	isRunning = false
	return nil
}

// handleWebSocket handles WebSocket connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade to WebSocket:", err)
		return
	}
	defer conn.Close()

	// Register new client
	clientsMux.Lock()
	clients[conn] = true
	clientsMux.Unlock()

	// Remove client when connection closes
	defer func() {
		clientsMux.Lock()
		delete(clients, conn)
		clientsMux.Unlock()
	}()

	// Keep the connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// broadcastMessage sends a message to all connected clients
func broadcastMessage(message string) {
	clientsMux.Lock()
	defer clientsMux.Unlock()

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Printf("Error sending message to client: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// LogWriter is a custom io.Writer that captures logs and sends them to WebSocket clients
type LogWriter struct {
	underlying io.Writer // The original writer to also write logs to
}

// NewLogWriter creates a new LogWriter
func NewLogWriter(underlying io.Writer) *LogWriter {
	return &LogWriter{
		underlying: underlying,
	}
}

// Write implements io.Writer and captures logs to send to WebSocket clients
func (w *LogWriter) Write(p []byte) (n int, err error) {
	// Write to the underlying writer
	if w.underlying != nil {
		w.underlying.Write(p)
	}

	// Process the log line and send it to WebSocket clients
	broadcastMessage(string(p))

	return len(p), nil
}

// serveHTML serves the HTML for the Vue.js app with Tailwind CSS
func serveHTML(w http.ResponseWriter) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Transpiler4 Build Logs</title>
    <!-- Tailwind CSS from CDN -->
    <script src="https://cdn.tailwindcss.com"></script>
    <!-- Vue.js from CDN -->
    <script src="https://unpkg.com/vue@3/dist/vue.global.js"></script>
    <style>
        .fade-enter-active, .fade-leave-active {
            transition: opacity 0.5s;
        }
        .fade-enter-from, .fade-leave-to {
            opacity: 0;
        }
        .log-container {
            height: calc(100vh - 120px);
            overflow-y: auto;
        }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div id="app" class="container mx-auto px-4 py-8">
        <header class="mb-8">
            <h1 class="text-3xl font-bold text-gray-800">Transpiler4 Build Logs</h1>
            <p class="text-gray-600">Real-time build logs from watch mode</p>
        </header>
        
        <div class="bg-white rounded-lg shadow-md p-4 mb-4">
            <div class="flex justify-between items-center mb-2">
                <h2 class="text-xl font-semibold text-gray-700">Live Build Logs</h2>
                <div class="flex items-center">
                    <span class="flex h-3 w-3 relative mr-2">
                        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                        <span class="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
                    </span>
                    <span class="text-sm text-gray-500">{{ connectionStatus }}</span>
                </div>
            </div>
            
            <div class="log-container bg-gray-800 text-gray-100 rounded p-4 font-mono text-sm">
                <transition-group name="fade">
                    <div v-for="(log, index) in logs" :key="index" class="py-1" :class="{'border-b border-gray-700': index < logs.length - 1}">
                        {{ log }}
                    </div>
                </transition-group>
                <div v-if="logs.length === 0" class="text-gray-500 italic">
                    Waiting for build logs...
                </div>
            </div>
        </div>
        
        <div class="text-center text-gray-500 text-sm">
            <p>Transpiler4 Watch Mode</p>
        </div>
    </div>

    <script>
        const { createApp, ref, onMounted, onUnmounted } = Vue;
        
        createApp({
            setup() {
                const logs = ref([]);
                const connectionStatus = ref('Connecting...');
                let socket = null;
                
                const connectWebSocket = () => {
                    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                    const wsUrl = protocol + '//' + window.location.host + '/ws';
                    
                    socket = new WebSocket(wsUrl);
                    
                    socket.onopen = () => {
                        connectionStatus.value = 'Connected';
                        console.log('WebSocket connection established');
                    };
                    
                    socket.onmessage = (event) => {
                        logs.value.push(event.data);
                        // Auto-scroll to bottom
                        setTimeout(() => {
                            const logContainer = document.querySelector('.log-container');
                            if (logContainer) {
                                logContainer.scrollTop = logContainer.scrollHeight;
                            }
                        }, 50);
                    };
                    
                    socket.onclose = () => {
                        connectionStatus.value = 'Disconnected - Reconnecting...';
                        console.log('WebSocket connection closed, attempting to reconnect...');
                        setTimeout(connectWebSocket, 3000);
                    };
                    
                    socket.onerror = (error) => {
                        console.error('WebSocket error:', error);
                        connectionStatus.value = 'Connection error';
                    };
                };
                
                onMounted(() => {
                    connectWebSocket();
                });
                
                onUnmounted(() => {
                    if (socket) {
                        socket.close();
                    }
                });
                
                return {
                    logs,
                    connectionStatus
                };
            }
        }).mount('#app');
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
