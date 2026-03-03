package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"openIntern/internal/models"
	"strings"
	"time"

	einoTool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	apiPluginMethodGET          = "GET"
	apiPluginMethodPOST         = "POST"
	apiPluginResponseStreaming  = "streaming"
	apiPluginResponseSync       = "non_streaming"
	apiPluginMaxResponseBytes   = 128 * 1024
	apiPluginStreamBufferSize   = 16
	apiPluginChunkSize          = 4096
	apiPluginScannerBufferBytes = 256 * 1024
)

type apiPluginInvokeInput struct {
	Query  map[string]any
	Header map[string]any
	Body   map[string]any
}

type apiPluginToolBase struct {
	def          models.Tool
	info         *schema.ToolInfo
	timeout      time.Duration
	queryFields  []codePluginField
	headerFields []codePluginField
	bodyFields   []codePluginField
}

type apiInvokableTool struct {
	*apiPluginToolBase
}

type apiStreamableTool struct {
	*apiPluginToolBase
}

var _ einoTool.InvokableTool = (*apiInvokableTool)(nil)
var _ einoTool.StreamableTool = (*apiStreamableTool)(nil)

func NewAPIPluginTool(def models.Tool) (einoTool.BaseTool, error) {
	base, err := newAPIPluginToolBase(def)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(def.ToolResponseMode)) {
	case "", apiPluginResponseSync:
		return &apiInvokableTool{apiPluginToolBase: base}, nil
	case apiPluginResponseStreaming:
		return &apiStreamableTool{apiPluginToolBase: base}, nil
	default:
		return nil, fmt.Errorf("unsupported tool_response_mode: %s", def.ToolResponseMode)
	}
}

func newAPIPluginToolBase(def models.Tool) (*apiPluginToolBase, error) {
	info, err := buildCodePluginToolInfo(def)
	if err != nil {
		return nil, err
	}

	queryFields, err := decodeCodePluginFields(def.QueryFieldsJSON)
	if err != nil {
		return nil, err
	}
	headerFields, err := decodeCodePluginFields(def.HeaderFieldsJSON)
	if err != nil {
		return nil, err
	}
	bodyFields, err := decodeCodePluginFields(def.BodyFieldsJSON)
	if err != nil {
		return nil, err
	}

	timeout := time.Duration(def.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &apiPluginToolBase{
		def:          def,
		info:         info,
		timeout:      timeout,
		queryFields:  queryFields,
		headerFields: headerFields,
		bodyFields:   bodyFields,
	}, nil
}

func (t *apiPluginToolBase) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

func (t *apiInvokableTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einoTool.Option) (string, error) {
	_ = opts

	req, cleanup, err := t.buildRequest(ctx, argumentsInJSON)
	if err != nil {
		return "", err
	}
	if cleanup != nil {
		defer cleanup()
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, truncated, err := readAPIPluginBody(resp.Body, apiPluginMaxResponseBytes)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", buildAPIPluginStatusError(resp.StatusCode, resp.Status, body, truncated)
	}

	return buildAPIPluginResponse(body, truncated)
}

func (t *apiStreamableTool) StreamableRun(ctx context.Context, argumentsInJSON string, opts ...einoTool.Option) (*schema.StreamReader[string], error) {
	_ = opts

	req, cleanup, err := t.buildRequest(ctx, argumentsInJSON)
	if err != nil {
		return nil, err
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, truncated, readErr := readAPIPluginBody(resp.Body, apiPluginMaxResponseBytes)
		_ = resp.Body.Close()
		if cleanup != nil {
			cleanup()
		}
		if readErr != nil {
			return nil, readErr
		}
		return nil, buildAPIPluginStatusError(resp.StatusCode, resp.Status, body, truncated)
	}

	sr, sw := schema.Pipe[string](apiPluginStreamBufferSize)
	go func() {
		defer sw.Close()
		defer resp.Body.Close()
		if cleanup != nil {
			defer cleanup()
		}

		if err := streamAPIPluginResponse(resp, sw); err != nil {
			sw.Send("", err)
		}
	}()
	return sr, nil
}

func (t *apiPluginToolBase) buildRequest(ctx context.Context, argumentsInJSON string) (*http.Request, func(), error) {
	input, err := t.decodeInput(argumentsInJSON)
	if err != nil {
		return nil, nil, err
	}

	endpoint, err := url.Parse(strings.TrimSpace(t.def.RequestURL))
	if err != nil {
		return nil, nil, errors.New("request_url must be a valid url")
	}

	query := endpoint.Query()
	if err := appendAPIPluginQuery(query, input.Query); err != nil {
		return nil, nil, err
	}
	endpoint.RawQuery = query.Encode()

	method := strings.ToUpper(strings.TrimSpace(t.def.APIRequestType))
	if method == "" {
		method = apiPluginMethodGET
	}

	bodyReader, contentType, err := buildAPIPluginRequestBody(method, input.Body)
	if err != nil {
		return nil, nil, err
	}

	requestCtx := ctx
	var cleanup func()
	if t.timeout > 0 {
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(ctx, t.timeout)
		cleanup = cancel
	}

	req, err := http.NewRequestWithContext(requestCtx, method, endpoint.String(), bodyReader)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}

	if err := appendAPIPluginHeaders(req.Header, input.Header); err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, err
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	return req, cleanup, nil
}

