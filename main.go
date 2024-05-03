package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

type Server struct {
	conns map[*websocket.Conn]bool
}

func NewServer() *Server {
	return &Server{
		conns: make(map[*websocket.Conn]bool),
	}
}

func (s *Server) handleWS(ws *websocket.Conn) {
	fmt.Println("New connection:", ws.RemoteAddr())
	s.conns[ws] = true
	s.readLoop(ws)
}

func (s *Server) readLoop(ws *websocket.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := ws.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed:", ws.RemoteAddr())
				break
			}
			fmt.Println("Read error:", err)
			continue
		}

		msg := buf[:n]
		fmt.Printf("Received message from %v: %s\n", ws.RemoteAddr(), msg)
		responseMsg := "Thank you for your message!"
		ws.Write([]byte(responseMsg))
		fmt.Printf("Sent response to %v: %s\n", ws.RemoteAddr(), responseMsg)
	}

	// Remove the connection from the map
	delete(s.conns, ws)
	fmt.Println("Removed connection:", ws.RemoteAddr())
}

func main() {
	server := NewServer()
	http.Handle("/ws", websocket.Handler(server.handleWS))
	fmt.Println("Server starting on http://localhost:8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
	}
}
