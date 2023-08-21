package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/RussellLuo/structool"
)

const (
	TypeHTTP = "http"
)

func init() {
	MustRegisterHTTP(orchestrator.GlobalRegistry)
}

func MustRegisterHTTP(r orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeHTTP,
		Constructor: func(decoder *structool.Codec, def *orchestrator.TaskDefinition) (orchestrator.Task, error) {
			h := &HTTP{
				def:    def,
				client: &http.Client{Timeout: def.Timeout},
			}
			if err := decoder.Decode(def.InputTemplate, &h.Input); err != nil {
				return nil, err
			}

			h.Encoding(h.Input.Encoding)

			return h, nil
		},
	})
}

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
	def *orchestrator.TaskDefinition

	client *http.Client
	codec  Codec

	Input struct {
		Encoding string                 `orchestrator:"encoding"`
		Method   string                 `orchestrator:"method"`
		URI      string                 `orchestrator:"uri"`
		Header   map[string][]string    `orchestrator:"header"`
		Body     map[string]interface{} `orchestrator:"body"`
	}

	Expression struct {
		URI    string                 `orchestrator:"uri"`
		Header map[string][]string    `orchestrator:"header"`
		Body   map[string]interface{} `orchestrator:"body"`
	}
}

func NewHTTP(name string) *HTTP {
	h := &HTTP{
		def: &orchestrator.TaskDefinition{
			Name: name,
			Type: TypeHTTP,
		},
		client: &http.Client{},
	}
	return h.Encoding("json")
}

func (h *HTTP) Timeout(timeout time.Duration) *HTTP {
	h.def.Timeout = timeout
	h.client.Timeout = timeout
	return h
}

func (h *HTTP) Encoding(encoding string) *HTTP {
	h.Input.Encoding = encoding

	codec, err := NewCodec(encoding)
	if err != nil {
		panic(err)
	}
	h.codec = codec

	return h
}

func (h *HTTP) Request(method, uri string) *HTTP {
	h.Input.Method = method
	h.Input.URI = uri
	return h
}

func (h *HTTP) Get(uri string) *HTTP {
	return h.Request("GET", uri)
}

func (h *HTTP) Post(uri string) *HTTP {
	return h.Request("POST", uri)
}

func (h *HTTP) Patch(uri string) *HTTP {
	return h.Request("PATCH", uri)
}

func (h *HTTP) Put(uri string) *HTTP {
	return h.Request("PUT", uri)
}

func (h *HTTP) Delete(uri string) *HTTP {
	return h.Request("DELETE", uri)
}

func (h *HTTP) Header(key string, values ...string) *HTTP {
	if h.Input.Header == nil {
		h.Input.Header = make(map[string][]string)
	}
	h.Input.Header[key] = values
	return h
}

func (h *HTTP) Body(body map[string]interface{}) *HTTP {
	h.Input.Body = body
	return h
}

func (h *HTTP) Name() string { return h.def.Name }

func (h *HTTP) String() string {
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, request:%s %s, header:%s, body:%s)",
		h.def.Type,
		h.def.Name,
		h.def.Timeout,
		h.Input.Method,
		h.Input.URI,
		h.Input.Header,
		h.Input.Body,
	)
}

func (h *HTTP) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if err := input.Decoder.Decode(h.Input, &h.Expression); err != nil {
		return nil, err
	}

	var body io.Reader
	if len(h.Expression.Body) > 0 {
		out, err := h.codec.Encode(h.Expression.Body)
		if err != nil {
			return nil, err
		}
		body = out
	}

	req, err := http.NewRequest(h.Input.Method, h.Expression.URI, body)
	if err != nil {
		return nil, err
	}
	for k, v := range h.Expression.Header {
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
