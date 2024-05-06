package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"golang.org/x/net/websocket"
)

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
	filePath := "epd_butmap_.bin"
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	img, err := png.Decode(bytes.NewReader(data))
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
