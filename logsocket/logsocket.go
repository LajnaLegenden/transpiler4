package logsocket

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/LajnaLegenden/transpiler4/helpers"
	"github.com/gorilla/websocket"
)

// LogMessage represents a structured log message
type LogMessage struct {
	Package string `json:"package"`
	Message string `json:"message"`
	Time    int64  `json:"time"`
}

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
func broadcastMessage(message LogMessage) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling log message: %v", err)
		return
	}

	clientsMux.Lock()
	defer clientsMux.Unlock()

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			log.Printf("Error sending message to client: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// SendPackageLog sends a log message associated with a specific package
func SendPackageLog(packageName, message string, timestamp int64) {
	logMsg := LogMessage{
		Package: packageName,
		Message: message,
		Time:    timestamp,
	}
	broadcastMessage(logMsg)
}

// LogWriter is a custom io.Writer that captures logs and sends them to WebSocket clients
type LogWriter struct {
	underlying  io.Writer // The original writer to also write logs to
	packageName string    // The package this writer is associated with
}

// NewLogWriter creates a new LogWriter for a specific package
func NewLogWriter(underlying io.Writer, packageName string) *LogWriter {
	return &LogWriter{
		underlying:  underlying,
		packageName: packageName,
	}
}

// Write implements io.Writer and captures logs to send to WebSocket clients
func (w *LogWriter) Write(p []byte) (n int, err error) {
	// Write to the underlying writer
	if w.underlying != nil {
		w.underlying.Write(p)
	}

	// Get current timestamp in milliseconds
	timestamp := helpers.GetCurrentTimeMillis()

	// Create a structured log message and send it to WebSocket clients
	logMsg := LogMessage{
		Package: w.packageName,
		Message: string(p),
		Time:    timestamp,
	}
	broadcastMessage(logMsg)

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
            height: calc(100vh - 200px);
            overflow-y: auto;
        }
        .tab {
            @apply px-4 py-2 text-sm font-medium text-center cursor-pointer;
            @apply border-b-2 transition-colors duration-200;
        }
        .tab.active {
            @apply border-blue-500 text-blue-600;
        }
        .tab:not(.active) {
            @apply border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300;
        }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div id="app" class="container mx-auto px-4 py-8">
        <header class="mb-6">
            <h1 class="text-3xl font-bold text-gray-800">MTCLI Build Logs</h1>
            <p class="text-gray-600">Real-time build logs from watch mode</p>
        </header>
        
        <div class="bg-white rounded-lg shadow-md p-4 mb-4">
            <div class="flex justify-between items-center mb-2">
                <h2 class="text-xl font-semibold text-gray-700">Live Build Logs</h2>
                <div class="flex items-center">
                    <span class="flex h-4 w-4 relative mr-2">
                        <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                        <span class="relative inline-flex rounded-full h-4 w-4 bg-green-500"></span>
                    </span>
                    <span class="text-sm font-medium text-gray-600">{{ connectionStatus }}</span>
                </div>
            </div>
            
            <!-- Tabs -->
            <div class="border-b border-gray-200 mb-4">
                <nav class="flex -mb-px overflow-x-auto">
                    <div 
                        v-for="tab in tabs" 
                        :key="tab"
                        @click="activeTab = tab"
                        class="tab"
                        :class="{'active': activeTab === tab}"
                    >
                        {{ tab }}
                        <span v-if="getLogCountForPackage(tab) > 0" 
                              class="ml-1 bg-blue-100 text-blue-600 text-xs font-semibold px-1.5 py-0.5 rounded-full">
                            {{ getLogCountForPackage(tab) }}
                        </span>
                    </div>
                </nav>
            </div>
            
            <!-- Log Display -->
            <div class="log-container bg-gray-800 text-gray-100 rounded p-4 font-mono text-sm">
                <transition-group name="fade">
                    <div v-for="log in filteredLogs" :key="log.id" class="py-1" :class="{'border-b border-gray-700': log !== filteredLogs[filteredLogs.length - 1]}">
                        <span class="text-gray-400 mr-2">{{ formatTime(log.time) }}</span>
                        {{ log.message }}
                    </div>
                </transition-group>
                <div v-if="filteredLogs.length === 0" class="text-gray-500 italic">
                    No logs available for {{ activeTab }}
                </div>
            </div>
        </div>
        
        <div class="text-center text-gray-500 text-sm mt-4 flex justify-between">
            <div>
                <span class="font-semibold">Total Messages:</span> {{ allLogs.length }}
            </div>
            <div>
                <button @click="clearLogs" class="px-3 py-1 bg-red-100 text-red-700 rounded hover:bg-red-200 transition">
                    Clear All Logs
                </button>
            </div>
        </div>
    </div>

    <script>
        const { createApp, ref, computed, onMounted, onUnmounted, watch } = Vue;
        
        createApp({
            setup() {
                const allLogs = ref([]);
                const connectionStatus = ref('Connecting...');
                const activeTab = ref('All');
                let nextId = 0;
                let socket = null;
                
                // Compute unique package tabs
                const tabs = computed(() => {
                    const packages = ['All'];
                    allLogs.value.forEach(log => {
                        if (!packages.includes(log.package)) {
                            packages.push(log.package);
                        }
                    });
                    return packages;
                });
                
                // Filter logs based on active tab
                const filteredLogs = computed(() => {
                    if (activeTab.value === 'All') {
                        return allLogs.value;
                    }
                    return allLogs.value.filter(log => log.package === activeTab.value);
                });
                
                // Get log count for a specific package
                const getLogCountForPackage = (packageName) => {
                    if (packageName === 'All') {
                        return allLogs.value.length;
                    }
                    return allLogs.value.filter(log => log.package === packageName).length;
                };
                
                // Format timestamp
                const formatTime = (timestamp) => {
                    const date = new Date(timestamp);
                    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
                };
                
                // Clear all logs
                const clearLogs = () => {
                    allLogs.value = [];
                };
                
                const connectWebSocket = () => {
                    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                    const wsUrl = protocol + '//' + window.location.host + '/ws';
                    
                    socket = new WebSocket(wsUrl);
                    
                    socket.onopen = () => {
                        connectionStatus.value = 'Connected';
                        console.log('WebSocket connection established');
                    };
                    
                    socket.onmessage = (event) => {
                        try {
                            const logData = JSON.parse(event.data);
                            
                            // Add unique ID for Vue's key tracking
                            const logEntry = {
                                id: nextId++,
                                package: logData.package || 'Unknown',
                                message: logData.message,
                                time: logData.time || Date.now()
                            };
                            
                            allLogs.value.push(logEntry);
                            
                            // Auto-switch to new package tab when it first appears
                            if (tabs.value.length === 2 && tabs.value.includes(logEntry.package) && activeTab.value === 'All') {
                                activeTab.value = logEntry.package;
                            }
                            
                            // Auto-scroll to bottom
                            setTimeout(() => {
                                const logContainer = document.querySelector('.log-container');
                                if (logContainer) {
                                    logContainer.scrollTop = logContainer.scrollHeight;
                                }
                            }, 50);
                        } catch (e) {
                            console.error('Error parsing WebSocket message:', e);
                            // Handle legacy plain text format
                            allLogs.value.push({
                                id: nextId++,
                                package: 'Unknown',
                                message: event.data,
                                time: Date.now()
                            });
                        }
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
                    allLogs,
                    filteredLogs,
                    tabs,
                    activeTab,
                    connectionStatus,
                    getLogCountForPackage,
                    formatTime,
                    clearLogs
                };
            }
        }).mount('#app');
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
