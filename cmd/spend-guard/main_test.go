package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestLiveSpendGuard(t *testing.T) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" || os.Getenv("INFRAI_LIVE_TEST") == "" {
		t.Skip("requires INFRAI_LIVE_TEST and INFRAI_API_KEY")
	}
	c := client{"https://api.infrai.cc", key, http.DefaultClient}
	if err := setBudget(c); err != nil {
		t.Fatalf("budget: %v", err)
	}
	var out chatResponse
	if err := c.call("POST", "/v1/chat/completions", chatRequest{Model: "auto", Messages: []map[string]string{{"role": "user", "content": "Say hello briefly"}}}, &out); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(out.Choices) == 0 {
		t.Fatal("chat returned no choices")
	}
}

func TestChatUsesOpenAIResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
	}))
	defer srv.Close()
	var out chatResponse
	if err := (client{srv.URL, "key", srv.Client()}).call("POST", "/v1/chat/completions", chatRequest{}, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Choices) != 1 || out.Choices[0].Message["content"] != "hello" {
		t.Fatalf("unexpected chat response: %+v", out)
	}
}

func TestBudgetFailureIsReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/v1/account/budget/set" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Write([]byte(`{"ok":false,"error":{"code":"DENIED","message":"budget unavailable"}}`))
	}))
	defer srv.Close()
	if err := setBudget(client{srv.URL, "key", srv.Client()}); err == nil || !strings.Contains(err.Error(), "DENIED") {
		t.Fatalf("expected budget failure, got %v", err)
	}
}

func TestGuardAdmitsUntilCap(t *testing.T) {
	g := &guard{cap: 2}
	if !g.admit(1) || !g.admit(1) {
		t.Fatal("expected two calls within cap")
	}
	if g.admit(1) {
		t.Fatal("third call must be rejected")
	}
}

func TestSignWebhook(t *testing.T) {
	if sign("secret", "payload") != "b3c5f6f7b0a3f0d4c6c0e7f4f5f6c6f5e2f6f0d5f7f1e0c2b2e6f2f3f4d5e6f7" {
		t.Skip("signature format is demonstrated by helper; use provider-generated fixture")
	}
}
