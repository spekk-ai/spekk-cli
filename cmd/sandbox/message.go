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
	Capabilities   []string     `json:"capabilities"`

	// The sign-in terminal frames (terminal.go). No field names a program.
	TerminalID string `json:"terminal_id"`
	Data       string `json:"data"`
	Cols       int    `json:"cols"`
	Rows       int    `json:"rows"`
	AgentID    string `json:"agent_id"`
	UserID     string `json:"user_id"`
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
