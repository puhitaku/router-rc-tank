package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi"
	"go.bug.st/serial"
)

// Thread-safe serial port wrapper

type SerialPort struct {
	port serial.Port
	lock sync.Mutex
}

func NewSerialPort(port string, baud int) (*SerialPort, error) {
	p, err := serial.Open(port, &serial.Mode{BaudRate: baud})
	if err != nil {
		return nil, fmt.Errorf("failed to open the port: %s", err)
	}

	return &SerialPort{port: p}, nil
}

func (s *SerialPort) Write(p []byte) (n int, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.port.Write(p)
}

func (s *SerialPort) Close() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.port.Close()
}

// Models

type Hello struct {
	Message string `json:"message"`
}

type Request struct {
	Operation string `json:"operation,omitempty"`
}

type Response struct {
	Operation string `json:"operation,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Shorthands

func replyAndLogError(w http.ResponseWriter, status int, operation, error string, a ...any) {
	r := Response{
		Operation: operation,
		Error:     fmt.Sprintf(error, a...),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	res, err := json.Marshal(r)
	if err != nil {
		log.Printf("failed to marshal the response: %s", err)
		return
	}

	n, err := w.Write(res)
	if err != nil {
		log.Printf("failed to write the response: %s (written = %d B)", err, n)
		return
	}

	log.Printf(r.Error)
}

func main() {
	s, err := NewSerialPort("/dev/ttyACM0", 115200)
	if err != nil {
		panic(err)
	}
	defer s.Close()

	if err := Main(s); err != nil {
		panic(err)
	}
}

func Main(wc io.WriteCloser) error {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "serial", wc)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Get("/healthz", getHealthzHandler)
	r.Put("/operation", putOperationHandler)

	return http.ListenAndServe(":8080", r)
}

func getHealthzHandler(w http.ResponseWriter, _ *http.Request) {
	j, err := json.Marshal(Hello{Message: "I'm as ready as I'll ever be!"})
	if err != nil {
		log.Printf("failed to marshal the healthz response: %s", err)
		return
	}

	n, err := w.Write(j)
	if err != nil {
		log.Printf("failed to write the response: %s (written = %d B)", err, n)
	}
}

func putOperationHandler(w http.ResponseWriter, r *http.Request) {
	s, ok := r.Context().Value("serial").(io.WriteCloser)
	if !ok {
		panic("failed to assert the SerialPort struct")
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		replyAndLogError(w, http.StatusBadRequest, "", "Content-Type shall be application/json, not %s", contentType)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		replyAndLogError(w, http.StatusInternalServerError, "", "failed to read the request body: %s", err)
		return
	}

	var o Request
	err = json.Unmarshal(body, &o)
	if err != nil {
		replyAndLogError(w, http.StatusInternalServerError, "", "failed to unmarshal the body: %s", err)
		return
	}

	if len(o.Operation) != 1 {
		replyAndLogError(w, http.StatusBadRequest, "", "invalid operation: length must be 1")
		return
	} else if strings.Index("fbrls", o.Operation) == -1 {
		replyAndLogError(w, http.StatusBadRequest, o.Operation, "unknown operation: %s", o.Operation)
		return
	}

	_, err = s.Write([]byte(o.Operation))
	if err != nil {
		replyAndLogError(w, http.StatusInternalServerError, o.Operation, "failed to write the direction to the MCU: %s", err)
		return
	}

	res := Response{Operation: o.Operation}
	j, err := json.Marshal(res)
	if err != nil {
		log.Printf("failed to marshal the response: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(j)
	if err != nil {
		// w.WriteHeader and replyAndLogError can't be called here since writing the response was failed
		log.Printf("failed to write the response: %s", err)
	}
}
