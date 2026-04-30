package events

import (
	"context"
	"net/http/httptest"
)

func ExampleNewHub() {
	hub := NewHub()
	_ = hub.ClientCount()
}

func ExampleHub_Run() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	hub := NewHub()
	hub.Run(ctx)
}

func ExampleHub_Handler() {
	hub := NewHub()
	_ = hub.Handler()
}

func ExampleHub_HandleWebSocket() {
	var hub *Hub
	hub.HandleWebSocket(httptest.NewRecorder(), httptest.NewRequest("GET", "/ws", nil))
}

func ExampleHub_Subscribe() {
	hub := NewHub()
	client := &Client{send: make(chan Message, 1)}
	_ = hub.Subscribe(client, "build")
}

func ExampleHub_SendToChannel() {
	hub := NewHub()
	client := &Client{send: make(chan Message, 1)}
	_ = hub.Subscribe(client, "build")
	_ = hub.SendToChannel("build", Message{Data: "ok"})
}

func ExampleHub_Broadcast() {
	hub := NewHub()
	hub.clients[&Client{send: make(chan Message, 1)}] = true
	_ = hub.Broadcast(Message{Data: "ok"})
}

func ExampleHub_ChannelSubscriberCount() {
	hub := NewHub()
	_ = hub.ChannelSubscriberCount("build")
}

func ExampleHub_ClientCount() {
	hub := NewHub()
	_ = hub.ClientCount()
}
