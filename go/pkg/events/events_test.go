package events

import (
	"context"
	"net/http"
	"net/http/httptest"

	core "dappco.re/go"
)

func TestEvents_NewHub_Good(t *core.T) {
	hub := NewHub()
	core.AssertEqual(t, 0, hub.ClientCount())
	core.AssertEqual(t, 0, hub.ChannelSubscriberCount("build"))
}

func TestEvents_NewHub_Bad(t *core.T) {
	hub := NewHub()
	result := hub.SendToChannel("missing", Message{})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, 0, hub.ClientCount())
}

func TestEvents_NewHub_Ugly(t *core.T) {
	first := NewHub()
	second := NewHub()
	core.AssertFalse(t, first == second)
	core.AssertEqual(t, 0, second.ClientCount())
}

func TestEvents_Hub_Run_Good(t *core.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	hub := NewHub()
	hub.Run(ctx)
	_, open := <-hub.done
	core.AssertFalse(t, open)
}

func TestEvents_Hub_Run_Bad(t *core.T) {
	var hub *Hub
	hub.Run(context.Background())
	core.AssertTrue(t, hub == nil)
}

func TestEvents_Hub_Run_Ugly(t *core.T) {
	ctx, cancel := context.WithCancel(context.Background())
	hub := NewHub()
	done := make(chan struct{})
	go func() {
		hub.Run(ctx)
		close(done)
	}()
	cancel()
	<-done
	core.AssertEqual(t, 0, hub.ClientCount())
}

func TestEvents_Hub_Handler_Good(t *core.T) {
	hub := NewHub()
	handler := hub.Handler()
	core.AssertFalse(t, handler == nil)
	core.AssertEqual(t, 0, hub.ClientCount())
}

func TestEvents_Hub_Handler_Bad(t *core.T) {
	var hub *Hub
	handler := hub.Handler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ws", nil))
	core.AssertEqual(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestEvents_Hub_Handler_Ugly(t *core.T) {
	hub := NewHub()
	handler := hub.Handler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ws", nil))
	core.AssertFalse(t, recorder.Code == http.StatusOK)
}

func TestEvents_Hub_HandleWebSocket_Good(t *core.T) {
	var hub *Hub
	recorder := httptest.NewRecorder()
	hub.HandleWebSocket(recorder, httptest.NewRequest(http.MethodGet, "/ws", nil))
	core.AssertEqual(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestEvents_Hub_HandleWebSocket_Bad(t *core.T) {
	hub := NewHub()
	recorder := httptest.NewRecorder()
	hub.HandleWebSocket(recorder, httptest.NewRequest(http.MethodGet, "/ws", nil))
	core.AssertFalse(t, recorder.Code == http.StatusOK)
}

func TestEvents_Hub_HandleWebSocket_Ugly(t *core.T) {
	hub := NewHub()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/ws", nil)
	hub.HandleWebSocket(recorder, request)
	core.AssertEqual(t, 0, hub.ClientCount())
}

func TestEvents_Hub_Subscribe_Good(t *core.T) {
	hub := NewHub()
	client := &Client{send: make(chan Message, 1)}
	result := hub.Subscribe(client, "build")
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, 1, hub.ChannelSubscriberCount("build"))
}

func TestEvents_Hub_Subscribe_Bad(t *core.T) {
	hub := NewHub()
	result := hub.Subscribe(nil, "build")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, 0, hub.ChannelSubscriberCount("build"))
}

func TestEvents_Hub_Subscribe_Ugly(t *core.T) {
	hub := NewHub()
	client := &Client{send: make(chan Message, 1)}
	result := hub.Subscribe(client, "")
	core.AssertFalse(t, result.OK)
}

func TestEvents_Hub_SendToChannel_Good(t *core.T) {
	hub := NewHub()
	client := &Client{send: make(chan Message, 1)}
	core.AssertTrue(t, hub.Subscribe(client, "build").OK)
	result := hub.SendToChannel("build", Message{Data: "ok"})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "build", (<-client.send).Channel)
}

func TestEvents_Hub_SendToChannel_Bad(t *core.T) {
	var hub *Hub
	result := hub.SendToChannel("build", Message{})
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "hub unavailable"))
}

func TestEvents_Hub_SendToChannel_Ugly(t *core.T) {
	hub := NewHub()
	result := hub.SendToChannel("", Message{Channel: "explicit"})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, 0, hub.ChannelSubscriberCount(""))
}

func TestEvents_Hub_Broadcast_Good(t *core.T) {
	hub := NewHub()
	client := &Client{send: make(chan Message, 1)}
	hub.clients[client] = true
	result := hub.Broadcast(Message{Data: "ok"})
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "ok", (<-client.send).Data.(string))
}

func TestEvents_Hub_Broadcast_Bad(t *core.T) {
	var hub *Hub
	result := hub.Broadcast(Message{})
	core.AssertFalse(t, result.OK)
	core.AssertTrue(t, core.Contains(result.Error(), "hub unavailable"))
}

func TestEvents_Hub_Broadcast_Ugly(t *core.T) {
	hub := NewHub()
	client := &Client{send: make(chan Message)}
	hub.clients[client] = true
	result := hub.Broadcast(Message{Data: "dropped"})
	core.AssertTrue(t, result.OK)
}

func TestEvents_Hub_ChannelSubscriberCount_Good(t *core.T) {
	hub := NewHub()
	client := &Client{send: make(chan Message, 1)}
	core.AssertTrue(t, hub.Subscribe(client, "build").OK)
	count := hub.ChannelSubscriberCount("build")
	core.AssertEqual(t, 1, count)
}

func TestEvents_Hub_ChannelSubscriberCount_Bad(t *core.T) {
	var hub *Hub
	count := hub.ChannelSubscriberCount("build")
	core.AssertEqual(t, 0, count)
}

func TestEvents_Hub_ChannelSubscriberCount_Ugly(t *core.T) {
	hub := NewHub()
	count := hub.ChannelSubscriberCount("")
	core.AssertEqual(t, 0, count)
}

func TestEvents_Hub_ClientCount_Good(t *core.T) {
	hub := NewHub()
	hub.clients[&Client{send: make(chan Message, 1)}] = true
	count := hub.ClientCount()
	core.AssertEqual(t, 1, count)
}

func TestEvents_Hub_ClientCount_Bad(t *core.T) {
	var hub *Hub
	count := hub.ClientCount()
	core.AssertEqual(t, 0, count)
}

func TestEvents_Hub_ClientCount_Ugly(t *core.T) {
	hub := NewHub()
	count := hub.ClientCount()
	core.AssertEqual(t, 0, count)
}
