package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/RussellLuo/orchestrator"
	"github.com/olivere/ndjson"
)

const (
	TypeHTTP = "http"
)

func init() {
	MustRegisterHTTP(orchestrator.GlobalRegistry)
}

func MustRegisterHTTP(r *orchestrator.Registry) {
	r.MustRegister(&orchestrator.TaskFactory{
		Type: TypeHTTP,
		New:  func() orchestrator.Task { return new(HTTP) },
	})
}

type Codec interface {
	Decode(in io.Reader, out any) error
	Encode(in any) (out io.Reader, err error)
}

func NewCodec(encoding string) (Codec, error) {
	switch encoding {
	case "json":
		return JSON{}, nil
	default:
		return nil, fmt.Errorf("unsupported encoding %q", encoding)
	}
}

type JSON struct{}

func (j JSON) Decode(in io.Reader, out any) error {
	return json.NewDecoder(in).Decode(out)
}

func (j JSON) Encode(in any) (io.Reader, error) {
	data, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}

// HTTP is a leaf task that is used to make calls to another service over HTTP.
type HTTP struct {
	orchestrator.TaskHeader

	Input struct {
		Encoding string                                 `json:"encoding"`
		Method   orchestrator.Expr[string]              `json:"method"`
		URI      orchestrator.Expr[string]              `json:"uri"`
		Query    orchestrator.Expr[map[string]any]      `json:"query"`
		Header   orchestrator.Expr[map[string][]string] `json:"header"`
		Body     orchestrator.Expr[map[string]any]      `json:"body"`
		// A filter expression for extracting fields from a server-sent event.
		SSEFilter string `json:"sse_filter"`
	} `json:"input"`

	client *http.Client
	codec  Codec
}

func (h *HTTP) Init(r *orchestrator.Registry) error {
	h.client = &http.Client{Timeout: h.Timeout}
	h.Encoding(h.Input.Encoding)
	return nil
}

func (h *HTTP) Encoding(encoding string) *HTTP {
	if encoding == "" {
		encoding = "json"
	}
	h.Input.Encoding = encoding

	codec, err := NewCodec(encoding)
	if err != nil {
		panic(err)
	}
	h.codec = codec

	return h
}

func (h *HTTP) getEncodingHeader() map[string][]string {
	switch h.Input.Encoding {
	case "json":
		return map[string][]string{
			"Content-Type": {"application/json"},
			"Accept":       {"application/json"},
		}
	default:
		return nil
	}
}

func (h *HTTP) String() string {
	return fmt.Sprintf(
		"%s(name:%s, timeout:%s, request:%s %v, header:%v, body:%v)",
		h.Type,
		h.Name,
		h.Timeout,
		h.Input.Method.Expr,
		h.Input.URI.Expr,
		h.Input.Header.Expr,
		h.Input.Body.Expr,
	)
}

