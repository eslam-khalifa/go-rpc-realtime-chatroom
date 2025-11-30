package commons

type Message struct {
	UserID string
	Text   string
}

type BroadcastMessage struct {
	SenderID string
	Content  string
	IsSystem bool
}

type ClientInfo struct {
	ID       string
	Name     string
	Callback string
}

type RegisterRequest struct {
	ID       string
	Name     string
	Callback string
}

type RegisterReply struct {
	Success bool
	Message string
}

type SendMessageRequest struct {
	UserID string
	Text   string
}

type SendMessageReply struct {
	Success bool
	Message string
	Error   string
}

type ReceiveMessageRequest struct {
	Message BroadcastMessage
}

type ReceiveMessageReply struct {
	Success bool
	Error   string
}

func GetServerAddress() string {
	return "0.0.0.0:9999"
}

func GetClientCallbackPort() string {
	return "0.0.0.0:0"
}