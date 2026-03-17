package openai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zendev-sh/goai/provider"
)

func TestImage_Generate(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("fake-png-data"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("auth = %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":[{"b64_json":"%s"}]}`, encoded)
	}))
	defer server.Close()

	model := Image("dall-e-3", WithAPIKey("test-key"), WithBaseURL(server.URL))
	result, err := model.DoGenerate(t.Context(), provider.ImageParams{
		Prompt: "a cat",
		N:      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("images = %d", len(result.Images))
	}
	if string(result.Images[0].Data) != "fake-png-data" {
		t.Errorf("data = %q", result.Images[0].Data)
	}
}

func TestImage_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprint(w, `{"error":{"message":"Rate limited"}}`)
	}))
	defer server.Close()

	model := Image("dall-e-3", WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := model.DoGenerate(t.Context(), provider.ImageParams{Prompt: "test", N: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImage_NoTokenSource(t *testing.T) {
	model := Image("dall-e-3")
	_, err := model.DoGenerate(t.Context(), provider.ImageParams{Prompt: "test", N: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImage_WithSize(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("img"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if req["size"] != "1024x1024" {
			t.Errorf("size = %v", req["size"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":[{"b64_json":"%s"}]}`, encoded)
	}))
	defer server.Close()

	model := Image("dall-e-3", WithAPIKey("k"), WithBaseURL(server.URL))
	_, err := model.DoGenerate(t.Context(), provider.ImageParams{Prompt: "test", N: 1, Size: "1024x1024"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestImage_ConnectionError(t *testing.T) {
	model := Image("dall-e-3", WithAPIKey("k"), WithBaseURL("http://127.0.0.1:1"))
	_, err := model.DoGenerate(t.Context(), provider.ImageParams{Prompt: "test", N: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImage_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `not json`)
	}))
	defer server.Close()

	model := Image("dall-e-3", WithAPIKey("k"), WithBaseURL(server.URL))
	_, err := model.DoGenerate(t.Context(), provider.ImageParams{Prompt: "test", N: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImage_InvalidBase64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":[{"b64_json":"!!!not-base64!!!"}]}`)
	}))
	defer server.Close()

	model := Image("dall-e-3", WithAPIKey("k"), WithBaseURL(server.URL))
	_, err := model.DoGenerate(t.Context(), provider.ImageParams{Prompt: "test", N: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImage_ReadBodyError(t *testing.T) {
	// Custom transport that returns a 200 OK with a body that errors on Read.
	transport := &errorBodyTransport{}
	client := &http.Client{Transport: transport}
	model := Image("dall-e-3", WithAPIKey("k"), WithBaseURL("http://fake"), WithHTTPClient(client))
	_, err := model.DoGenerate(t.Context(), provider.ImageParams{Prompt: "test", N: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

// errorBodyTransport returns a 200 response with a body that fails on Read.
type errorBodyTransport struct{}

func (t *errorBodyTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(&failReader{}),
		Header:     make(http.Header),
	}, nil
}

type failReader struct{}

func (f *failReader) Read(_ []byte) (int, error) {
	return 0, fmt.Errorf("read error")
}

func TestImage_WithHeaders(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("img"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "val" {
			t.Errorf("X-Custom = %q", r.Header.Get("X-Custom"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":[{"b64_json":"%s"}]}`, encoded)
	}))
	defer server.Close()

	model := Image("dall-e-3", WithAPIKey("k"), WithBaseURL(server.URL), WithHeaders(map[string]string{"X-Custom": "val"}))
	_, err := model.DoGenerate(t.Context(), provider.ImageParams{Prompt: "test", N: 1})
	if err != nil {
		t.Fatal(err)
	}
}

func TestImage_WithHTTPClient(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("img"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"data":[{"b64_json":"%s"}]}`, encoded)
	}))
	defer server.Close()

	customClient := &http.Client{}
	model := Image("dall-e-3", WithAPIKey("k"), WithBaseURL(server.URL), WithHTTPClient(customClient))
	result, err := model.DoGenerate(t.Context(), provider.ImageParams{Prompt: "test", N: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Images) != 1 {
		t.Errorf("images = %d", len(result.Images))
	}
}

func TestImage_ModelID(t *testing.T) {
	model := Image("dall-e-3", WithAPIKey("k"))
	if model.ModelID() != "dall-e-3" {
		t.Errorf("ModelID() = %q", model.ModelID())
	}
}

func TestImage_EnvVarResolution(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	m := Image("dall-e-3")
	im := m.(*imageModel)
	if im.opts.tokenSource == nil {
		t.Error("tokenSource should be set from OPENAI_API_KEY")
	}
}

func TestImage_EnvVarBaseURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	t.Setenv("OPENAI_BASE_URL", "https://custom.openai.com/v1")
	m := Image("dall-e-3")
	im := m.(*imageModel)
	if im.opts.baseURL != "https://custom.openai.com/v1" {
		t.Errorf("baseURL = %q", im.opts.baseURL)
	}
}

func TestImage_EnvVarNotOverrideExplicit(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "https://env.url")
	m := Image("dall-e-3", WithAPIKey("explicit"), WithBaseURL("https://explicit.url"))
	im := m.(*imageModel)
	if im.opts.baseURL != "https://explicit.url" {
		t.Errorf("baseURL = %q", im.opts.baseURL)
	}
}
