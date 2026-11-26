package quotepush

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	defaultPingInterval   = 20 * time.Second
	defaultInitialBackoff = 200 * time.Millisecond
	defaultMaxBackoff     = 5 * time.Second
	defaultReadLimit      = 1 << 20
	authResponseTimeout   = 10 * time.Second
)

// Options controls one bounded quote-push session.
type Options struct {
	URL             string
	AccessToken     string
	Subscribe       Subscribe
	Duration        time.Duration
	MaxMessages     int
	PingInterval    time.Duration
	RefreshInterval time.Duration
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	Output          io.Writer
	Refresh         func(context.Context) (string, error)
	Dial            func(context.Context, string) (*websocket.Conn, error)
}

// Run connects, authenticates, subscribes, and writes each server message as one JSON line.
// The access token is never written to Output.
func Run(ctx context.Context, opt Options) error {
	if opt.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if opt.Output == nil {
		return fmt.Errorf("output is required")
	}
	if opt.URL == "" {
		return fmt.Errorf("quote push endpoint is required")
	}
	if opt.AccessToken == "" {
		return fmt.Errorf("access token is required")
	}
	ctx, cancel := context.WithTimeout(ctx, opt.Duration)
	defer cancel()

	secret := &secretBox{value: opt.AccessToken}
	backoff := opt.initialBackoff()
	var authed bool
	var lastErr error
	remaining := opt.MaxMessages
	for {
		if err := ctx.Err(); err != nil {
			return finish(authed, lastErr, err)
		}
		pushes, ready, err := runOnce(ctx, opt, secret, remaining)
		if ready {
			authed = true
		}
		if opt.MaxMessages > 0 {
			remaining -= pushes
			if remaining < 0 {
				remaining = 0
			}
		}
		if errors.Is(err, errLimitReached) {
			return nil
		}
		var fatal fatalError
		if errors.As(err, &fatal) {
			return fatal.err
		}
		if err != nil && ctx.Err() == nil {
			lastErr = err
		}
		if ctx.Err() != nil {
			return finish(authed, lastErr, ctx.Err())
		}
		if opt.MaxMessages > 0 && remaining == 0 {
			return nil
		}
		if waitErr := wait(ctx, backoff); waitErr != nil {
			return finish(authed, lastErr, waitErr)
		}
		backoff = nextBackoff(backoff, opt.maxBackoff())
	}
}

