package deck

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"
)

// Herdr sizes a tab's panes from an attached client's surface size; with no client attached
// every tab stays 120x40 (LESSONS #95). Sizer makes Deck a Herdr "client shell" for sizing only:
// it sends the endpoint hello with the Deck card area in cells, then ClientShellResize on every
// change, and reads and discards everything the server streams back. Wire format (herdr-app
// src/protocol/wire.rs + endpoint.rs, Herdr 0.9.5): u32 little-endian length + bincode
// "standard" payload (varint integers, varint enum tags, varint-length strings).
const (
	clientShellResizeTag     = 12 // ClientMessage::ClientShellResize
	clientEndpointControlTag = 20 // ClientMessage::EndpointControl (stable, append-only)
	serverEndpointControlTag = 20 // ServerMessage::EndpointControl
	maxServerFrame           = 32 << 20
	sizerRetry               = 5 * time.Second
)

type Sizer struct {
	Path string                                        // the client socket marker path (herdr-client.sock)
	Dial func(path string) (io.ReadWriteCloser, error) // tests replace it

	mu        sync.Mutex
	conn      io.ReadWriteCloser
	want      [2]uint16 // cols, rows asked by the page
	sent      [2]uint16 // last size Herdr accepted on this connection
	failedAt  time.Time
	lastError string
	writeMu   sync.Mutex
}

func clientSocketPath(apiSocket string) string {
	return filepath.Join(filepath.Dir(apiSocket), "herdr-client.sock")
}

// SizerStatus is what /api/size answers: the size asked for, the size Herdr has, and why not.
type SizerStatus struct {
	Cols     uint16 `json:"cols"`
	Rows     uint16 `json:"rows"`
	Attached bool   `json:"attached"`
	Error    string `json:"error,omitempty"`
}

// Set asks Herdr for a cols x rows pane area. It connects on first use, reconnects after a
// failure at most every 5 s, and sends nothing when the size has not changed.
func (z *Sizer) Set(cols, rows uint16) SizerStatus {
	z.mu.Lock()
	defer z.mu.Unlock()
	z.want = [2]uint16{cols, rows}
	if z.conn == nil {
		if time.Since(z.failedAt) < sizerRetry {
			return z.statusLocked()
		}
		if err := z.connectLocked(cols, rows); err != nil {
			z.failedAt, z.lastError = time.Now(), err.Error()
			return z.statusLocked()
		}
		return z.statusLocked()
	}
	if z.sent == z.want {
		return z.statusLocked()
	}
	if err := z.write(encodeShellResize(cols, rows)); err != nil {
		z.dropLocked(fmt.Errorf("Herdr client resize: %w", err))
		return z.statusLocked()
	}
	z.sent = z.want
	return z.statusLocked()
}

func (z *Sizer) statusLocked() SizerStatus {
	s := SizerStatus{Cols: z.sent[0], Rows: z.sent[1], Attached: z.conn != nil}
	if z.conn == nil {
		s.Cols, s.Rows, s.Error = 0, 0, z.lastError
	}
	return s
}

func (z *Sizer) connectLocked(cols, rows uint16) error {
	conn, err := z.Dial(z.Path)
	if err != nil {
		return fmt.Errorf("Herdr client socket %s: %w", z.Path, err)
	}
	hello, err := encodeEndpointHello(cols, rows)
	if err != nil {
		conn.Close()
		return err
	}
	if _, err = conn.Write(hello); err != nil {
		conn.Close()
		return fmt.Errorf("Herdr client hello: %w", err)
	}
	reader := bufio.NewReaderSize(conn, 64<<10)
	timed, _ := conn.(interface{ SetReadDeadline(time.Time) error })
	if timed != nil {
		_ = timed.SetReadDeadline(time.Now().Add(5 * time.Second))
	}
	kind, data, err := readServerControl(reader)
	if timed != nil && err == nil {
		err = timed.SetReadDeadline(time.Time{})
	}
	if err != nil {
		conn.Close()
		return fmt.Errorf("Herdr client welcome: %w", err)
	}
	if err = checkWelcome(kind, data); err != nil {
		conn.Close()
		return err
	}
	z.conn, z.sent, z.lastError = conn, [2]uint16{cols, rows}, ""
	go z.drain(conn, reader)
	return nil
}

// drain reads and discards server frames (snapshots, surfaces) so Herdr never blocks on Deck.
func (z *Sizer) drain(conn io.ReadWriteCloser, r *bufio.Reader) {
	var err error
	for err == nil {
		var n uint32
		if err = binary.Read(r, binary.LittleEndian, &n); err == nil {
			if n > maxServerFrame {
				err = fmt.Errorf("server frame of %d bytes exceeds the limit", n)
			} else {
				_, err = r.Discard(int(n))
			}
		}
	}
	z.mu.Lock()
	defer z.mu.Unlock()
	if z.conn == conn {
		z.dropLocked(fmt.Errorf("Herdr client connection closed: %w", err))
	}
}

