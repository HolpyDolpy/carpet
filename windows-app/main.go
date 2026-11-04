package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed web/index.html
var assets embed.FS

type app struct {
	mu          sync.Mutex
	token       string
	folder      string
	link        workbookLink
	pendingOpen string
	server      *http.Server
}

func main() {
	base, err := os.UserConfigDir()
	if err != nil {
		showError("Carpets Ameer", err.Error())
		return
	}
	folder := filepath.Join(base, "CarpetsAmeer")
	if err = os.MkdirAll(folder, 0700); err != nil {
		showError("Carpets Ameer", err.Error())
		return
	}
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		showError("Carpets Ameer", err.Error())
		return
	}
	a := &app{token: hex.EncodeToString(b), folder: folder}
	if err = a.loadLink(); err != nil {
		showError("Carpets Ameer - settings error", err.Error())
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", a.home)
	mux.HandleFunc("GET /api/bootstrap", a.bootstrap)
	mux.HandleFunc("POST /api/snapshot", a.postSnapshot)
	mux.HandleFunc("POST /api/select-save", a.selectSave)
	mux.HandleFunc("POST /api/select-open", a.selectOpen)
	mux.HandleFunc("POST /api/confirm-open", a.confirmOpen)
	mux.HandleFunc("POST /api/save-workbook", a.saveWorkbook)
	mux.HandleFunc("POST /api/show-folder", a.showFolder)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	listenAddr := "127.0.0.1:0"
	if os.Getenv("CARPETS_TEST_MODE") == "1" {
		listenAddr = "127.0.0.1:18567"
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		showError("Carpets Ameer", err.Error())
		return
	}
	a.server = &http.Server{Handler: mux, ReadHeaderTimeout: 8 * time.Second, ReadTimeout: 90 * time.Second, WriteTimeout: 120 * time.Second}
	url := "http://" + listener.Addr().String() + "/"
	if os.Getenv("CARPETS_TEST_MODE") == "1" {
		fmt.Println("CARPETS_APP_URL=" + url)
	} else {
		go func() {
			time.Sleep(150 * time.Millisecond)
			if e := runDesktop(url, a.folder, func() { _ = a.server.Close() }); e != nil {
				showError("Carpets Ameer desktop window", e.Error())
			}
		}()
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() { <-stop; a.server.Close() }()
	if err = a.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		showError("Carpets Ameer", err.Error())
	}
}
func (a *app) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' data: blob:; connect-src 'self'; img-src 'self' data: blob:")
	html, err := assets.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	io.WriteString(w, strings.ReplaceAll(string(html), "__CARPETS_NATIVE_TOKEN__", a.token))
}
func (a *app) requirePost(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-Carpets-Token") != a.token {
		http.Error(w, "This request was not authorized", http.StatusForbidden)
		return false
	}
	if r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "Invalid request origin", http.StatusForbidden)
		return false
	}
	return true
}
func respond(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	jsonEncode(w, data)
}
func fail(w http.ResponseWriter, code int, e error) {
	respond(w, code, map[string]any{"error": e.Error()})
}
func (a *app) showFolder(w http.ResponseWriter, r *http.Request) {
	if !a.requirePost(w, r) {
		return
	}
	a.mu.Lock()
	path := a.link.Path
	a.mu.Unlock()
	if path == "" {
		path = a.folder
	} else {
		path = filepath.Join(filepath.Dir(path), "Carpets_Ameer_Backups")
		if err := os.MkdirAll(path, 0700); err != nil {
			fail(w, 500, err)
			return
		}
	}
	if err := openFolder(path); err != nil {
		fail(w, 500, err)
		return
	}
	respond(w, 200, map[string]any{"ok": true})
}
func readLimited(w http.ResponseWriter, r *http.Request, max int64) ([]byte, error) {
	return io.ReadAll(http.MaxBytesReader(w, r.Body, max))
}
func stringError(s string) error { return fmt.Errorf("%s", s) }
