package quotepush

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestQuoteEndpoint(t *testing.T) {
	got, err := QuoteEndpoint("https://webapi.futunn.com")
	if err != nil {
		t.Fatal(err)
	}
	if got != "wss://webapi-quote.futunn.com/ws" {
		t.Fatalf("endpoint = %s", got)
	}
	got, err = QuoteEndpoint("https://webapi.moomoo.com/api/v1.0")
	if err != nil {
		t.Fatal(err)
	}
	if got != "wss://webapi-quote.moomoo.com/ws" {
		t.Fatalf("endpoint = %s", got)
	}
}

func TestNormalizeSubscribe(t *testing.T) {
	sub, err := NormalizeSubscribe([]string{"HK.00700"}, []string{"quote", "order_book", "quote"}, "1m", "none")
	if err != nil {
		t.Fatal(err)
	}
	frame := subscribeMessage(sub)
	if len(frame.Quote) != 1 || len(frame.OrderBook) != 1 || len(frame.Ticker) != 0 || len(frame.Kline) != 0 {
		t.Fatalf("frame = %+v", frame)
	}
	if _, err := NormalizeSubscribe(nil, []string{"quote"}, "", ""); err == nil {
		t.Fatal("expected symbol error")
	}
	if _, err := NormalizeSubscribe([]string{"US.AAPL"}, []string{"broker"}, "", ""); err == nil {
		t.Fatal("expected type error")
	}
}

func TestParseInboundLiveEnvelope(t *testing.T) {
	ack, err := parseInbound([]byte(`{"ret_code":0,"ret_msg":"success","data":{"code":0,"id":"sub-1","message":""}}`))
	if err != nil {
		t.Fatal(err)
	}
	if ack.failed() || ack.push() {
		t.Fatalf("ack = %+v", ack)
	}
	rejected, err := parseInbound([]byte(`{"ret_code":12,"ret_msg":"quota exceeded","data":{"code":12,"message":"quota exceeded"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !rejected.failed() || rejected.Message != "quota exceeded" {
		t.Fatalf("rejected = %+v", rejected)
	}
	legacy, err := parseInbound([]byte(`{"code":401,"message":"unauthorized"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !legacy.failed() || legacy.Message != "unauthorized" {
		t.Fatalf("legacy = %+v", legacy)
	}
	quote, err := parseInbound([]byte(`{"type":"QUOTE","symbol":"HK.00700"}`))
	if err != nil {
		t.Fatal(err)
	}
	sub := Subscribe{Types: []string{"quote", "order_book"}}
	if !quote.countsTowardLimit(sub) {
		t.Fatal("QUOTE should count")
	}
	state, err := parseInbound([]byte(`{"type":"MARKET_STATE","data":{"market":"HK"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if state.countsTowardLimit(sub) {
		t.Fatal("MARKET_STATE should not count")
	}
	broker, err := parseInbound([]byte(`{"type":"BROKER_QUEUE","symbol":"HK.00700"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !broker.countsTowardLimit(sub) {
		t.Fatal("BROKER_QUEUE should count with order_book")
	}
	if broker.countsTowardLimit(Subscribe{Types: []string{"quote"}}) {
		t.Fatal("BROKER_QUEUE should not count for quote only")
	}
}

func TestSanitizeMessageRemovesToken(t *testing.T) {
	raw := []byte(`{"authorization":"Bearer secret-token","type":"QUOTE","note":"secret-token"}`)
	out, err := sanitizeMessage(raw, "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("secret-token")) || bytes.Contains(out, []byte("authorization")) {
		t.Fatalf("secret leaked: %s", out)
	}
	if !bytes.Contains(out, []byte(`"type":"QUOTE"`)) {
		t.Fatalf("payload = %s", out)
	}
}

func TestRunPrintsPushWithoutToken(t *testing.T) {
	const token = "secret-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		ctx := r.Context()
		_, authFrame, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if !bytes.Contains(authFrame, []byte(token)) {
			return
		}
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"session_id":"s1","server_time":1}`))
		_, _, _ = conn.Read(ctx)
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"id":"sub-1","code":0,"message":""}`))
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"QUOTE","symbol":"HK.00700"}`))
		_, _, _ = conn.Read(ctx)
	}))
	defer srv.Close()

	var output bytes.Buffer
	err := Run(context.Background(), testOptions(srv, token, &output, 1))
	if err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if strings.Contains(text, token) {
		t.Fatalf("token leaked: %s", text)
	}
	if !strings.Contains(text, `"type":"QUOTE"`) {
		t.Fatalf("output = %s", text)
	}
}

func TestRunReconnectsAfterDrop(t *testing.T) {
	const token = "secret-token"
	var connects atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		ctx := r.Context()
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
		n := connects.Add(1)
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"session_id":"s1","server_time":1}`))
		if n == 1 {
			_ = conn.Close(websocket.StatusGoingAway, "drop")
			return
		}
		_, _, _ = conn.Read(ctx)
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"ORDER_BOOK","symbol":"HK.00700"}`))
		_, _, _ = conn.Read(ctx)
	}))
	defer srv.Close()

	var output bytes.Buffer
	err := Run(context.Background(), testOptions(srv, token, &output, 1))
	if err != nil {
		t.Fatal(err)
	}
	if connects.Load() < 2 {
		t.Fatalf("connects = %d", connects.Load())
	}
	if !strings.Contains(output.String(), `"type":"ORDER_BOOK"`) {
		t.Fatalf("output = %s", output.String())
	}
}

func TestRunAuthRejectionIsFatal(t *testing.T) {
	const token = "secret-token"
	var connects atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connects.Add(1)
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		ctx := r.Context()
		_, _, _ = conn.Read(ctx)
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"code":401,"message":"unauthorized"}`))
		_, _, _ = conn.Read(ctx)
	}))
	defer srv.Close()

	var output bytes.Buffer
	err := Run(context.Background(), testOptions(srv, token, &output, 5))
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(output.String(), token) {
		t.Fatalf("token leaked: %s", output.String())
	}
	if connects.Load() != 1 {
		t.Fatalf("connects = %d", connects.Load())
	}
}

