package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/auth"
	"github.com/futunn/futu/internal/quotepush"
)

const (
	defaultSubscribeDuration = 20 * time.Second
	maxSubscribeDuration     = 30 * time.Minute
	defaultMaxPushMessages   = 50
)

func init() {
	quoteCmd.AddCommand(newQuoteSubscribeCommand())
}

func newQuoteSubscribeCommand() *cobra.Command {
	var (
		symbols     string
		types       string
		duration    time.Duration
		maxMessages int
		klinePeriod string
		klineAdjust string
	)
	command := &cobra.Command{
		Use:   "subscribe",
		Short: "Subscribe to realtime quote pushes",
		Long: `Subscribe to realtime quote pushes for a bounded time.

The command uses the stored OAuth token and does not print it. Each server
message is one compact JSON line on stdout. Push messages have a type such as
QUOTE, ORDER_BOOK, TICKER, KLINE, or BROKER_QUEUE.

BROKER_QUEUE is included when order_book is subscribed for an eligible symbol.
It is not a separate --types value.`,
		Example: `  futu quote subscribe --symbols HK.00700 --types quote
  futu quote subscribe --symbols HK.00700,US.AAPL --types quote,order_book --duration 20s
  futu quote subscribe --symbols US.AAPL --types kline --kline-period 1m`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runQuoteSubscribe(cmd, symbols, types, duration, maxMessages, klinePeriod, klineAdjust)
		},
	}
	command.Flags().StringVar(&symbols, "symbols", "", "Comma-separated symbols, for example HK.00700,US.AAPL")
	command.Flags().StringVar(&types, "types", "quote", "Comma-separated types: quote, order_book, ticker, kline")
	command.Flags().DurationVar(&duration, "duration", defaultSubscribeDuration, "How long to listen before exiting (maximum 30m)")
	command.Flags().IntVar(&maxMessages, "max-messages", defaultMaxPushMessages, "Stop after this many subscribed pushes; MARKET_STATE is printed but not counted. 0 means no cap")
	command.Flags().StringVar(&klinePeriod, "kline-period", "1m", "K-line period used when types includes kline")
	command.Flags().StringVar(&klineAdjust, "kline-adjust", "none", "K-line adjustment used when types includes kline")
	_ = command.MarkFlagRequired("symbols")
	return command
}

func runQuoteSubscribe(cmd *cobra.Command, rawSymbols, rawTypes string, duration time.Duration, maxMessages int, period, adjust string) error {
	if duration <= 0 || duration > maxSubscribeDuration {
		return fmt.Errorf("duration must be greater than 0 and at most %s", maxSubscribeDuration)
	}
	if maxMessages < 0 {
		return fmt.Errorf("max-messages must be zero or positive")
	}
	symbols, err := parseFutuCodes(splitComma(rawSymbols))
	if err != nil {
		return err
	}
	sub, err := quotepush.NormalizeSubscribe(symbols, splitComma(rawTypes), period, adjust)
	if err != nil {
		return err
	}
	endpoint, err := quotepush.QuoteEndpoint(cfg.APIBase)
	if err != nil {
		return err
	}
	token, refresh, err := quotePushCredentials()
	if err != nil {
		return err
	}
	return quotepush.Run(cmd.Context(), quotepush.Options{
		URL:             endpoint,
		AccessToken:     token,
		Subscribe:       sub,
		Duration:        duration,
		MaxMessages:     maxMessages,
		RefreshInterval: 5 * time.Minute,
		Output:          os.Stdout,
		Refresh:         refresh,
	})
}

func quotePushCredentials() (string, func(context.Context) (string, error), error) {
	oauthBase, err := currentOAuthBaseURL()
	if err != nil {
		return "", nil, err
	}
	path := effectiveTokenFile()
	token, err := auth.EnsureValidWithBaseURL(oauthBase, cfg.ClientID, path)
	if err != nil {
		return "", nil, fmt.Errorf("authentication required: %w\nRun 'futu auth login' to authenticate", err)
	}
	refresh := func(context.Context) (string, error) {
		store, err := auth.LoadToken(path)
		if err != nil {
			return "", err
		}
		updated, err := auth.RefreshStoredTokenWithBaseURL(oauthBase, cfg.ClientID, path, store)
		if err != nil {
			return "", err
		}
		if httpClient != nil {
			httpClient.SetToken(updated.AccessToken)
		}
		return updated.AccessToken, nil
	}
	return token, refresh, nil
}

func splitComma(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
