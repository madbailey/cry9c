package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Adjust the origin checking to suit your needs
	},
}

// Client holds the WebSocket connection and the send channel
type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var clients = make(map[*Client]bool) // connected clients
var broadcast = make(chan []byte)    // broadcast channel

func main() {
	http.HandleFunc("/ws", handleConnections)
	go handleMessages()

	log.Println("HTTP server starting on port :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	client := &Client{conn: ws, send: make(chan []byte)}
	clients[client] = true

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, client)
			break
		}
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			select {
			case client.send <- msg:
				// send the message
				err := client.conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					log.Printf("error: %v", err)
					client.conn.Close()
					delete(clients, client)
				}
			default:
				// failed to send
				close(client.send)
				delete(clients, client)
			}
		}
	}
}