func TestRunMarketStateDoesNotConsumeLimit(t *testing.T) {
	const token = "secret-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		ctx := r.Context()
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"session_id":"s1","server_time":1}`))
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"ret_code":0,"ret_msg":"success","data":{"code":0,"id":"sub-1","message":""}}`))
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"MARKET_STATE","data":{"market":"HK"}}`))
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"type":"QUOTE","symbol":"HK.00700"}`))
		_, _, _ = conn.Read(ctx)
	}))
	defer srv.Close()

	var output bytes.Buffer
	err := Run(context.Background(), testOptions(srv, token, &output, 1))
	if err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, `"type":"MARKET_STATE"`) || !strings.Contains(text, `"type":"QUOTE"`) {
		t.Fatalf("output = %s", text)
	}
}

func TestRunRejectsRetCode(t *testing.T) {
	const token = "secret-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		ctx := r.Context()
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"session_id":"s1","server_time":1}`))
		if _, _, err := conn.Read(ctx); err != nil {
			return
		}
		_ = conn.Write(ctx, websocket.MessageText, []byte(`{"ret_code":12,"ret_msg":"quota exceeded","data":{"code":12,"message":"quota exceeded"}}`))
		_, _, _ = conn.Read(ctx)
	}))
	defer srv.Close()

	var output bytes.Buffer
	err := Run(context.Background(), testOptions(srv, token, &output, 5))
	if err == nil || !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("error = %v", err)
	}
}

func testOptions(srv *httptest.Server, token string, output io.Writer, maxMessages int) Options {
	return Options{
		URL:            "ws" + strings.TrimPrefix(srv.URL, "http"),
		AccessToken:    token,
		Subscribe:      Subscribe{Symbols: []string{"HK.00700"}, Types: []string{"quote"}},
		Duration:       3 * time.Second,
		MaxMessages:    maxMessages,
		PingInterval:   time.Hour,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     20 * time.Millisecond,
		Output:         output,
	}
}
