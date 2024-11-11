package internal

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"strings"
)

type JsonBodyConverter struct{}

func (bc JsonBodyConverter) ConvertBody(translated map[string]string) ([]byte, error) {
	result, err := json.Marshal(translated)
	if err != nil {
		return result, err
	}
	return result, nil
}

type YmlBodyConverter struct{}

func (bc YmlBodyConverter) ConvertBody(translated map[string]string) ([]byte, error) {
	result, err := yaml.Marshal(translated)
	if err != nil {
		return result, err
	}
	return result, nil
}

type PimTranslatorBodyConvertor struct{}

func (t PimTranslatorBodyConvertor) ConvertBody(translated map[string]string) ([]byte, error) {
	var result string

	for k, v := range translated {
		result += fmt.Sprintf("'%v': '%v'\n", t.sanitize(k), t.sanitize(v))
	}

	return []byte(result), nil
}

func (t PimTranslatorBodyConvertor) sanitize(v string) string {
	return strings.ReplaceAll(v, "'", "")
}
