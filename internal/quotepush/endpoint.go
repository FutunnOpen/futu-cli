package quotepush

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// QuoteEndpoint maps an API base URL to the quote-push WebSocket endpoint.
// https://webapi.futunn.com becomes wss://webapi-quote.futunn.com/ws.
func QuoteEndpoint(apiBase string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(apiBase))
	if err != nil || parsed.Hostname() == "" {
		return "", fmt.Errorf("invalid api_base %q", apiBase)
	}
	host := parsed.Hostname()
	switch {
	case strings.HasPrefix(host, "webapi-quote."):
	case strings.HasPrefix(host, "webapi."):
		host = "webapi-quote." + strings.TrimPrefix(host, "webapi.")
	default:
		return "", fmt.Errorf("cannot derive quote push endpoint from api_base %q", apiBase)
	}
	if port := parsed.Port(); port != "" {
		host = net.JoinHostPort(host, port)
	}
	return "wss://" + host + "/ws", nil
}