func finish(authed bool, lastErr, ctxErr error) error {
	if authed {
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("quote subscribe timed out: %w", ctxErr)
}

func runOnce(ctx context.Context, opt Options, secret *secretBox, remaining int) (int, bool, error) {
	conn, err := opt.dial(ctx)
	if err != nil {
		return 0, false, err
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	conn.SetReadLimit(defaultReadLimit)

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		writeMu sync.Mutex
		fatalMu sync.Mutex
		fatal   error
	)
	fail := func(err error) {
		fatalMu.Lock()
		if fatal == nil {
			fatal = err
		}
		fatalMu.Unlock()
		cancel()
	}
	loadFatal := func() error {
		fatalMu.Lock()
		defer fatalMu.Unlock()
		return fatal
	}

	if err := writeJSON(&writeMu, connCtx, conn, authMessage(secret.get())); err != nil {
		return 0, false, err
	}

	authed := false
	subscribed := false
	keepaliveStarted := false
	pushes := 0
	for {
		if err := loadFatal(); err != nil {
			return pushes, authed, fatalError{err: err}
		}
		readCtx := connCtx
		cancelRead := func() {}
		if !authed {
			var cancelAuth context.CancelFunc
			readCtx, cancelAuth = context.WithTimeout(connCtx, authResponseTimeout)
			cancelRead = cancelAuth
		}
		_, data, err := conn.Read(readCtx)
		cancelRead()
		if err != nil {
			if cause := loadFatal(); cause != nil {
				return pushes, authed, fatalError{err: cause}
			}
			if ctx.Err() != nil {
				return pushes, authed, ctx.Err()
			}
			return pushes, authed, err
		}
		safe, err := sanitizeMessage(data, secret.get())
		if err != nil {
			return pushes, authed, err
		}
		msg, err := parseInbound(safe)
		if err != nil {
			return pushes, authed, err
		}
		if msg.push() {
			if err := writeLine(opt.Output, safe); err != nil {
				return pushes, authed, err
			}
			if msg.countsTowardLimit(opt.Subscribe) {
				pushes++
				if remaining > 0 && pushes >= remaining {
					return pushes, true, errLimitReached
				}
			}
			continue
		}
		if msg.failed() {
			text := msg.Message
			if text == "" {
				text = "request rejected"
			}
			if !authed {
				return pushes, false, fatalError{err: fmt.Errorf("quote push authentication failed: %s", text)}
			}
			return pushes, true, fatalError{err: fmt.Errorf("quote subscribe rejected: %s", text)}
		}
		if err := writeLine(opt.Output, safe); err != nil {
			return pushes, authed, err
		}
		if msg.authenticated() {
			authed = true
			if !keepaliveStarted {
				keepaliveStarted = true
				go keepAlive(connCtx, conn, &writeMu, opt, secret, fail)
			}
			if !subscribed {
				if err := writeJSON(&writeMu, connCtx, conn, subscribeMessage(opt.Subscribe)); err != nil {
					return pushes, true, err
				}
				subscribed = true
			}
			continue
		}
	}
}

func keepAlive(ctx context.Context, conn *websocket.Conn, writeMu *sync.Mutex, opt Options, secret *secretBox, fail func(error)) {
	pingEvery := opt.pingInterval()
	ping := time.NewTicker(pingEvery)
	defer ping.Stop()
	var refreshC <-chan time.Time
	if opt.RefreshInterval > 0 && opt.Refresh != nil {
		refresh := time.NewTicker(opt.RefreshInterval)
		defer refresh.Stop()
		refreshC = refresh.C
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			writeMu.Lock()
			err := conn.Ping(ctx)
			writeMu.Unlock()
			if err != nil {
				return
			}
		case <-refreshC:
			next, err := opt.Refresh(ctx)
			if err != nil {
				fail(fmt.Errorf("refresh quote push token: %w", err))
				return
			}
			if next == "" {
				fail(fmt.Errorf("refresh quote push token: empty access token"))
				return
			}
			secret.set(next)
			if err := writeJSON(writeMu, ctx, conn, refreshMessage(next)); err != nil && ctx.Err() == nil {
				fail(err)
				return
			}
		}
	}
}

func writeJSON(mu *sync.Mutex, ctx context.Context, conn *websocket.Conn, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	return conn.Write(ctx, websocket.MessageText, data)
}

func writeLine(w io.Writer, payload []byte) error {
	if _, err := w.Write(append(append([]byte{}, payload...), '\n')); err != nil {
		return fmt.Errorf("write quote push: %w", err)
	}
	return nil
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (opt Options) dial(ctx context.Context) (*websocket.Conn, error) {
	if opt.Dial != nil {
		return opt.Dial(ctx, opt.URL)
	}
	conn, _, err := websocket.Dial(ctx, opt.URL, &websocket.DialOptions{
		CompressionMode: websocket.CompressionContextTakeover,
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (opt Options) pingInterval() time.Duration {
	if opt.PingInterval > 0 {
		return opt.PingInterval
	}
	return defaultPingInterval
}

func (opt Options) initialBackoff() time.Duration {
	if opt.InitialBackoff > 0 {
		return opt.InitialBackoff
	}
	return defaultInitialBackoff
}

func (opt Options) maxBackoff() time.Duration {
	if opt.MaxBackoff > 0 {
		return opt.MaxBackoff
	}
	return defaultMaxBackoff
}

func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}

var errLimitReached = errors.New("quote push message limit reached")

type fatalError struct {
	err error
}

func (e fatalError) Error() string {
	return e.err.Error()
}

func (e fatalError) Unwrap() error {
	return e.err
}

type secretBox struct {
	mu    sync.Mutex
	value string
}

func (s *secretBox) get() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value
}

func (s *secretBox) set(value string) {
	s.mu.Lock()
	s.value = value
	s.mu.Unlock()
}
