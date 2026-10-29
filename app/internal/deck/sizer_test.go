package deck

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeHerdrClient answers one Deck sizing client like Herdr 0.9.5: it decodes the endpoint hello, sends
// the welcome (or a rejection) and records every later frame.
func fakeHerdrClient(t *testing.T, welcome string) (*Sizer, chan []byte, chan map[string]any) {
	t.Helper()
	frames, hellos := make(chan []byte, 8), make(chan map[string]any, 1)
	z := &Sizer{Path: "lab", Dial: func(string) (io.ReadWriteCloser, error) {
		deck, herdr := net.Pipe()
		go func() {
			r := bufio.NewReader(herdr)
			payload := readFrame(t, r)
			tag, p, _ := readVarint(payload)
			kind, p, _ := readStr(p)
			data, _, _ := readStr(p)
			if tag != 20 || kind != "endpoint.hello.v1" {
				t.Errorf("hello tag %d kind %q", tag, kind)
			}
			var hello map[string]any
			if err := json.Unmarshal([]byte(data), &hello); err != nil {
				t.Errorf("hello JSON: %v", err)
			}
			hellos <- hello
			b := str(str(varint(nil, 20), "endpoint.welcome.v1"), welcome)
			herdr.Write(frame(b))
			herdr.Write(frame([]byte{12, 1, 2, 3})) // a snapshot Deck must skip
			for {
				f := readFrame(t, r)
				if f == nil {
					return
				}
				frames <- f
			}
		}()
		return deck, nil
	}}
	return z, frames, hellos
}
func readFrame(t *testing.T, r *bufio.Reader) []byte {
	var n uint32
	if binary.Read(r, binary.LittleEndian, &n) != nil {
		return nil
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil
	}
	return b
}

func TestSizerHelloCarriesTheCardAreaThenResizesOnlyOnChange(t *testing.T) {
	z, frames, hellos := fakeHerdrClient(t, `{"generation":1,"server_version":"0.9.5"}`)
	if s := z.Set(200, 50); !s.Attached || s.Cols != 200 || s.Rows != 50 || s.Error != "" {
		t.Fatalf("first Set = %+v", s)
	}
	hello := <-hellos
	size := hello["surface_size"].(map[string]any)
	if size["cols"] != 200.0 || size["rows"] != 50.0 || hello["surface_active"] != true || hello["generation"] != 1.0 {
		t.Fatalf("hello = %v", hello)
	}
	z.Set(200, 50) // same size: nothing on the wire
	z.Set(300, 37)
	select {
	case f := <-frames:
		// ClientShellResize: tag 12, cell 0x0, cols 300 (varint 251 + u16 LE), rows 37, pixel_mouse false
		if want := []byte{12, 0, 0, 251, 0x2c, 0x01, 37, 0}; string(f) != string(want) {
			t.Fatalf("resize frame % x, want % x", f, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no resize frame")
	}
	select {
	case f := <-frames:
		t.Fatalf("unexpected extra frame % x", f)
	case <-time.After(100 * time.Millisecond):
	}
	z.Close()
}

func TestSizerReportsHerdrsRefusalAndWaitsBeforeRetrying(t *testing.T) {
	z, _, _ := fakeHerdrClient(t, `{"generation":1,"error":{"code":"invalid_surface","message":"too big"}}`)
	s := z.Set(200, 50)
	if s.Attached || !strings.Contains(s.Error, "too big") {
		t.Fatalf("Set = %+v", s)
	}
	dials := 0
	z.Dial = func(string) (io.ReadWriteCloser, error) { dials++; return nil, io.EOF }
	if z.Set(201, 50); dials != 0 {
		t.Fatalf("redialled %d times inside the retry window", dials)
	}
}
