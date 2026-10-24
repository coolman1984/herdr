package deck

import (
	"context"
	"os"
	"testing"
)

func TestLabDirectSocketReadOnly(t *testing.T) {
	path := os.Getenv("DECK_TEST_SOCKET")
	if path == "" {
		t.Skip("set DECK_TEST_SOCKET to the isolated lab socket")
	}
	s := &Socket{Path: path}
	defer s.Close()
	cli := &CLI{Socket: s}
	for range 3 {
		raw, err := cli.Call(context.Background(), "api", "snapshot")
		if err != nil {
			t.Fatal(err)
		}
		state, err := parseSnapshot(raw, path)
		if err != nil {
			t.Fatal(err)
		}
		if len(state.Panes) == 0 {
			t.Fatal("lab has no panes")
		}
		screen, err := cli.Call(context.Background(), "pane", "read", state.Panes[0].ID, "--source", "recent", "--lines", "40", "--format", "ansi")
		if err != nil {
			t.Fatal(err)
		}
		if len(screen) == 0 {
			t.Fatal("empty lab screen")
		}
	}
	if s.Connections != 6 {
		t.Fatalf("opened %d connections, wanted 6 protocol-22 one-response connections", s.Connections)
	}
	t.Log("6 direct read requests, protocol closes each RPC after one response, 0 CLI processes")
}