func (z *Sizer) dropLocked(err error) {
	if z.conn != nil {
		z.conn.Close()
	}
	z.conn, z.sent, z.failedAt, z.lastError = nil, [2]uint16{}, time.Now(), err.Error()
}

// Close detaches from Herdr (the tabs keep their last size).
func (z *Sizer) Close() {
	z.mu.Lock()
	defer z.mu.Unlock()
	if z.conn != nil {
		_ = z.write(frame([]byte{4})) // ClientMessage::Detach
		z.conn.Close()
		z.conn = nil
	}
}

func (z *Sizer) write(b []byte) error {
	z.writeMu.Lock()
	defer z.writeMu.Unlock()
	_, err := z.conn.Write(b)
	return err
}

func checkWelcome(kind, data string) error {
	if kind != "endpoint.welcome.v1" {
		return fmt.Errorf("Herdr client welcome: unexpected %q", kind)
	}
	var w struct {
		Generation uint32 `json:"generation"`
		Error      *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(data), &w); err != nil {
		return fmt.Errorf("Herdr client welcome: %w", err)
	}
	if w.Error != nil {
		return fmt.Errorf("Herdr refused the Deck client: %s", w.Error.Message)
	}
	if w.Generation != 1 {
		return fmt.Errorf("Herdr client welcome: endpoint generation %d, Deck speaks 1", w.Generation)
	}
	return nil
}

// ---- bincode "standard" encoding (only what Deck sends) ----

func varint(b []byte, v uint64) []byte {
	switch {
	case v < 251:
		return append(b, byte(v))
	case v <= 0xFFFF:
		return binary.LittleEndian.AppendUint16(append(b, 251), uint16(v))
	case v <= 0xFFFFFFFF:
		return binary.LittleEndian.AppendUint32(append(b, 252), uint32(v))
	}
	return binary.LittleEndian.AppendUint64(append(b, 253), v)
}
func str(b []byte, s string) []byte { return append(varint(b, uint64(len(s))), s...) }
func frame(payload []byte) []byte {
	return append(binary.LittleEndian.AppendUint32(nil, uint32(len(payload))), payload...)
}

func encodeEndpointHello(cols, rows uint16) ([]byte, error) {
	hello, err := json.Marshal(map[string]any{
		"generation": 1, "cell_width_px": 0, "cell_height_px": 0,
		"surface_size": map[string]uint16{"cols": cols, "rows": rows},
		"pixel_mouse":  false, "direct_graphics": false, "endpoint_keybindings": false, "mouse_capture": false,
		"surface_active": true, "surface_reuse": true, "surface_delta": true, "surface_scroll": true,
		"snapshot_codecs": []string{"shell.snapshot.v1"}, "surface_codecs": []string{"shell.surface.v1"},
		"input_codecs": []string{"shell.input.semantic.v1"}, "blob_codecs": []string{"shell.blob.v1"},
	})
	if err != nil {
		return nil, err
	}
	b := varint(nil, clientEndpointControlTag)
	b = str(b, "endpoint.hello.v1")
	return frame(str(b, string(hello))), nil
}

func encodeShellResize(cols, rows uint16) []byte {
	b := varint(nil, clientShellResizeTag)
	b = varint(b, 0) // cell_width_px (unknown)
	b = varint(b, 0) // cell_height_px
	b = varint(b, uint64(cols))
	b = varint(b, uint64(rows))
	return frame(append(b, 0)) // pixel_mouse false
}

// readServerControl reads one frame and decodes it as ServerMessage::EndpointControl.
func readServerControl(r *bufio.Reader) (kind, data string, err error) {
	var n uint32
	if err = binary.Read(r, binary.LittleEndian, &n); err != nil {
		return
	}
	if n > 2<<20 {
		return "", "", fmt.Errorf("first frame of %d bytes is too large", n)
	}
	payload := make([]byte, n)
	if _, err = io.ReadFull(r, payload); err != nil {
		return
	}
	p := payload
	tag, p, err := readVarint(p)
	if err != nil {
		return
	}
	if tag != serverEndpointControlTag {
		return "", "", fmt.Errorf("server message %d, expected endpoint control", tag)
	}
	if kind, p, err = readStr(p); err != nil {
		return
	}
	data, _, err = readStr(p)
	return
}
func readVarint(p []byte) (uint64, []byte, error) {
	if len(p) == 0 {
		return 0, p, io.ErrUnexpectedEOF
	}
	need := map[byte]int{251: 2, 252: 4, 253: 8}[p[0]]
	if p[0] < 251 {
		return uint64(p[0]), p[1:], nil
	}
	if need == 0 || len(p) < 1+need {
		return 0, p, fmt.Errorf("bad varint")
	}
	var v uint64
	for i := need; i >= 1; i-- {
		v = v<<8 | uint64(p[i])
	}
	return v, p[1+need:], nil
}
func readStr(p []byte) (string, []byte, error) {
	n, p, err := readVarint(p)
	if err != nil {
		return "", p, err
	}
	if uint64(len(p)) < n {
		return "", p, io.ErrUnexpectedEOF
	}
	return string(p[:n]), p[n:], nil
}
