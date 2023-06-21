package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"text/template"
	"time"
)

type Message struct {
	ID        int       `json:"id"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
}

var (
	messages []Message
	mutex    sync.Mutex
)

type TemplateData struct {
	Messages     []Message
	ServerStatus string
}

func main() {
	http.HandleFunc("/messages", handleMessages)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/", handleHome)
	log.Fatal(http.ListenAndServe(":8000", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html"))

	data := TemplateData{
		Messages:     messages,
		ServerStatus: "Running",
	}

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
		}

		message := Message{
			ID:        len(messages) + 1,
			Sender:    r.Form.Get("sender"),
			Recipient: r.Form.Get("recipient"),
			Timestamp: time.Now(),
			Content:   r.Form.Get("content"),
		}

		mutex.Lock()
		messages = append(messages, message)
		mutex.Unlock()
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		log.Println(err)
	}
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getMessages(w, r)
	case http.MethodPost:
		createMessage(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func createMessage(w http.ResponseWriter, r *http.Request) {
	var message Message
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mutex.Lock()
	message.ID = len(messages) + 1
	message.Timestamp = time.Now()
	messages = append(messages, message)
	mutex.Unlock()

	w.WriteHeader(http.StatusCreated)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "Running"})
}
