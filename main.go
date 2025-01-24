package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Message struct {
	ID      int
	Content string
}

type Client struct {
	Channel chan string
	Timeout time.Time
}

type Topic struct {
	Messages []Message
	Clients  map[*Client]bool
	mu       sync.Mutex
}

type Server struct {
	Topics map[string]*Topic
	mu     sync.Mutex
	MsgID  int
}

func NewServer() *Server {
	return &Server{
		Topics: make(map[string]*Topic),
	}
}

func (s *Server) nextMessageID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MsgID++
	return s.MsgID
}

func (s *Server) getOrCreateTopic(name string) *Topic {
	s.mu.Lock()
	defer s.mu.Unlock()
	if topic, exists := s.Topics[name]; exists {
		return topic
	}
	topic := &Topic{
		Clients: make(map[*Client]bool),
	}
	s.Topics[name] = topic
	return topic
}

func (s *Server) publishMessage(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimPrefix(r.URL.Path, "/infocenter/")
	if topicName == "" {
		http.Error(w, "Topic not specified", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Failed to read message", http.StatusInternalServerError)
		return
	}
	messageContent := string(body)
	messageID := s.nextMessageID()

	topic := s.getOrCreateTopic(topicName)
	topic.mu.Lock()
	topic.Messages = append(topic.Messages, Message{ID: messageID, Content: messageContent})
	for client := range topic.Clients {
		select {
		case client.Channel <- fmt.Sprintf("id: %d\nevent: msg\ndata: %s\n\n", messageID, messageContent):
		default:
		}
	}
	topic.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) subscribeToTopic(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimPrefix(r.URL.Path, "/infocenter/")
	if topicName == "" {
		http.Error(w, "Topic not specified", http.StatusBadRequest)
		return
	}
	topic := s.getOrCreateTopic(topicName)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	client := &Client{
		Channel: make(chan string, 10),
		Timeout: time.Now().Add(30 * time.Second),
	}
	topic.mu.Lock()
	topic.Clients[client] = true
	topic.mu.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	disconnected := make(chan struct{})
	go func() {
		<-r.Context().Done()
		close(disconnected)
	}()

	timeConnected := time.Now()
	for {
		select {
		case msg := <-client.Channel:
			_, err := fmt.Fprint(w, msg)
			if err != nil {
				break
			}
			flusher.Flush()
		case <-time.After(30 * time.Second):
			_, _ = fmt.Fprintf(w, "event: timeout\ndata: %ds\n\n", int(time.Since(timeConnected).Seconds()))
			flusher.Flush()
			topic.mu.Lock()
			delete(topic.Clients, client)
			topic.mu.Unlock()
			return
		case <-disconnected:
			topic.mu.Lock()
			delete(topic.Clients, client)
			topic.mu.Unlock()
			return
		}
	}
}

func main() {
	server := NewServer()

	http.HandleFunc("/infocenter/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			server.publishMessage(w, r)
		} else if r.Method == http.MethodGet {
			server.subscribeToTopic(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