func (h *HTTP) Execute(ctx context.Context, input orchestrator.Input) (orchestrator.Output, error) {
	if err := h.Input.Method.Evaluate(input); err != nil {
		return nil, err
	}
	if err := h.Input.URI.Evaluate(input); err != nil {
		return nil, err
	}
	if err := h.Input.Query.Evaluate(input); err != nil {
		return nil, err
	}
	if err := h.Input.Header.Evaluate(input); err != nil {
		return nil, err
	}
	if err := h.Input.Body.Evaluate(input); err != nil {
		return nil, err
	}

	var body io.Reader
	if len(h.Input.Body.Value) > 0 {
		out, err := h.codec.Encode(h.Input.Body.Value)
		if err != nil {
			return nil, err
		}
		body = out
	}

	req, err := http.NewRequestWithContext(ctx, h.Input.Method.Value, h.Input.URI.Value, body)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for key, value := range h.Input.Query.Value {
		q.Add(key, fmt.Sprintf("%v", value))
	}
	req.URL.RawQuery = q.Encode()

	for k, v := range h.Input.Header.Value {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}
	for k, v := range h.getEncodingHeader() {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}

	respContentType := resp.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(respContentType)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}

	var respBody any

	switch mediatype {
	case "text/event-stream": // Sever-Sent Events
		respBody = orchestrator.NewIterator(ctx, func(sender *orchestrator.IteratorSender) {
			defer sender.End() // End the iteration

			defer resp.Body.Close()
			reader := NewEventStreamReader(resp.Body, 1<<16)

			for {
				event, err := reader.ReadEvent()
				if err != nil {
					if err == io.EOF {
						// Reach the end of the response payload.
						return
					}

					sender.Send(nil, err)
					return
				}

				// Send the event if it has something useful.
				if len(event.Data) > 0 {
					data := string(event.Data)
					if h.Input.SSEFilter != "" {
						evaluator := orchestrator.NewEvaluatorWithData(map[string]any{"data": data})
						value, err := evaluator.Evaluate(h.Input.SSEFilter)
						if err != nil {
							sender.Send(nil, fmt.Errorf("failed to evaluate '%s': %v", h.Input.SSEFilter, err))
							return
						}
						// We assume that the event data is always a string.
						data = fmt.Sprintf("%v", value)
					}
					// For simplicity, currently we only handle data-only sever-sent events.
					if continue_ := sender.Send(orchestrator.Output{"data": data}, nil); !continue_ {
						return
					}
				}
			}
		})

	case "application/x-ndjson": // Newline-delimited JSON
		respBody = orchestrator.NewIterator(ctx, func(sender *orchestrator.IteratorSender) {
			defer sender.End() // End the iteration

			defer resp.Body.Close()
			reader := ndjson.NewReaderSize(resp.Body, 1<<16)

			for reader.Next() {
				dataBytes := reader.Bytes()
				data := string(dataBytes)
				if len(data) > 0 {
					if h.Input.SSEFilter != "" {
						evaluator := orchestrator.NewEvaluatorWithData(map[string]any{"data": data})
						value, err := evaluator.Evaluate(h.Input.SSEFilter)
						if err != nil {
							sender.Send(nil, fmt.Errorf("failed to evaluate '%s': %v", h.Input.SSEFilter, err))
							return
						}
						// We assume that the event data is always a string.
						data = fmt.Sprintf("%v", value)
					}

					// For compatibility, currently we send the data as a string (i.e. mimic a server-sent event).
					if continue_ := sender.Send(orchestrator.Output{"data": data}, nil); !continue_ {
						return
					}
				}
			}

			if err := reader.Err(); err != nil {
				sender.Send(nil, err)
				return
			}
		})

	case "application/json": // JSON
		defer resp.Body.Close()
		var m any
		if err := h.codec.Decode(resp.Body, &m); err != nil {
			return nil, err
		}
		respBody = m

	default: // Other content
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil && err != io.EOF {
			return nil, err
		}
		respBody = string(b)
	}

	return orchestrator.Output{
		"status": resp.StatusCode,
		"header": resp.Header,
		"body":   respBody,
	}, nil
}

type HTTPBuilder struct {
	task *HTTP
}

func NewHTTP(name string) *HTTPBuilder {
	task := &HTTP{
		TaskHeader: orchestrator.TaskHeader{
			Name: name,
			Type: TypeHTTP,
		},
		client: &http.Client{},
	}
	task = task.Encoding("json")
	return &HTTPBuilder{task: task}
}

func (b *HTTPBuilder) Timeout(timeout time.Duration) *HTTPBuilder {
	b.task.Timeout = timeout
	b.task.client.Timeout = timeout
	return b
}

func (b *HTTPBuilder) Request(method, uri string) *HTTPBuilder {
	b.task.Input.Method = orchestrator.Expr[string]{Expr: method}
	b.task.Input.URI = orchestrator.Expr[string]{Expr: uri}
	return b
}

func (b *HTTPBuilder) Get(uri string) *HTTPBuilder {
	return b.Request("GET", uri)
}

func (b *HTTPBuilder) Post(uri string) *HTTPBuilder {
	return b.Request("POST", uri)
}

func (b *HTTPBuilder) Patch(uri string) *HTTPBuilder {
	return b.Request("PATCH", uri)
}

func (b *HTTPBuilder) Put(uri string) *HTTPBuilder {
	return b.Request("PUT", uri)
}

func (b *HTTPBuilder) Delete(uri string) *HTTPBuilder {
	return b.Request("DELETE", uri)
}

func (b *HTTPBuilder) Query(key string, value any) *HTTPBuilder {
	if b.task.Input.Query.Expr == nil {
		b.task.Input.Query = orchestrator.Expr[map[string]any]{Expr: make(map[string]any)}
	}
	b.task.Input.Query.Expr.(map[string]any)[key] = value
	return b
}

func (b *HTTPBuilder) Header(key string, values ...string) *HTTPBuilder {
	if b.task.Input.Header.Expr == nil {
		b.task.Input.Header = orchestrator.Expr[map[string][]string]{Expr: make(map[string][]string)}
	}
	b.task.Input.Header.Expr.(map[string][]string)[key] = values
	return b
}

func (b *HTTPBuilder) Body(body map[string]any) *HTTPBuilder {
	b.task.Input.Body = orchestrator.Expr[map[string]any]{Expr: body}
	return b
}

func (b *HTTPBuilder) Build() orchestrator.Task {
	return b.task
}
