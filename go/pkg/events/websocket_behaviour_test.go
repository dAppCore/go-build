package events

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	core "dappco.re/go"
	"github.com/gorilla/websocket"
)

// dialHub spins up a real httptest server fronting the hub handler and returns a
// connected gorilla client. It drives the live upgrade + read/write loops that
// the recorder-based tests cannot reach.
func dialHub(t *core.T) (*Hub, *websocket.Conn, func()) {
	t.Helper()
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	go hub.Run(ctx)

	server := httptest.NewServer(hub.Handler())
	wsURL := "ws" + core.TrimPrefix(server.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		cancel()
		server.Close()
		t.Fatalf("dial: %v", err)
	}
	if resp != nil {
		_ = resp.Body.Close()
	}

	teardown := func() {
		_ = conn.Close()
		cancel()
		server.Close()
	}
	return hub, conn, teardown
}

// waitFor polls until cond is true or the deadline elapses; the hub registers
// clients asynchronously so a brief settle window is required.
func waitFor(cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

func TestEvents_WebSocket_Connect_Good(t *core.T) {
	hub, _, teardown := dialHub(t)
	defer teardown()
	core.AssertTrue(t, waitFor(func() bool { return hub.ClientCount() == 1 }))
}

func TestEvents_WebSocket_SubscribeAndReceive_Good(t *core.T) {
	hub, conn, teardown := dialHub(t)
	defer teardown()
	core.AssertTrue(t, waitFor(func() bool { return hub.ClientCount() == 1 }))

	// Subscribe drives the readLoop's TypeSubscribe branch.
	core.AssertEqual(t, nil, conn.WriteJSON(Message{Type: TypeSubscribe, Data: "build"}))
	core.AssertTrue(t, waitFor(func() bool { return hub.ChannelSubscriberCount("build") == 1 }))

	// SendToChannel pushes through the client's writeLoop and lands on the wire.
	core.AssertTrue(t, hub.SendToChannel("build", Message{Type: TypeEvent, Data: "ping"}).OK)

	var got Message
	core.AssertEqual(t, nil, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
	core.AssertEqual(t, nil, conn.ReadJSON(&got))
	core.AssertEqual(t, "build", got.Channel)
	core.AssertEqual(t, "ping", got.Data.(string))
	core.AssertFalse(t, got.Timestamp.IsZero())
}

func TestEvents_WebSocket_Disconnect_RemovesClient_Ugly(t *core.T) {
	hub, conn, teardown := dialHub(t)
	defer teardown()
	core.AssertTrue(t, waitFor(func() bool { return hub.ClientCount() == 1 }))
	core.AssertEqual(t, nil, conn.WriteJSON(Message{Type: TypeSubscribe, Data: "build"}))
	core.AssertTrue(t, waitFor(func() bool { return hub.ChannelSubscriberCount("build") == 1 }))

	// Closing the client exits readLoop, which triggers removeClient and prunes
	// the now-empty channel.
	core.AssertEqual(t, nil, conn.Close())
	core.AssertTrue(t, waitFor(func() bool { return hub.ClientCount() == 0 }))
	core.AssertTrue(t, waitFor(func() bool { return hub.ChannelSubscriberCount("build") == 0 }))
}

func TestEvents_WebSocket_Upgrade_Bad(t *core.T) {
	// A plain HTTP GET (no Upgrade headers) against a live hub fails the gorilla
	// upgrade and never registers a client.
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	server := httptest.NewServer(hub.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	core.AssertEqual(t, nil, err)
	_ = resp.Body.Close()
	core.AssertFalse(t, resp.StatusCode == http.StatusSwitchingProtocols)
	core.AssertEqual(t, 0, hub.ClientCount())
}
