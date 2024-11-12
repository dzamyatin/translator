package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"translator/internal"
)

var failedExecutable = errors.New("Command failed")

type ITranslator interface {
	translate(string) (string, error)
}

type Translator struct{}

func (t Translator) translate(word string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	//ctx := context.Background()
	//
	cmd := exec.CommandContext(
		ctx,
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

	//cmd := exec.Command(
	//	"trans",
	//	"en:ru",
	//	"-no-view",
	//	"-j",
	//	"-show-alternatives",
	//	"n",
	//	"-show-dictionary",
	//	"n",
	//	"-b",
	//	"-no-pager",
	//	word,
	//)

	cmd.WaitDelay = 2 * time.Second

	timer := time.NewTimer(3 * time.Second)
	go func() {
		select {
		case <-timer.C:
			log.Println("TIMEOUT")
			err := cmd.Cancel()
			if err != nil {
				log.Println(err)
			}
		case <-ctx.Done():
			return
		}
	}()

	errPipe, _ := cmd.StderrPipe()
	pipe, err := cmd.Output()

	cancel()

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

type kv struct {
	k, v string
}

func (h Handler) process(toTranslate map[string]string) map[string]string {
	translated := make(map[string]string)

	chIn := make(chan kv)
	chOut := make(chan kv)

	wg := &sync.WaitGroup{}
	i := atomic.Int64{}
	for k := 0; k < 30; k++ {
		wg.Add(1)
		go func() {
			for {
				select {
				case in, ok := <-chIn:

					if !ok {
						wg.Done()
						return
					}

					res, err := h.translator.translate(in.v)
					if err != nil {
						fmt.Println(err)
						break
					}
					fmt.Printf("el:%v |%v|%v|\n", i.Add(1), in.k, res)

					if !h.isValid(in.v, res) {
						break
					}

					chOut <- kv{in.k, res}
				}
			}
		}()
	}

	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	go func() {
		for v := range chOut {
			translated[v.k] = v.v
		}
		wg2.Done()
	}()

	for k, v := range toTranslate {
		chIn <- kv{k, v}
	}

	close(chIn)
	wg.Wait()
	close(chOut)
	wg2.Wait()

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
