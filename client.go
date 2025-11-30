package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"chatroom/commons"
)

type ClientCallback struct {
	messageChannel chan commons.BroadcastMessage
	shutdownChannel chan struct{}
	wg sync.WaitGroup
}

func NewClientCallback() *ClientCallback {
	return &ClientCallback{
		messageChannel:  make(chan commons.BroadcastMessage, 100),
		shutdownChannel: make(chan struct{}),
	}
}

func (c *ClientCallback) ReceiveMessage(req commons.ReceiveMessageRequest, reply *commons.ReceiveMessageReply) error {
	select {
	case c.messageChannel <- req.Message:
		reply.Success = true
		return nil
	case <-time.After(1 * time.Second):
		reply.Success = false
		reply.Error = "Message channel timeout"
		return fmt.Errorf("message channel timeout")
	}
}

func (c *ClientCallback) messageReceiver() {
	defer c.wg.Done()

	for {
		select {
		case msg := <-c.messageChannel:
			if msg.IsSystem {
				fmt.Printf("\n[SYSTEM] %s\n", msg.Content)
			} else {
				fmt.Printf("\n%s\n", msg.Content)
			}
			fmt.Print("You: ")

		case <-c.shutdownChannel:
			return
		}
	}
}

func (c *ClientCallback) Shutdown() {
	close(c.shutdownChannel)
	c.wg.Wait()
}

type ChatClient struct {
	serverAddr string
	clientID   string
	name       string
	callback   *ClientCallback
	rpcClient  *rpc.Client
	listener   net.Listener
	callbackAddr string
}

func NewChatClient(serverAddr, clientID, name string) *ChatClient {
	return &ChatClient{
		serverAddr: serverAddr,
		clientID:   clientID,
		name:       name,
		callback:   NewClientCallback(),
	}
}

func (c *ChatClient) startCallbackServer() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start callback server: %v", err)
	}

	c.listener = listener
	c.callbackAddr = listener.Addr().String()

	rpcServer := rpc.NewServer()
	err = rpcServer.Register(c.callback)
	if err != nil {
		listener.Close()
		return fmt.Errorf("failed to register callback server: %v", err)
	}

	c.callback.wg.Add(1)
	go c.callback.messageReceiver()

	c.callback.wg.Add(1)
	go func() {
		defer c.callback.wg.Done()
		for {
			select {
			case <-c.callback.shutdownChannel:
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					select {
					case <-c.callback.shutdownChannel:
						return
					default:
						log.Printf("[ERROR] Failed to accept callback connection: %v", err)
						continue
					}
				}
				go rpcServer.ServeConn(conn)
			}
		}
	}()

	log.Printf("[CLIENT] Callback server started on %s", c.callbackAddr)
	return nil
}

func (c *ChatClient) connect() error {
	client, err := rpc.Dial("tcp", c.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}
	c.rpcClient = client
	return nil
}

func (c *ChatClient) register() error {
	req := commons.RegisterRequest{
		ID:       c.clientID,
		Name:     c.name,
		Callback: c.callbackAddr,
	}

	var reply commons.RegisterReply
	err := c.rpcClient.Call("ChatServer.Register", req, &reply)
	if err != nil {
		return fmt.Errorf("registration failed: %v", err)
	}

	if !reply.Success {
		return fmt.Errorf("registration failed: %s", reply.Message)
	}

	log.Printf("[CLIENT] Successfully registered as %s", c.clientID)
	return nil
}

func (c *ChatClient) unregister() error {
	if c.rpcClient == nil {
		return nil
	}

	var reply string
	err := c.rpcClient.Call("ChatServer.Unregister", c.clientID, &reply)
	if err != nil {
		return fmt.Errorf("unregistration failed: %v", err)
	}

	log.Printf("[CLIENT] Successfully unregistered: %s", reply)
	return nil
}

func (c *ChatClient) sendMessage(text string) error {
	req := commons.SendMessageRequest{
		UserID: c.clientID,
		Text:   text,
	}

	var reply commons.SendMessageReply
	err := c.rpcClient.Call("ChatServer.SendMessage", req, &reply)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	if !reply.Success {
		return fmt.Errorf("message send failed: %s", reply.Error)
	}

	return nil
}

func (c *ChatClient) run() error {
	if err := c.startCallbackServer(); err != nil {
		return err
	}
	defer c.listener.Close()

	time.Sleep(100 * time.Millisecond)

	if err := c.connect(); err != nil {
		return err
	}
	defer c.rpcClient.Close()

	if err := c.register(); err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	inputDone := make(chan struct{})
	go func() {
		defer close(inputDone)
		reader := bufio.NewReader(os.Stdin)

		for {
			fmt.Print("You: ")
			text, err := reader.ReadString('\n')
			if err != nil {
				log.Printf("[ERROR] Failed to read input: %v", err)
				return
			}

			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}

			if text == "exit" {
				fmt.Println("\n[CLIENT] Exiting chat...")
				return
			}

			if err := c.sendMessage(text); err != nil {
				log.Printf("[ERROR] Failed to send message: %v", err)
			}
		}
	}()

	select {
	case <-inputDone:
	case <-sigChan:
		fmt.Println("\n[CLIENT] Received interrupt signal, shutting down...")
	}

	if err := c.unregister(); err != nil {
		log.Printf("[ERROR] Failed to unregister: %v", err)
	}

	c.callback.Shutdown()
	return nil
}

func main() {
	serverAddr := commons.GetServerAddress()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("Failed to read name:", err)
	}
	name = strings.TrimSpace(name)

	if name == "" {
		log.Fatal("Name cannot be empty")
	}

	clientID := fmt.Sprintf("%s-%d", name, time.Now().UnixNano())

	fmt.Printf("Welcome, %s! (ID: %s)\n", name, clientID)
	fmt.Println("Connected to chat server at", serverAddr)
	fmt.Println("Type your messages below (type 'exit' to quit)")
	fmt.Println()

	client := NewChatClient(serverAddr, clientID, name)
	if err := client.run(); err != nil {
		log.Fatalf("Client error: %v", err)
	}

	fmt.Println("[CLIENT] Client shutdown complete")
}