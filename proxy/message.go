package proxy

import (
	jsoniter "github.com/json-iterator/go"
)

type MessageType string

const (
	MessageTypeHTTP      MessageType = "http"
	MessageTypeSSE       MessageType = "sse"
	MessageTypeWebsocket MessageType = "websocket"
)

type Message struct {
	typ  MessageType
	data any
}

func (m *Message) Type() MessageType {
	return m.typ
}

func (m *Message) HTTP() *HTTPMessage {
	return m.data.(*HTTPMessage)
}

func (m *Message) WebSocket() *WebsocketMessage {
	return m.data.(*WebsocketMessage)
}

func (m *Message) SSE() *SSEMessage {
	return m.data.(*SSEMessage)
}

type HTTPMessage struct {
	Id         uint64            `json:"id,omitempty"`
	Method     string            `json:"method,omitempty"`
	Type       string            `json:"type,omitempty"`
	Time       uint16            `json:"time,omitempty"` // ms
	Size       uint16            `json:"size,omitempty"`
	Status     uint16            `json:"status,omitempty"`
	Url        string            `json:"url,omitempty"`
	RemoteAddr string            `json:"remote_addr,omitempty"`
	ReqHeader  map[string]string `json:"req_header,omitempty"`
	ReqCookie  map[string]string `json:"req_cookie,omitempty"`
	ReqTls     map[string]string `json:"req_tls,omitempty"`
	ReqBody    string            `json:"req_body,omitempty"`
	RespHeader map[string]string `json:"resp_header,omitempty"`
	RespCookie map[string]string `json:"resp_cookie,omitempty"`
	RespTls    map[string]string `json:"resp_tls,omitempty"`
	RespBody   string            `json:"resp_body,omitempty"`
}

func (m *HTTPMessage) String() string {
	message, _ := jsoniter.MarshalToString(m)
	return message
}

type WebsocketMessage struct {
	Id         uint64            `json:"id,omitempty"`
	Method     string            `json:"method,omitempty"`
	Type       string            `json:"type,omitempty"`
	Time       uint16            `json:"time,omitempty"` // ms
	Size       uint16            `json:"size,omitempty"`
	Status     uint16            `json:"status,omitempty"`
	Url        string            `json:"url,omitempty"`
	RemoteAddr string            `json:"remote_addr,omitempty"`
	ReqHeader  map[string]string `json:"req_header,omitempty"`
	ReqCookie  map[string]string `json:"req_cookie,omitempty"`
	ReqTls     map[string]string `json:"req_tls,omitempty"`
	ReqBody    chan []byte       `json:"req_body,omitempty"`
	RespHeader map[string]string `json:"resp_header,omitempty"`
	RespCookie map[string]string `json:"resp_cookie,omitempty"`
	RespTls    map[string]string `json:"resp_tls,omitempty"`
	RespBody   chan []byte       `json:"resp_body,omitempty"`
}

func (m *WebsocketMessage) String() string {
	message, _ := jsoniter.MarshalToString(m)
	return message
}

type SSEMessage struct {
	Id         uint64            `json:"id,omitempty"`
	Method     string            `json:"method,omitempty"`
	Type       string            `json:"type,omitempty"`
	Time       uint16            `json:"time,omitempty"` // ms
	Size       uint16            `json:"size,omitempty"`
	Status     uint16            `json:"status,omitempty"`
	Url        string            `json:"url,omitempty"`
	RemoteAddr string            `json:"remote_addr,omitempty"`
	ReqHeader  map[string]string `json:"req_header,omitempty"`
	ReqCookie  map[string]string `json:"req_cookie,omitempty"`
	ReqTls     map[string]string `json:"req_tls,omitempty"`
	ReqBody    string            `json:"req_body,omitempty"`
	RespHeader map[string]string `json:"resp_header,omitempty"`
	RespCookie map[string]string `json:"resp_cookie,omitempty"`
	RespTls    map[string]string `json:"resp_tls,omitempty"`
	RespBody   chan []byte       `json:"resp_body,omitempty"`
}

func (m *SSEMessage) String() string {
	message, _ := jsoniter.MarshalToString(m)
	return message
}
