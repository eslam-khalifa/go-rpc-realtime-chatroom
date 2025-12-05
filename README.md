# go-rpc-realtime-chatroom

A **production-ready, real-time RPC chat system** implemented in **Go** using the standard `net/rpc` package. The system features concurrent message broadcasting, client callbacks, and robust error handling.

---

## 🎥 Demo Video

Watch the project in action here:

[👉 Demo Video](https://drive.google.com/file/d/1AqqoJU9RDAT40hrKHwZNgJxQjpf8fSS6/view?usp=drive_link)

---

## 🎯 Features

- **Real-time Broadcasting**: Messages from any client are instantly broadcast to all other connected clients
- **Concurrent Architecture**: Uses goroutines, channels, and Mutex for safe concurrent operations
- **Client Callbacks**: Clients expose RPC servers to receive messages asynchronously (no polling)
- **Join/Leave Notifications**: Automatic system messages when clients connect or disconnect
- **Race Condition Safe**: Fully compatible with `go run -race` for race detection
- **Graceful Shutdown**: Proper cleanup of resources and connections
- **Error Handling**: Comprehensive error handling for network failures, timeouts, and disconnects

---

## 🏗️ Architecture

### Server Architecture

The server uses a **broadcaster pattern** with the following components:

1. **Client Registry**: A Mutex-protected map storing all connected clients
2. **Broadcast Channel**: A buffered channel that queues messages for broadcasting
3. **Broadcaster Goroutine**: Continuously reads from the broadcast channel and sends messages to all clients
4. **Per-Connection Goroutines**: Each client connection is handled in a separate goroutine

### Client Architecture

Each client consists of:

1. **RPC Client**: Connects to the main server to send messages
2. **RPC Callback Server**: Exposes an RPC endpoint to receive messages from the server
3. **Message Receiver Goroutine**: Displays incoming messages in real-time
4. **Input Handler**: Reads user input and sends messages to the server

### Message Flow

```
Client A → SendMessage RPC → Server
                              ↓
                         Broadcast Channel
                              ↓
                    Broadcaster Goroutine
                              ↓
              ┌───────────────┼───────────────┐
              ↓               ↓               ↓
        Client B         Client C         Client D
    (via callback)    (via callback)    (via callback)
```

---

## 🗂️ Project Structure

```
go-rpc-realtime-chatroom/
├── client.go              # Client implementation with RPC callback server
├── server.go              # Server with broadcasting and client management
├── go.mod                 # Go module definition
├── commons/
│   └── shared.go          # Shared types and constants
├── README.md              # This file
└── LICENSE                # License file
```

---

## ⚙️ Requirements

- **Go 1.20+** (or later)
- Network connectivity for client-server communication

---

## 🚀 Quick Start

### 1️⃣ Start the Server

In one terminal, run:

```bash
go run server.go
```

Expected output:

```
[SERVER] Real-time RPC chat server running on 0.0.0.0:9999
[SERVER] Waiting for client connections...
```

### 2️⃣ Start Multiple Clients

In separate terminals, run:

```bash
go run client.go
```

For each client, you'll be prompted:

```
Enter your name: Alice
Welcome, Alice! (ID: Alice-1234567890)
Connected to chat server at 0.0.0.0:9999
Type your messages below (type 'exit' to quit)

You: 
```

### 3️⃣ Chat in Real-Time

When any client sends a message, it's immediately broadcast to all other clients:

**Client 1 (Alice):**
```
You: Hello everyone!
```

**Client 2 (Bob) sees:**
```
Alice: Hello everyone!
You: 
```

**Client 3 (Charlie) sees:**
```
Alice: Hello everyone!
You: 
```

**Server logs:**
```
[JOIN] User Alice (ID: Alice-1234567890) joined the chat from 127.0.0.1:54321
[JOIN] User Bob (ID: Bob-1234567891) joined the chat from 127.0.0.1:54322
[JOIN] User Charlie (ID: Charlie-1234567892) joined the chat from 127.0.0.1:54323
[BROADCAST] Alice: Hello everyone!
```

---

## 🔧 Concurrency Features

### Server-Side Concurrency

- **Mutex Protection**: The client registry is protected by `sync.RWMutex` for safe concurrent access
- **Broadcast Channel**: Messages are queued in a buffered channel, preventing blocking
- **Broadcaster Goroutine**: Single goroutine handles all message broadcasting
- **Per-Client Goroutines**: Each client callback is sent in a separate goroutine

### Client-Side Concurrency

- **Message Receiver Goroutine**: Continuously receives and displays messages
- **Input Handler**: Separate goroutine handles user input
- **Callback Server**: Handles multiple concurrent RPC calls from the server

### Race Condition Safety

The code is designed to be race-condition free. Test with:

```bash
go run -race server.go
go run -race client.go
```

---

## 📡 RPC Methods

### Server Methods

#### `ChatServer.Register`
Registers a new client with the server.

**Request:**
```go
type RegisterRequest struct {
    ID       string // Unique client identifier
    Name     string // Display name
    Callback string // RPC callback address (host:port)
}
```

**Reply:**
```go
type RegisterReply struct {
    Success bool
    Message string
}
```

#### `ChatServer.Unregister`
Unregisters a client when they disconnect.

**Request:** `string` (client ID)

**Reply:** `string` (status message)

#### `ChatServer.SendMessage`
Sends a message that will be broadcast to all other clients.

**Request:**
```go
type SendMessageRequest struct {
    UserID string // ID of the sending client
    Text   string // Message text
}
```

**Reply:**
```go
type SendMessageReply struct {
    Success bool
    Message string
    Error   string
}
```

### Client Callback Methods

#### `ClientCallback.ReceiveMessage`
Called by the server to deliver messages to the client.

**Request:**
```go
type ReceiveMessageRequest struct {
    Message BroadcastMessage
}
```

**Reply:**
```go
type ReceiveMessageReply struct {
    Success bool
    Error   string
}
```

---

## 🛡️ Error Handling

The system handles various error scenarios:

- **Client Disconnects**: Automatically detected and cleaned up
- **Network Timeouts**: 5-second timeout for broadcast channel operations
- **Write Failures**: Failed client callbacks trigger automatic unregistration
- **Concurrent Broadcasts**: Buffered channels prevent blocking
- **Invalid Input**: Validation for empty messages and missing client IDs

---

## 📝 Logging

The system provides detailed logging:

- `[SERVER]` - Server lifecycle events
- `[JOIN]` - Client registration
- `[LEAVE]` - Client disconnection
- `[BROADCAST]` - Message broadcasting
- `[CLIENT]` - Client events
- `[ERROR]` - Error conditions
- `[WARN]` - Warning conditions

---

## 🧪 Testing with Multiple Clients

1. Start the server in one terminal
2. Start Client 1:
   ```bash
   go run client.go
   # Enter name: Alice
   ```
3. Start Client 2:
   ```bash
   go run client.go
   # Enter name: Bob
   ```
4. Start Client 3:
   ```bash
   go run client.go
   # Enter name: Charlie
   ```

All clients will see join messages and can chat in real-time!

---

## 🔍 Code Quality

This implementation follows production-grade standards:

- ✅ **Clean Architecture**: Modular, maintainable design
- ✅ **Error Handling**: Comprehensive error handling throughout
- ✅ **Concurrency Safety**: Mutex protection and channel-based communication
- ✅ **Resource Management**: Proper cleanup and graceful shutdown
- ✅ **Documentation**: Extensive comments explaining the architecture
- ✅ **Idiomatic Go**: Follows Go best practices and conventions

---

## 🚨 Important Notes

1. **Client Callback Address**: Clients automatically bind to `127.0.0.1:0` (OS-assigned port). For production, you may need to configure this based on your network setup.

2. **Client ID Generation**: Currently uses `name-timestamp` format. In production, use UUIDs or a more robust ID generation scheme.

3. **Network Configuration**: The server listens on `0.0.0.0:9999` by default. Modify `commons/shared.go` to change the address.

4. **Graceful Shutdown**: Use Ctrl+C to gracefully disconnect. The client will unregister from the server.

---

## 📦 Files Explained

| File                | Description                                                                 |
| ------------------- | --------------------------------------------------------------------------- |
| `server.go`         | RPC server with broadcasting, client management, and concurrent message delivery |
| `client.go`         | RPC client with callback server for real-time message reception             |
| `commons/shared.go` | Shared types, structs, and constants used by both server and client         |

---

## 🧾 License

MIT License — free to use and modify.
See the [LICENSE](LICENSE) file for details.

---

## 🎓 Learning Resources

This implementation demonstrates:

- Go RPC programming with `net/rpc`
- Concurrent programming with goroutines and channels
- Mutex-based synchronization
- Channel-based message passing
- Graceful shutdown patterns
- Error handling in concurrent systems
- Real-time communication patterns

---

## 🤝 Contributing

Contributions are welcome! Please ensure:

- Code follows Go best practices
- All concurrent operations are race-condition safe
- Error handling is comprehensive
- Code is well-commented

---

**Enjoy real-time chatting! 🚀**