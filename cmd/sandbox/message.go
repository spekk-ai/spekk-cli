package main

type Message struct {
	Type           string       `json:"type"`
	Text           string       `json:"text"`
	SystemPrompt   string       `json:"system_prompt"`
	SessionID      string       `json:"session_id"`
	AgentSessionID string       `json:"agent_session_id"`
	Attachments    []Attachment `json:"attachments"`
	Error          string       `json:"error"`
	Detail         string       `json:"detail"`
	Protocol       string       `json:"protocol"`
}

const (
	MessageTypeMessage          = "message"
	MessageTypeCancel           = "cancel"
	MessageTypeHeartbeat        = "heartbeat"
	MessageTypeHeartbeatAck     = "heartbeat_ack"
	MessageTypeStream           = "stream"
	MessageTypeResult           = "result"
	MessageTypeError            = "error"
	MessageTypeConversationOpen = "conversation_open"
	MessageTypeWelcome          = "welcome"
)

type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mimetype"`
}
