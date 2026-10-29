package deck

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSSEDisconnectReleasesClientsAndGoroutines(t *testing.T) {
	baseline := runtime.NumGoroutine()
	a := testApp(&fakeHerdr{status: "idle", screen: "ready"})
	server := httptest.NewUnstartedServer(a)
	a.origin = "http://" + server.Listener.Addr().String()
	server.Start()
	transport := &http.Transport{DisableKeepAlives: true}
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	for range 12 {
		response, err := client.Get(server.URL + "/api/events?t=test-token")
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		reader := bufio.NewReader(response.Body)
		line, err := reader.ReadString('\n')
		response.Body.Close()
		if err != nil || !strings.HasPrefix(line, "event: hello") {
			server.Close()
			t.Fatalf("bad SSE: %q %v", line, err)
		}
	}
	transport.CloseIdleConnections()
	server.Close()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		clients := len(a.clients)
		a.mu.Unlock()
		if clients == 0 && runtime.NumGoroutine() <= baseline+2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	t.Fatalf("leak: clients=%d goroutines before=%d after=%d", len(a.clients), baseline, runtime.NumGoroutine())
}
func TestBackloggedSSEClientResyncsToLatestState(t *testing.T) {
	a := testApp(&fakeHerdr{})
	ch := make(chan []byte, 2)
	a.clients[ch] = true
	for i := range 5 {
		old := a.state
		next := old
		next.Panes = []Pane{{ID: "w1:p1", Key: "lab|w1:p1", Screen: strings.Repeat("x", i+1)}}
		a.publish(old, next)
	}
	found := false
	for len(ch) > 0 {
		message := string(<-ch)
		if strings.Contains(message, `"screen":"xxxxx"`) {
			found = true
		}
	}
	if !found || len(a.clients) != 1 {
		t.Fatal("latest snapshot lost or client disconnected")
	}
}
