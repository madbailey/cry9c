package main

import (
	"fmt"
	"io"
	"net/http"
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

func main() {
	server := NewServer()
	http.Handle("/ws", websocket.Handler(server.handleWS))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./site/index.html")
	})
	http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./site/about.html")
	})
	http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir("./styles"))))
	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir("./resources"))))

	fmt.Println("Server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
