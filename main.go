package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
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

// sortPixels sorts the pixels of an image row by row based on the brightness.
func sortPixels(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	sortedImage := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		var row []color.Color
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			row = append(row, img.At(x, y))
		}
		sort.Slice(row, func(i, j int) bool {
			r, g, b, _ := row[i].RGBA()
			r2, g2, b2, _ := row[j].RGBA()
			brightness1 := r + g + b
			brightness2 := r2 + g2 + b2
			return brightness1 < brightness2
		})
		for x, clr := range row {
			sortedImage.Set(bounds.Min.X+x, y, clr)
		}
	}

	return sortedImage
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	filePath := "dog.png"
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err) // Consider using http.Error for production
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read", http.StatusInternalServerError) // Consider using http.Error for production
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		http.Error(w, "Failed to decode image", http.StatusInternalServerError)
		return
	}

	// Apply pixel sorting here
	sortedImg := sortPixels(img)

	var buf bytes.Buffer
	if err := png.Encode(&buf, sortedImg); err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(buf.Bytes())
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