func (t *apiPluginToolBase) decodeInput(argumentsInJSON string) (apiPluginInvokeInput, error) {
	raw, err := decodeCodePluginInput(argumentsInJSON)
	if err != nil {
		return apiPluginInvokeInput{}, err
	}

	for key := range raw {
		switch key {
		case "query", "header", "body":
		default:
			return apiPluginInvokeInput{}, fmt.Errorf("%s is not allowed", key)
		}
	}

	query, err := decodeAPIPluginSection(raw, "query", t.queryFields)
	if err != nil {
		return apiPluginInvokeInput{}, err
	}
	header, err := decodeAPIPluginSection(raw, "header", t.headerFields)
	if err != nil {
		return apiPluginInvokeInput{}, err
	}
	body, err := decodeAPIPluginSection(raw, "body", t.bodyFields)
	if err != nil {
		return apiPluginInvokeInput{}, err
	}

	return apiPluginInvokeInput{
		Query:  query,
		Header: header,
		Body:   body,
	}, nil
}

func decodeAPIPluginSection(input map[string]any, key string, fields []codePluginField) (map[string]any, error) {
	value, ok := input[key]
	if !ok {
		return nil, nil
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("%s is not allowed", key)
	}

	obj, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be object", key)
	}
	if err := validateCodePluginObject(obj, fields, key); err != nil {
		return nil, err
	}
	return obj, nil
}

func appendAPIPluginQuery(values url.Values, input map[string]any) error {
	for key, value := range input {
		items, err := flattenAPIPluginValues(value)
		if err != nil {
			return fmt.Errorf("query.%s %w", key, err)
		}
		for _, item := range items {
			values.Add(key, item)
		}
	}
	return nil
}

func appendAPIPluginHeaders(headers http.Header, input map[string]any) error {
	for key, value := range input {
		items, err := flattenAPIPluginValues(value)
		if err != nil {
			return fmt.Errorf("header.%s %w", key, err)
		}
		headers.Set(key, strings.Join(items, ","))
	}
	return nil
}

func flattenAPIPluginValues(value any) ([]string, error) {
	switch v := value.(type) {
	case []any:
		if len(v) == 0 {
			return []string{}, nil
		}
		result := make([]string, 0, len(v))
		for _, item := range v {
			text, err := stringifyAPIPluginScalar(item)
			if err != nil {
				return nil, err
			}
			result = append(result, text)
		}
		return result, nil
	default:
		text, err := stringifyAPIPluginScalar(v)
		if err != nil {
			return nil, err
		}
		return []string{text}, nil
	}
}

func stringifyAPIPluginScalar(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case json.Number:
		return v.String(), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(v), nil
	case nil:
		return "", errors.New("must not be null")
	default:
		return "", errors.New("must be a scalar or scalar array")
	}
}

func buildAPIPluginRequestBody(method string, body map[string]any) (io.Reader, string, error) {
	switch method {
	case apiPluginMethodGET:
		if len(body) > 0 {
			return nil, "", errors.New("body is not allowed for GET api tools")
		}
		return nil, "", nil
	case apiPluginMethodPOST:
		if len(body) == 0 {
			return nil, "", nil
		}
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(payload), "application/json", nil
	default:
		return nil, "", fmt.Errorf("unsupported api_request_type: %s", method)
	}
}

func readAPIPluginBody(body io.Reader, limit int) (string, bool, error) {
	if limit <= 0 {
		limit = apiPluginMaxResponseBytes
	}

	data, err := io.ReadAll(io.LimitReader(body, int64(limit+1)))
	if err != nil {
		return "", false, err
	}
	if len(data) > limit {
		return string(data[:limit]), true, nil
	}
	return string(data), false, nil
}

func buildAPIPluginStatusError(statusCode int, status string, body string, truncated bool) error {
	detail := strings.TrimSpace(body)
	if truncated {
		detail += " ...(truncated)"
	}
	if detail == "" {
		detail = http.StatusText(statusCode)
	}
	if strings.TrimSpace(status) == "" {
		status = fmt.Sprintf("status %d", statusCode)
	}
	return fmt.Errorf("%s: %s", status, detail)
}

func buildAPIPluginResponse(body string, truncated bool) (string, error) {
	if !truncated {
		return body, nil
	}

	payload, err := json.Marshal(map[string]any{
		"truncated": true,
		"body":      body,
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func streamAPIPluginResponse(resp *http.Response, sw *schema.StreamWriter[string]) error {
	contentType := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	switch {
	case strings.Contains(contentType, "text/event-stream"):
		return streamAPIPluginSSE(resp.Body, sw)
	case strings.Contains(contentType, "application/x-ndjson"),
		strings.Contains(contentType, "application/ndjson"),
		strings.Contains(contentType, "application/jsonl"):
		return streamAPIPluginNDJSON(resp.Body, sw)
	default:
		return streamAPIPluginChunks(resp.Body, sw)
	}
}

func streamAPIPluginSSE(body io.Reader, sw *schema.StreamWriter[string]) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), apiPluginScannerBufferBytes)

	dataLines := make([]string, 0, 2)
	flush := func() bool {
		if len(dataLines) == 0 {
			return false
		}
		chunk := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		return sw.Send(chunk, nil)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if flush() {
				return nil
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimLeft(line[len("data:"):], " \t"))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if flush() {
		return nil
	}
	return nil
}

func streamAPIPluginNDJSON(body io.Reader, sw *schema.StreamWriter[string]) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), apiPluginScannerBufferBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if sw.Send(line, nil) {
			return nil
		}
	}
	return scanner.Err()
}

func streamAPIPluginChunks(body io.Reader, sw *schema.StreamWriter[string]) error {
	buf := make([]byte, apiPluginChunkSize)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if sw.Send(string(buf[:n]), nil) {
				return nil
			}
		}
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
}
