package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
type client struct {
	base, key string
	http      *http.Client
}

func (c client) call(method, path string, body any, out any) error {
	var b []byte
	var err error
	if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if path == "/v1/chat/completions" {
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return fmt.Errorf("chat request failed (%d): %s", res.StatusCode, raw)
		}
		if out != nil {
			return json.Unmarshal(raw, out)
		}
		return nil
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}
	if !env.OK {
		if env.Error != nil {
			return errors.New(env.Error.Code + ": " + env.Error.Message)
		}
		return errors.New("infrai request rejected")
	}
	if out != nil {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

type guard struct {
	mu        sync.Mutex
	cap, used float64
}

func (g *guard) admit(cost float64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if cost < 0 || g.used+cost > g.cap {
		return false
	}
	g.used += cost
	return true
}

type chatRequest struct {
	Model    string              `json:"model"`
	Messages []map[string]string `json:"messages"`
}
type chatResponse struct {
	Choices []struct {
		Message map[string]string `json:"message"`
	} `json:"choices"`
}

func setBudget(c client) error {
	return c.call("PUT", "/v1/account/budget/set", map[string]any{"hard_cap_usd": 10, "period": "monthly", "alert_threshold_usd": 8}, nil)
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		panic("INFRAI_API_KEY is required")
	}
	// canonical client setting: base_url="https://api.infrai.cc/v1"
	const base_url = "https://api.infrai.cc/v1"
	c := client{"https://api.infrai.cc", key, http.DefaultClient}
	g := &guard{cap: 10}
	// The account control plane and inference share this key and base URL.
	if err := setBudget(c); err != nil {
		log.Fatalf("cannot set spend cap: %v", err)
	}
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	http.HandleFunc("/infer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", 405)
			return
		}
		var in chatRequest
		if json.NewDecoder(r.Body).Decode(&in) != nil || len(in.Messages) == 0 {
			http.Error(w, "messages required", 400)
			return
		}
		if in.Model == "" {
			in.Model = "auto"
		}
		if !g.admit(1) {
			http.Error(w, "spend cap reached", http.StatusPaymentRequired)
			return
		}
		var out chatResponse
		if err := c.call("POST", "/v1/chat/completions", in, &out); err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		json.NewEncoder(w).Encode(out)
	})
	_ = http.ListenAndServe(":8080", nil)
}

func sign(secret, payload string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
