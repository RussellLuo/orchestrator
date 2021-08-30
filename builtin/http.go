package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/RussellLuo/orchestrator"
)

const (
	TypeHTTP = "http"
)

type Codec interface {
	Decode(in io.Reader, out interface{}) error
	Encode(in interface{}) (out io.Reader, err error)
}

func NewCodec(encoding string) (Codec, error) {
	switch encoding {
	case "", "json":
		return JSON{}, nil
	default:
		return nil, fmt.Errorf("unsupported encoding %q", encoding)
	}
}

type JSON struct{}

func (j JSON) Decode(in io.Reader, out interface{}) error {
	return json.NewDecoder(in).Decode(out)
}

func (j JSON) Encode(in interface{}) (io.Reader, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}

// HTTP is a leaf task that is used to make calls to another service over HTTP.
type HTTP struct {
	*orchestrator.TaskDefinition

	client *http.Client
	codec  Codec

	input struct {
		Encoding string                 `orchestrator:"encoding"`
		Method   string                 `orchestrator:"method"`
		URI      string                 `orchestrator:"uri"`
		Header   map[string][]string    `orchestrator:"header"`
		Body     map[string]interface{} `orchestrator:"body"`
	}
}

func NewHTTP(def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
	if def.Type != TypeHTTP {
		def.Type = TypeHTTP
	}

	h := &HTTP{
		TaskDefinition: def,
		client:         &http.Client{Timeout: def.Timeout},
	}

	codec, err := NewCodec(h.input.Encoding)
	if err != nil {
		return nil, err
	}
	h.codec = codec

	return h, nil
}

func (h *HTTP) Definition() *orchestrator.TaskDefinition {
	return h.TaskDefinition
}

func (h *HTTP) Execute(ctx context.Context, decoder *orchestrator.Decoder) (orchestrator.Output, error) {
	if err := decoder.Decode(h.InputTemplate, &h.input); err != nil {
		return nil, err
	}

	var body io.Reader
	if len(h.input.Body) > 0 {
		out, err := h.codec.Encode(h.input.Body)
		if err != nil {
			return nil, err
		}
		body = out
	}

	req, err := http.NewRequest(h.input.Method, h.input.URI, body)
	if err != nil {
		return nil, err
	}
	for k, v := range h.input.Header {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respBody map[string]interface{}
	if err := h.codec.Decode(resp.Body, &respBody); err != nil {
		return nil, err
	}

	return orchestrator.Output{
		"status": resp.StatusCode,
		"header": resp.Header,
		"body":   respBody,
	}, nil
}
