package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

// Models

type Request struct {
	Operation string `json:"operation,omitempty"`
}

type Response struct {
	Operation string  `json:"operation"`
	Error     *string `json:"error"`
}

func (r Response) Marshal() []byte {
	j, _ := json.Marshal(r) // this will never fail
	return j
}

// Shorthands

func StrPtr(s string) *string {
	return &s
}

func ReplyAndLogError(w http.ResponseWriter, status int, operation, error string, a ...any) {
	r := Response{
		Operation: operation,
		Error:     StrPtr(fmt.Sprintf(error, a...)),
	}

	w.WriteHeader(status)
	n, err := w.Write(r.Marshal())
	if err != nil {
		log.Printf("failed to write the response: %s (written = %d B)", err, n)
	}

	log.Printf(*r.Error)
}

func main() {
	s, err := NewSerialPort("/dev/ttyACM0", 115200)
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), "serial", s)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Get("/healthz", HealthzHandler)
	r.Put("/operation", OperationPutHandler)

	if err := http.ListenAndServe(":8080", r); err != nil {
		panic(err)
	}
}

func HealthzHandler(w http.ResponseWriter, _ *http.Request) {
	n, err := w.Write([]byte("{\"message\":\"I'm as ready as I'll ever be!\"}"))
	if err != nil {
		log.Printf("failed to write the response: %s (written = %d B)", err, n)
	}
}

func OperationPutHandler(w http.ResponseWriter, r *http.Request) {
	s, ok := r.Context().Value("serial").(*SerialPort)
	if !ok {
		panic("failed to assert the SerialPort struct")
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ReplyAndLogError(w, http.StatusInternalServerError, "", "failed to read the request body: %s", err)
		return
	}

	var o Request
	err = json.Unmarshal(body, &o)
	if err != nil {
		ReplyAndLogError(w, http.StatusInternalServerError, "", "failed to unmarshal the body: %s", err)
		return
	}

	if len(o.Operation) != 1 {
		ReplyAndLogError(w, http.StatusBadRequest, "", "invalid operation: length must be 1")
		return
	} else if strings.Index("fbrls", o.Operation) == -1 {
		ReplyAndLogError(w, http.StatusBadRequest, o.Operation, "unknown operation: %s", o.Operation)
		return
	}

	_, err = s.Write([]byte(o.Operation))
	if err != nil {
		ReplyAndLogError(w, http.StatusInternalServerError, o.Operation, "failed to write the direction to the MCU: %s", err)
		return
	}

	_, err = w.Write(Response{Operation: o.Operation}.Marshal())
	if err != nil {
		// w.WriteHeader and ReplyAndLogError can't be called here since writing the header
		// and the body had already attempted (and failed)
		log.Printf("failed to write the response: %s", err)
	}
}
