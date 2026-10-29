package deck

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

type instanceRecord struct {
	URL string `json:"url"`
}

func Main(assets fs.FS) {
	if err := run(assets); err != nil {
		if item := flag.Lookup("headless"); item != nil && item.Value.String() == "true" {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		messageBox(err.Error())
		os.Exit(1)
	}
}
func run(assets fs.FS) error {
	debug.SetGCPercent(50)
	debug.SetMemoryLimit(24 << 20)
	session := flag.String("session", "", "Herdr session (otherwise inherited socket)")
	headless := flag.Bool("headless", false, "HTTP server only for isolated tests")
	address := flag.String("address-file", "", "test-only address file")
	debugMode := flag.Bool("debug", false, "enable token-protected pprof in isolated tests")
	flag.Parse()
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	root := filepath.Dir(filepath.Dir(executable))
	// go run and test binaries can use an explicit workspace root.
	if v := os.Getenv("HERDR_DECK_ROOT"); v != "" {
		root = v
	}
	identity := os.Getenv("HERDR_SOCKET_PATH")
	if *session != "" {
		identity = filepath.Join(os.Getenv("APPDATA"), "herdr", "sessions", *session, "herdr.sock")
		if err = os.Setenv("HERDR_SOCKET_PATH", identity); err != nil {
			return err
		}
		for _, key := range []string{"HERDR_PANE_ID", "HERDR_TAB_ID", "HERDR_WORKSPACE_ID"} {
			if err = os.Unsetenv(key); err != nil {
				return err
			}
		}
	}
	if identity == "" {
		identity = filepath.Join(os.Getenv("APPDATA"), "herdr", "herdr.sock")
	}
	identity = strings.ToLower(filepath.Clean(identity))
	sum := sha256.Sum256([]byte(identity))
	name := hex.EncodeToString(sum[:8])
	mutex, existing, err := instanceMutex(name)
	if err != nil {
		return err
	}
	defer closeMutex(mutex)
	recordPath := filepath.Join(os.Getenv("APPDATA"), "herdr", "deck-instance-"+name+".json")
	if existing {
		var lastErr error
		for range 20 {
			data, e := os.ReadFile(recordPath)
			lastErr = e
			if e == nil {
				var rec instanceRecord
				if e = json.Unmarshal(data, &rec); e == nil && strings.HasPrefix(rec.URL, "http://127.0.0.1:") {
					client := http.Client{Timeout: time.Second}
					req, e := http.NewRequest("POST", strings.Replace(rec.URL, "/?t=", "/api/reopen?t=", 1), strings.NewReader(`{}`))
					if e != nil {
						return e
					}
					req.Header.Set("Content-Type", "application/json")
					res, e := client.Do(req)
					lastErr = e
					if e == nil {
						res.Body.Close()
						if res.StatusCode == 200 {
							return nil
						}
						lastErr = fmt.Errorf("reopen returned %d", res.StatusCode)
					}
				} else {
					lastErr = fmt.Errorf("invalid instance record")
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
		return fmt.Errorf("cannot reach the existing Deck window: %v", lastErr)
	}
	tokenBytes := make([]byte, 32)
	if _, err = rand.Read(tokenBytes); err != nil {
		return err
	}
	token := hex.EncodeToString(tokenBytes)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	origin := "http://" + listener.Addr().String()
	url := origin + "/?t=" + token
	profiles, err := readProfiles(root)
	if err != nil {
		return fmt.Errorf("agent profiles: %w", err)
	}
	bin := os.Getenv("HERDR_BIN")
	if bin == "" {
		bin = filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Herdr", "bin", "herdr.exe")
	}
	bridge := &CLI{Bin: bin, Session: *session, Root: root}
	if os.Getenv("HERDR_BIN") == "" {
		bridge.Socket = &Socket{Path: identity}
		defer bridge.Socket.Close()
	}
	app := newApp(bridge, identity, token, origin, root, profiles)
	app.sessionName, app.herdrBin = *session, bin
	if bridge.Socket != nil {
		app.sizer = &Sizer{Path: clientSocketPath(identity), Dial: dialPipe}
		defer app.sizer.Close()
	}
	app.debug = *debugMode
	defer app.closeJobs()
	var edge string
	if !*headless {
		edge, err = findEdge()
		if err != nil {
			return err
		}
	}
	app.reopen = func() error {
		if *headless {
			return nil
		}
		return openWindow(edge, url, filepath.Join(os.Getenv("APPDATA"), "herdr", "deck-profile"))
	}
	files, err := fs.Sub(assets, "web")
	if err != nil {
		return err
	}
	app.files = staticHandler(files)
	if err = os.MkdirAll(filepath.Dir(recordPath), 0700); err != nil {
		return err
	}
	rec, _ := json.Marshal(instanceRecord{URL: url})
	if err = os.WriteFile(recordPath, rec, 0600); err != nil {
		return err
	}
	defer os.Remove(recordPath)
	if *address != "" {
		if !*headless {
			return fmt.Errorf("address-file requires headless mode")
		}
		if err = os.WriteFile(*address, rec, 0600); err != nil {
			return err
		}
		defer os.Remove(*address)
	}
	server := &http.Server{Handler: app, ReadHeaderTimeout: 3 * time.Second, IdleTimeout: 10 * time.Second, MaxHeaderBytes: 8192}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go app.poll(ctx)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	if err = app.reopen(); err != nil {
		return err
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err = <-done:
			if err == http.ErrServerClosed {
				return nil
			}
			return err
		case <-ticker.C:
			if app.shouldExit(time.Now()) {
				shutdown, c := context.WithTimeout(context.Background(), time.Second)
				defer c()
				return server.Shutdown(shutdown)
			}
		}
	}
}

var textTypes = map[string]string{".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8", ".css": "text/css; charset=utf-8"}

type compressedFile struct {
	plain, gz []byte
	tag       string
}

// staticHandler serves the embedded page and caches assets and fonts immutably. HTML, JS and CSS
// are gzip-compressed once and revalidated by ETag, so a new exe is picked up at the next load.
func staticHandler(files fs.FS) http.Handler {
	static := http.FileServer(http.FS(files))
	var mu sync.Mutex
	cache := map[string]*compressedFile{}
	load := func(name string) (*compressedFile, error) {
		mu.Lock()
		defer mu.Unlock()
		if f := cache[name]; f != nil {
			return f, nil
		}
		data, err := fs.ReadFile(files, name)
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		zw.Write(data)
		if err = zw.Close(); err != nil {
			return nil, err
		}
		hash := sha256.Sum256(data)
		f := &compressedFile{plain: data, gz: buf.Bytes(), tag: `"` + hex.EncodeToString(hash[:16]) + `"`}
		cache[name] = f
		return f, nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		immutable := strings.HasPrefix(name, "assets/") || strings.HasPrefix(name, "fonts/")
		if kind, ok := textTypes[filepath.Ext(name)]; ok && !immutable && !strings.Contains(name, "..") {
			f, err := load(name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			h := w.Header()
			h.Set("Content-Type", kind)
			h.Set("Cache-Control", "no-cache")
			h.Set("ETag", f.tag)
			h.Set("Vary", "Accept-Encoding")
			if r.Header.Get("If-None-Match") == f.tag {
				w.WriteHeader(304)
				return
			}
			body := f.plain
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				h.Set("Content-Encoding", "gzip")
				body = f.gz
			}
			h.Set("Content-Length", strconv.Itoa(len(body)))
			if r.Method != "HEAD" {
				w.Write(body)
			}
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/fonts/") {
			data, err := fs.ReadFile(files, strings.TrimPrefix(r.URL.Path, "/"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			hash := sha256.Sum256(data)
			tag := `"` + hex.EncodeToString(hash[:]) + `"`
			w.Header().Set("ETag", tag)
			w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
			if strings.HasSuffix(r.URL.Path, ".woff2") {
				w.Header().Set("Content-Type", "font/woff2")
			}
			if r.Header.Get("If-None-Match") == tag {
				w.WriteHeader(304)
				return
			}
		}
		static.ServeHTTP(w, r)
	})
}
