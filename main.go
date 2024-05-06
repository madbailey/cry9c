package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

var favicon = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00,
	0x10, 0x00, 0x00, 0x00, 0x0f, 0x04, 0x03, 0x00, 0x00, 0x00, 0x1f, 0x5d, 0x52, 0x1c, 0x00, 0x00, 0x00, 0x0f, 0x50,
	0x4c, 0x54, 0x45, 0x7a, 0xdf, 0xfd, 0xfd, 0xff, 0xfc, 0x39, 0x4d, 0x52, 0x19, 0x16, 0x15, 0xc3, 0x8d, 0x76, 0xc7,
	0x36, 0x2c, 0xf5, 0x00, 0x00, 0x00, 0x40, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x95, 0xc9, 0xd1, 0x0d, 0xc0, 0x20,
	0x0c, 0x03, 0xd1, 0x23, 0x5d, 0xa0, 0x49, 0x17, 0x20, 0x4c, 0xc0, 0x10, 0xec, 0x3f, 0x53, 0x8d, 0xc2, 0x02, 0x9c,
	0xfc, 0xf1, 0x24, 0xe3, 0x31, 0x54, 0x3a, 0xd1, 0x51, 0x96, 0x74, 0x1c, 0xcd, 0x18, 0xed, 0x9b, 0x9a, 0x11, 0x85,
	0x24, 0xea, 0xda, 0xe0, 0x99, 0x14, 0xd6, 0x3a, 0x68, 0x6f, 0x41, 0xdd, 0xe2, 0x07, 0xdb, 0xb5, 0x05, 0xca, 0xdb,
	0xb2, 0x9a, 0xdd, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

type Server struct {
	conns map[*websocket.Conn]bool
	lock  sync.Mutex
}

func NewServer() *Server {
	return &Server{
		conns: make(map[*websocket.Conn]bool),
	}
}

func (s *Server) handleWS(ws *websocket.Conn) {
	fmt.Println("New incoming connection from client:", ws.RemoteAddr())
	s.lock.Lock()
	s.conns[ws] = true
	s.lock.Unlock()
	defer s.closeConn(ws)
	s.readLoop(ws)
}

func (s *Server) readLoop(ws *websocket.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := ws.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Client disconnected:", ws.RemoteAddr())
			} else {
				fmt.Println("Read error:", err)
			}
			return
		}
		msg := buf[:n]
		s.broadcast(msg)
	}
}

func (s *Server) closeConn(ws *websocket.Conn) {
	s.lock.Lock()
	delete(s.conns, ws)
	s.lock.Unlock()
	ws.Close()
}

func (s *Server) broadcast(b []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for ws := range s.conns {
		go func(ws *websocket.Conn) {
			if _, err := ws.Write(b); err != nil {
				fmt.Println("Write error:", err)
			}
		}(ws)
	}
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	img, err := png.Decode(bytes.NewReader(favicon))
	if err != nil {
		http.Error(w, "Failed to decode image", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		return
	}

	base64Image := base64.StdEncoding.EncodeToString(buf.Bytes())
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(base64Image))
}

func main() {
	server := NewServer()
	http.Handle("/ws", websocket.Handler(server.handleWS))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/image", serveImage) // Serve the image via HTTP

	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
