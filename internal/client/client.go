package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type MessagesRequest struct {
	Model     string           `json:"model"`
	Messages  []Message        `json:"messages"`
	System    string           `json:"system,omitempty"`
	MaxTokens int              `json:"max_tokens"`
	Stream    bool             `json:"stream"`
	Tools     []ToolDefinition `json:"tools,omitempty"`
}

type ContentBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id,omitempty"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type MessagesResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	Usage        Usage          `json:"usage"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// SSE event types
type SSEEvent struct {
	Event string
	Data  string
}

type StreamMessageStart struct {
	Type    string           `json:"type"`
	Message MessagesResponse `json:"message"`
}

type StreamContentBlockStart struct {
	Type         string       `json:"type"`
	Index        int          `json:"index"`
	ContentBlock ContentBlock `json:"content_block"`
}

type StreamContentBlockDelta struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type        string `json:"type"`
		Text        string `json:"text,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
	} `json:"delta"`
}

type StreamMessageDelta struct {
	Type  string `json:"type"`
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage *Usage `json:"usage,omitempty"`
}

type StreamCallback struct {
	OnText           func(text string)
	OnToolUseStart   func(id, name string)
	OnToolUseInput   func(partialJSON string)
	OnMessageStart   func(resp *MessagesResponse)
	OnMessageDelta   func(stopReason string, usage *Usage)
	OnContentBlockStop func(index int)
	OnError          func(err error)
}

func (c *Client) SendMessageStream(req *MessagesRequest, cb *StreamCallback) (*MessagesResponse, error) {
	req.Stream = true
	if req.MaxTokens == 0 {
		req.MaxTokens = 16384
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(errBody))
	}

	return c.parseSSEStream(resp.Body, cb)
}

func (c *Client) parseSSEStream(reader io.Reader, cb *StreamCallback) (*MessagesResponse, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var result MessagesResponse
	var currentEvent string
	var toolInputs = make(map[int]*strings.Builder)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		switch currentEvent {
		case "message_start":
			var msg StreamMessageStart
			if err := json.Unmarshal([]byte(data), &msg); err == nil {
				result = msg.Message
				if cb != nil && cb.OnMessageStart != nil {
					cb.OnMessageStart(&result)
				}
			}

		case "content_block_start":
			var block StreamContentBlockStart
			if err := json.Unmarshal([]byte(data), &block); err == nil {
				for len(result.Content) <= block.Index {
					result.Content = append(result.Content, ContentBlock{})
				}
				result.Content[block.Index] = block.ContentBlock
				if block.ContentBlock.Type == "tool_use" {
					toolInputs[block.Index] = &strings.Builder{}
					if cb != nil && cb.OnToolUseStart != nil {
						cb.OnToolUseStart(block.ContentBlock.ID, block.ContentBlock.Name)
					}
				}
			}

		case "content_block_delta":
			var delta StreamContentBlockDelta
			if err := json.Unmarshal([]byte(data), &delta); err == nil {
				switch delta.Delta.Type {
				case "text_delta":
					if delta.Index < len(result.Content) {
						result.Content[delta.Index].Text += delta.Delta.Text
					}
					if cb != nil && cb.OnText != nil {
						cb.OnText(delta.Delta.Text)
					}
				case "input_json_delta":
					if sb, ok := toolInputs[delta.Index]; ok {
						sb.WriteString(delta.Delta.PartialJSON)
					}
					if cb != nil && cb.OnToolUseInput != nil {
						cb.OnToolUseInput(delta.Delta.PartialJSON)
					}
				}
			}

		case "content_block_stop":
			var stop struct {
				Index int `json:"index"`
			}
			if err := json.Unmarshal([]byte(data), &stop); err == nil {
				if sb, ok := toolInputs[stop.Index]; ok {
					if stop.Index < len(result.Content) {
						result.Content[stop.Index].Input = json.RawMessage(sb.String())
					}
					delete(toolInputs, stop.Index)
				}
				if cb != nil && cb.OnContentBlockStop != nil {
					cb.OnContentBlockStop(stop.Index)
				}
			}

		case "message_delta":
			var delta StreamMessageDelta
			if err := json.Unmarshal([]byte(data), &delta); err == nil {
				result.StopReason = delta.Delta.StopReason
				if delta.Usage != nil {
					result.Usage = *delta.Usage
				}
				if cb != nil && cb.OnMessageDelta != nil {
					cb.OnMessageDelta(delta.Delta.StopReason, delta.Usage)
				}
			}

		case "error":
			var errData struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(data), &errData); err == nil {
				apiErr := fmt.Errorf("stream error: %s", errData.Error.Message)
				if cb != nil && cb.OnError != nil {
					cb.OnError(apiErr)
				}
				return nil, apiErr
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read stream: %w", err)
	}

	return &result, nil
}

// DeviceCodeRequest for login flow
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type DeviceTokenResponse struct {
	Status   string `json:"status"`
	APIToken string `json:"api_token,omitempty"`
	Username string `json:"username,omitempty"`
	Plan     string `json:"plan,omitempty"`
	Error    string `json:"error,omitempty"`
}

func (c *Client) RequestDeviceCode() (*DeviceCodeResponse, error) {
	resp, err := c.httpClient.Post(c.baseURL+"/auth/device/code", "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device code error (status %d): %s", resp.StatusCode, string(body))
	}

	var result DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func (c *Client) PollDeviceToken(deviceCode string) (*DeviceTokenResponse, error) {
	body, _ := json.Marshal(map[string]string{"device_code": deviceCode})
	resp, err := c.httpClient.Post(c.baseURL+"/auth/device/token", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("poll device token: %w", err)
	}
	defer resp.Body.Close()

	var result DeviceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}
