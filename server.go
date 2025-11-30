package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"

	"chatroom/commons"
)

type ChatServer struct {
	clients map[string]*commons.ClientInfo
	clientsMutex sync.RWMutex
	broadcastChannel chan commons.BroadcastMessage
	shutdownChannel chan struct{}
	wg sync.WaitGroup
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients:         make(map[string]*commons.ClientInfo),
		broadcastChannel: make(chan commons.BroadcastMessage, 100),
		shutdownChannel: make(chan struct{}),
	}
}

func (s *ChatServer) Register(req commons.RegisterRequest, reply *commons.RegisterReply) error {
	if req.ID == "" {
		reply.Success = false
		reply.Message = "Client ID cannot be empty"
		return fmt.Errorf("invalid client ID")
	}

	if req.Callback == "" {
		reply.Success = false
		reply.Message = "Callback address cannot be empty"
		return fmt.Errorf("invalid callback address")
	}

	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	if _, exists := s.clients[req.ID]; exists {
		reply.Success = false
		reply.Message = fmt.Sprintf("Client ID %s already registered", req.ID)
		return fmt.Errorf("client already registered: %s", req.ID)
	}

	s.clients[req.ID] = &commons.ClientInfo{
		ID:       req.ID,
		Name:     req.Name,
		Callback: req.Callback,
	}

	log.Printf("[JOIN] User %s (ID: %s) joined the chat from %s", req.Name, req.ID, req.Callback)

	joinMessage := commons.BroadcastMessage{
		SenderID: req.ID,
		Content:  fmt.Sprintf("User %s joined the chat", req.ID),
		IsSystem: true,
	}

	select {
	case s.broadcastChannel <- joinMessage:
	default:
		log.Printf("[WARN] Broadcast channel full, dropping join message for %s", req.ID)
	}

	reply.Success = true
	reply.Message = fmt.Sprintf("Successfully registered as %s", req.ID)
	return nil
}

func (s *ChatServer) Unregister(clientID string, reply *string) error {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	client, exists := s.clients[clientID]
	if !exists {
		*reply = "Client not found"
		return fmt.Errorf("client not found: %s", clientID)
	}

	log.Printf("[LEAVE] User %s (ID: %s) left the chat", client.Name, clientID)

	delete(s.clients, clientID)

	leaveMessage := commons.BroadcastMessage{
		SenderID: clientID,
		Content:  fmt.Sprintf("User %s left the chat", clientID),
		IsSystem: true,
	}

	select {
	case s.broadcastChannel <- leaveMessage:
	default:
		log.Printf("[WARN] Broadcast channel full, dropping leave message for %s", clientID)
	}

	*reply = "Successfully unregistered"
	return nil
}

func (s *ChatServer) SendMessage(req commons.SendMessageRequest, reply *commons.SendMessageReply) error {
	if req.Text == "" {
		reply.Success = false
		reply.Error = "Message text cannot be empty"
		return fmt.Errorf("empty message text")
	}

	s.clientsMutex.RLock()
	client, exists := s.clients[req.UserID]
	s.clientsMutex.RUnlock()

	if !exists {
		reply.Success = false
		reply.Error = "Client not registered"
		return fmt.Errorf("client not registered: %s", req.UserID)
	}

	formattedMessage := fmt.Sprintf("%s: %s", client.Name, req.Text)
	log.Printf("[BROADCAST] %s", formattedMessage)

	broadcastMsg := commons.BroadcastMessage{
		SenderID: req.UserID,
		Content:  formattedMessage,
		IsSystem: false,
	}

	select {
	case s.broadcastChannel <- broadcastMsg:
		reply.Success = true
		reply.Message = "Message sent successfully"
		return nil
	case <-time.After(5 * time.Second):
		reply.Success = false
		reply.Error = "Broadcast channel timeout"
		return fmt.Errorf("broadcast channel timeout")
	}
}

func (s *ChatServer) broadcaster() {
	defer s.wg.Done()

	log.Println("[BROADCASTER] Broadcaster goroutine started")

	for {
		select {
		case msg := <-s.broadcastChannel:
			s.clientsMutex.RLock()
			clients := make([]*commons.ClientInfo, 0, len(s.clients))
			for _, client := range s.clients {
				if client.ID != msg.SenderID {
					clients = append(clients, client)
				}
			}
			s.clientsMutex.RUnlock()

			if len(clients) > 0 {
				var wg sync.WaitGroup
				for _, client := range clients {
					wg.Add(1)
					go s.sendToClient(client, msg, &wg)
				}
				wg.Wait()
			}

		case <-s.shutdownChannel:
			log.Println("[BROADCASTER] Broadcaster goroutine shutting down")
			return
		}
	}
}

func (s *ChatServer) sendToClient(client *commons.ClientInfo, msg commons.BroadcastMessage, wg *sync.WaitGroup) {
	defer wg.Done()

	rpcClient, err := rpc.Dial("tcp", client.Callback)
	if err != nil {
		log.Printf("[ERROR] Failed to connect to client %s at %s: %v", client.ID, client.Callback, err)
		go s.cleanupClient(client.ID)
		return
	}
	defer rpcClient.Close()

	var reply commons.ReceiveMessageReply
	err = rpcClient.Call("ClientCallback.ReceiveMessage", commons.ReceiveMessageRequest{Message: msg}, &reply)
	if err != nil {
		log.Printf("[ERROR] Failed to send message to client %s: %v", client.ID, err)
		go s.cleanupClient(client.ID)
		return
	}

	if !reply.Success {
		log.Printf("[ERROR] Client %s reported error receiving message: %s", client.ID, reply.Error)
	}
}

func (s *ChatServer) cleanupClient(clientID string) {
	var reply string
	err := s.Unregister(clientID, &reply)
	if err != nil {
		log.Printf("[ERROR] Failed to cleanup client %s: %v", clientID, err)
	}
}

func (s *ChatServer) Shutdown() {
	log.Println("[SHUTDOWN] Initiating server shutdown...")
	close(s.shutdownChannel)
	s.wg.Wait()
	log.Println("[SHUTDOWN] Server shutdown complete")
}

func main() {
	server := NewChatServer()

	err := rpc.Register(server)
	if err != nil {
		log.Fatalf("Failed to register RPC server: %v", err)
	}

	server.wg.Add(1)
	go server.broadcaster()

	addr := commons.GetServerAddress()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	defer listener.Close()

	log.Printf("[SERVER] Real-time RPC chat server running on %s", addr)
	log.Println("[SERVER] Waiting for client connections...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-server.shutdownChannel:
				log.Println("[SERVER] Server is shutting down, stopping accept loop")
				return
			default:
				log.Printf("[ERROR] Failed to accept connection: %v", err)
				continue
			}
		}

		go func(c net.Conn) {
			log.Printf("[CONNECTION] New connection from %s", c.RemoteAddr())
			rpc.ServeConn(c)
			log.Printf("[CONNECTION] Connection from %s closed", c.RemoteAddr())
		}(conn)
	}
}