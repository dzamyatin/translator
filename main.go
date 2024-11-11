package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type ITranslator interface {
	translate(string) (string, error)
}

type Translator struct{}

func (t Translator) translate(word string) (string, error) {

	return word, nil
}

type Handler struct {
	translator ITranslator
}

func NewHandler() Handler {
	return Handler{Translator{}}
}
func (h Handler) process(toTranslate map[string]string) map[string]string {
	translated := make(map[string]string)

	for k, v := range toTranslate {
		res, err := h.translator.translate(v)
		if err != nil {
			continue
		}
		translated[k] = res
	}

	return toTranslate
}
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var toTranslate map[string]string
	var translated map[string]string

	body, err := io.ReadAll(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &toTranslate)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if toTranslate == nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	translated = h.process(toTranslate)

	result, err := json.Marshal(translated)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(result)
}

func main() {
	fmt.Println(`Listen and serve POST http://localhost:8080/ {"key": "value"}`)

	handler := NewHandler()

	err := http.ListenAndServe("0.0.0.0:8080", handler)

	if err != nil {
		log.Fatal(err)
	}
}
