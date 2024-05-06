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

	"golang.org/x/image/draw"
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

func resizeImage(img image.Image, width, height int) image.Image {
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(newImg, newImg.Bounds(), img, img.Bounds(), draw.Over, nil)
	return newImg
}

func sortPixels(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	sortedImage := image.NewRGBA(bounds)
	width, height := bounds.Dx(), bounds.Dy()
	rowChannel := make(chan int, height) // Channel to signal when a row is ready

	for y := 0; y < height; y++ {
		go func(y int) {
			row := make([]color.Color, width)
			for x := 0; x < width; x++ {
				row[x] = img.At(bounds.Min.X+x, bounds.Min.Y+y)
			}

			sort.Slice(row, func(i, j int) bool {
				r, g, b, _ := row[i].RGBA()
				r2, g2, b2, _ := row[j].RGBA()
				return r+g+b < r2+g2+b2
			})

			for x, clr := range row {
				sortedImage.Set(bounds.Min.X+x, bounds.Min.Y+y, clr)
			}
			rowChannel <- y // Signal completion of this row
		}(y)
	}

	// Wait for all goroutines to complete
	for i := 0; i < height; i++ {
		<-rowChannel
	}

	return sortedImage
}

func serveImage(w http.ResponseWriter, r *http.Request) {
	filePath := "dog.png"
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		log.Printf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		log.Printf("Failed to read file: %v", err)
		return
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		http.Error(w, "Failed to decode image", http.StatusInternalServerError)
		log.Printf("Failed to decode image: %v", err)
		return
	}
	resizedImg := resizeImage(img, 500, 500) // Resize the image to 500x500

	// Apply pixel sorting here
	sortedImg := sortPixels(resizedImg)

	var buf bytes.Buffer
	if err := png.Encode(&buf, sortedImg); err != nil {
		http.Error(w, "Failed to encode image", http.StatusInternalServerError)
		log.Printf("Failed to encode image: %v", err)
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
