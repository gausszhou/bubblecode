package agent

import "encoding/json"

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	Index    int              `json:"index,omitempty"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
}

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    *string    `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type Conversation struct {
	Messages []ChatMessage
	System   string
}

func NewConversation(system string) *Conversation {
	return &Conversation{
		System: system,
	}
}

func (c *Conversation) AddUserMessage(text string) {
	t := text
	c.Messages = append(c.Messages, ChatMessage{
		Role:    "user",
		Content: &t,
	})
}

func (c *Conversation) AddAssistantMessage(content string, toolCalls []ToolCall) {
	c.Messages = append(c.Messages, ChatMessage{
		Role:      "assistant",
		Content:   strPtr(content),
		ToolCalls: toolCalls,
	})
}

func (c *Conversation) AddToolResult(toolCallID, name, result string) {
	c.Messages = append(c.Messages, ChatMessage{
		Role:       "tool",
		Content:    strPtr(result),
		ToolCallID: toolCallID,
		Name:       name,
	})
}

func (c *Conversation) ToLLMMessages() []ChatMessage {
	msgs := make([]ChatMessage, 0, 1+len(c.Messages))
	msgs = append(msgs, ChatMessage{
		Role:    "system",
		Content: strPtr(c.System),
	})
	msgs = append(msgs, c.Messages...)
	return msgs
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (c *Conversation) MarshalMessages() ([]byte, error) {
	return json.Marshal(c.ToLLMMessages())
}
