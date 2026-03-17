package httpc

import (
	"io"
	"testing"
)

func TestMustMarshalJSON(t *testing.T) {
	data := MustMarshalJSON(map[string]any{"key": "value", "num": 42})
	if len(data) == 0 {
		t.Fatal("expected non-empty JSON")
	}
	want := `{"key":"value","num":42}`
	if string(data) != want {
		t.Errorf("got %s, want %s", data, want)
	}
}

func TestMustMarshalJSON_Panic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for unmarshalable value")
		}
	}()

	// Channels cannot be marshaled to JSON.
	MustMarshalJSON(make(chan int))
}

func TestMustNewRequest(t *testing.T) {
	body := []byte(`{"prompt":"hello"}`)
	req := MustNewRequest(t.Context(), "POST", "https://api.example.com/v1/chat", body)

	if req.Method != "POST" {
		t.Errorf("Method = %q, want POST", req.Method)
	}
	if req.URL.String() != "https://api.example.com/v1/chat" {
		t.Errorf("URL = %q", req.URL.String())
	}

	reqBody, _ := io.ReadAll(req.Body)
	if string(reqBody) != string(body) {
		t.Errorf("Body = %q, want %q", reqBody, body)
	}
}

func TestMustNewRequest_NilBody(t *testing.T) {
	req := MustNewRequest(t.Context(), "GET", "https://api.example.com/v1/models", nil)

	if req.Method != "GET" {
		t.Errorf("Method = %q, want GET", req.Method)
	}
	if req.Body != nil {
		t.Error("expected nil body for GET request")
	}
}

func TestMustNewRequest_Panic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil context")
		}
	}()

	// nil context causes http.NewRequestWithContext to fail.
	MustNewRequest(nil, "GET", "https://example.com", nil) //nolint:staticcheck
}

func TestParseDataURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantMedia string
		wantData  string
		wantOK    bool
	}{
		{
			name:      "valid data URL",
			url:       "data:image/png;base64,abc123",
			wantMedia: "image/png",
			wantData:  "abc123",
			wantOK:    true,
		},
		{
			name:   "no base64 marker",
			url:    "data:image/png,abc",
			wantOK: false,
		},
		{
			name:   "not a data URL",
			url:    "https://example.com",
			wantOK: false,
		},
		{
			name:   "empty string",
			url:    "",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			media, data, ok := ParseDataURL(tt.url)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if media != tt.wantMedia {
				t.Errorf("mediaType = %q, want %q", media, tt.wantMedia)
			}
			if data != tt.wantData {
				t.Errorf("data = %q, want %q", data, tt.wantData)
			}
		})
	}
}
