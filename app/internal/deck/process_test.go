package deck

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("DECK_TEST_STUB") == "1" {
		if len(os.Args) > 1 && os.Args[1] == "slow" {
			time.Sleep(10 * time.Second)
			return
		}
		console, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
		json.NewEncoder(os.Stdout).Encode(map[string]any{"args": os.Args[1:], "console": console != 0, "socket": os.Getenv("HERDR_SOCKET_PATH")})
		return
	}
	os.Exit(m.Run())
}
func TestExecutableStubPreservesArgumentBoundariesAndHasNoConsole(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DECK_TEST_STUB", "1")
	t.Setenv("HERDR_BIN", executable)
	t.Setenv("HERDR_SOCKET_PATH", `I:\lab\herdr.sock`)
	cli := &CLI{Bin: os.Getenv("HERDR_BIN"), Session: "lab"}
	prompt := "quotes \"yes\"\n$(never); C:\\folder with spaces\\"
	out, err := cli.Call(context.Background(), "agent", "prompt", "coder", prompt)
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Args    []string `json:"args"`
		Console bool     `json:"console"`
		Socket  string   `json:"socket"`
	}
	if err = json.Unmarshal(out, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.Args) != 6 || v.Args[0] != "--session" || v.Args[5] != prompt || v.Console || v.Socket != `I:\lab\herdr.sock` {
		t.Fatalf("unsafe child behavior: %+v", v)
	}
}
func TestEveryCLIRequestHasACancelDeadline(t *testing.T) {
	executable, _ := os.Executable()
	t.Setenv("DECK_TEST_STUB", "1")
	cli := &CLI{Bin: executable}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := cli.Call(ctx, "slow")
	if err == nil || !strings.Contains(err.Error(), "timed out") || time.Since(started) > 3*time.Second {
		t.Fatalf("deadline not honored: %v", err)
	}
}
func TestNamedMutexDetectsSecondInstance(t *testing.T) {
	name := fmt.Sprintf("test-%d", time.Now().UnixNano())
	first, exists, err := instanceMutex(name)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMutex(first)
	if exists {
		t.Fatal("fresh mutex exists")
	}
	second, exists, err := instanceMutex(name)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMutex(second)
	if !exists {
		t.Fatal("second launch missed mutex")
	}
}
