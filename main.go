package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"translator/internal"
)

var failedExecutable = errors.New("Command failed")

type ITranslator interface {
	translate(string) (string, error)
}

type Translator struct{}

func (t Translator) translate(word string) (string, error) {
	cmd := exec.Command(
		"trans",
		"en:ru",
		"-no-view",
		"-j",
		"-show-alternatives",
		"n",
		"-show-dictionary",
		"n",
		"-b",
		"-no-pager",
		word,
	)

	errPipe, _ := cmd.StderrPipe()

	pipe, err := cmd.Output()
	if err != nil {
		return "", err
	}

	if !cmd.ProcessState.Success() {
		fmt.Printf("ERROR: Cant execute: %v by: %v", word, errPipe)
		return "", failedExecutable
	}

	return strings.TrimRight(string(pipe), "\n"), nil
}

type IBodyConverter interface {
	ConvertBody(translated map[string]string) ([]byte, error)
}

type Handler struct {
	translator    ITranslator
	bodyConverter IBodyConverter
}

func NewHandler() Handler {
	return Handler{
		Translator{},
		internal.PimTranslatorBodyConvertor{},
		//internal.YmlBodyConverter{},
	}
}
func (h Handler) process(toTranslate map[string]string) map[string]string {
	translated := make(map[string]string)
	var i int

	fmt.Printf("Found: %v", len(toTranslate))
	for k, v := range toTranslate {
		i++

		res, err := h.translator.translate(v)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("el:%v |%v|%v|\n", i, v, res)

		if !h.isValid(v, res) {
			continue
		}

		translated[k] = res
	}

	return translated
}
func (h Handler) isValid(v string, res string) bool {
	if res == v {
		fmt.Println("Not changed")
		return false
	}

	if res == "" {
		fmt.Println("Empty")
		return false
	}

	return true
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

	result, err := h.bodyConverter.ConvertBody(translated)

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
