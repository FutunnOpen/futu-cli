package service

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/FutunnOpen/futu-cli/internal/client"
)

// Endpoint names for quote API routes.
const (
	endpointStockQuote  = "quote.stock_quote"
	endpointSnapshot    = "quote.snapshot"
	endpointKLine       = "quote.kline"
	endpointDepth       = "quote.depth"
	endpointTicker      = "quote.ticker"
	endpointRank        = "quote.rank"
	endpointOptionChain = "quote.option.chain"
)

// QuoteService provides access to market data endpoints.
type QuoteService struct {
	client *client.Client
}

// NewQuoteService creates a new QuoteService backed by the given client.
func NewQuoteService(c *client.Client) *QuoteService {
	return &QuoteService{client: c}
}

// --- Response types ---

// Snapshot represents a point-in-time quote for a single security.
type Snapshot struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	LastPrice float64 `json:"last_price"`
	Change    float64 `json:"change"`
	ChangePct float64 `json:"change_pct"`
	Volume    int64   `json:"volume"`
	Turnover  float64 `json:"turnover"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Open      float64 `json:"open"`
	PrevClose float64 `json:"prev_close"`
	Timestamp int64   `json:"timestamp"`
}

// KLine represents a single candlestick bar.
type KLine struct {
	Time     int64   `json:"time"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   int64   `json:"volume"`
	Turnover float64 `json:"turnover"`
}

// DepthEntry represents one price level in the order book.
type DepthEntry struct {
	Price  float64 `json:"price"`
	Volume int64   `json:"volume"`
	Order  int     `json:"order"`
}

// Depth represents the full order book for a symbol.
type Depth struct {
	Asks []DepthEntry `json:"asks"`
	Bids []DepthEntry `json:"bids"`
}

// Ticker represents a single tick (trade) record.
type Ticker struct {
	Time      int64   `json:"time"`
	Price     float64 `json:"price"`
	Volume    int64   `json:"volume"`
	Direction string  `json:"direction"` // "B"/"S"/"N"
}

// RankItem represents one entry in a market ranking list.
type RankItem struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	LastPrice float64 `json:"last_price"`
	ChangePct float64 `json:"change_pct"`
	Volume    int64   `json:"volume"`
	Turnover  float64 `json:"turnover"`
}

// OptionChainItem represents a single option contract in a chain.
type OptionChainItem struct {
	Symbol      string  `json:"symbol"`
	Name        string  `json:"name"`
	StrikePrice float64 `json:"strike_price"`
	ExpiryDate  string  `json:"expiry_date"`
	Type        string  `json:"type"` // "C"/"P"
	LastPrice   float64 `json:"last_price"`
	ImpliedVol  float64 `json:"implied_vol"`
	OpenInt     int64   `json:"open_interest"`
}

// StockQuote represents a real-time quote from the Futu stock-quote API.
type StockQuote struct {
	Code           string         `json:"code"`
	Name           string         `json:"name"`
	SCName         string         `json:"sc_name"`
	TCName         string         `json:"tc_name"`
	DataTime       int64          `json:"data_time"`
	DataDate       string         `json:"data_date"`
	LastPrice      float64        `json:"last_price"`
	OpenPrice      float64        `json:"open_price"`
	HighPrice      float64        `json:"high_price"`
	LowPrice       float64        `json:"low_price"`
	PrevClosePrice float64        `json:"prev_close_price"`
	Volume         int64          `json:"volume"`
	Turnover       float64        `json:"turnover"`
	TurnoverRate   float64        `json:"turnover_rate"`
	Amplitude      float64        `json:"amplitude"`
	SecStatus      string         `json:"sec_status"`
	Suspension     bool           `json:"suspension"`
	DarkStatus     string         `json:"dark_status"`
	ListingDate    string         `json:"listing_date"`
	OptionExData   *OptionExData  `json:"option_ex_data,omitempty"`
	FutureExData   *FutureExData  `json:"future_ex_data,omitempty"`
	PreMarket      *MarketSession `json:"pre_market,omitempty"`
	AfterMarket    *MarketSession `json:"after_market,omitempty"`
	Overnight      *MarketSession `json:"overnight,omitempty"`
}

// OptionExData contains extended data for option contracts.
type OptionExData struct {
	StrikePrice          float64 `json:"strike_price"`
	ContractSize         int64   `json:"contract_size"`
	OpenInterest         int64   `json:"open_interest"`
	ImpliedVolatility    float64 `json:"implied_volatility"`
	Premium              float64 `json:"premium"`
	Delta                float64 `json:"delta"`
	Gamma                float64 `json:"gamma"`
	Vega                 float64 `json:"vega"`
	Theta                float64 `json:"theta"`
	Rho                  float64 `json:"rho"`
	NetOpenInterest      int64   `json:"net_open_interest"`
	ContractNominalValue float64 `json:"contract_nominal_value"`
	OwnerLotMultiplier   int64   `json:"owner_lot_multiplier"`
	ContractMultiplier   int64   `json:"contract_multiplier"`
	OptionType           string  `json:"option_type"`
	IndexOptionType      int32   `json:"index_option_type"`
	ExpiryDateDistance   int64   `json:"expiry_date_distance"`
	OptionAreaType       string  `json:"option_area_type"`
}

// FutureExData contains extended data for futures contracts.
type FutureExData struct {
	LastSettlePrice float64 `json:"last_settle_price"`
	Position        int64   `json:"position"`
	PositionChange  int64   `json:"position_change"`
}

// MarketSession holds price/volume data for a trading session
// (pre-market, after-hours, or overnight).
type MarketSession struct {
	Price      float64 `json:"price"`
	HighPrice  float64 `json:"high_price"`
	LowPrice   float64 `json:"low_price"`
	Volume     int64   `json:"volume"`
	Turnover   float64 `json:"turnover"`
	ChangeVal  float64 `json:"change_val"`
	ChangeRate float64 `json:"change_rate"`
	Amplitude  float64 `json:"amplitude"`
}

// --- API response wrappers (one per endpoint) ---

type stockQuoteRequest struct {
	CodeList []string `json:"code_list"`
}

type stockQuoteResponse struct {
	RetCode int    `json:"ret_code"`
	RetMsg  string `json:"ret_msg"`
	Data    struct {
		QuoteList []StockQuote `json:"quote_list"`
	} `json:"data"`
}

type snapshotResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []Snapshot `json:"list"`
	} `json:"data"`
}

type klineResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []KLine `json:"list"`
	} `json:"data"`
}

type depthResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    Depth  `json:"data"`
}

type tickerResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []Ticker `json:"list"`
	} `json:"data"`
}

type rankResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []RankItem `json:"list"`
	} `json:"data"`
}

type optionChainResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []OptionChainItem `json:"list"`
	} `json:"data"`
}

// --- Service methods ---

// GetStockQuote returns real-time quotes for the given Futu-format codes
// (e.g. "HK.09988", "US.AAPL").
func (s *QuoteService) GetStockQuote(ctx context.Context, codes []string) ([]StockQuote, error) {
	body := stockQuoteRequest{CodeList: codes}
	var resp stockQuoteResponse
	if err := s.client.Call(ctx, endpointStockQuote, client.CallOpts{Body: body}, &resp); err != nil {
		return nil, fmt.Errorf("get stock quote: %w", err)
	}
	if err := checkRetCode(resp.RetCode, resp.RetMsg); err != nil {
		return nil, err
	}
	return resp.Data.QuoteList, nil
}

// GetSnapshot returns real-time snapshots for the given symbols.
func (s *QuoteService) GetSnapshot(ctx context.Context, symbols []string) ([]Snapshot, error) {
	params := url.Values{}
	params.Set("symbols", strings.Join(symbols, ","))

	var resp snapshotResponse
	if err := s.client.Call(ctx, endpointSnapshot, client.CallOpts{Params: params}, &resp); err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetKLine returns candlestick bars for a symbol.
func (s *QuoteService) GetKLine(ctx context.Context, symbol string, period string, count int) ([]KLine, error) {
	params := buildKLineParams(symbol, period, count)

	var resp klineResponse
	if err := s.client.Call(ctx, endpointKLine, client.CallOpts{Params: params}, &resp); err != nil {
		return nil, fmt.Errorf("get kline: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetDepth returns the order book for a symbol.
func (s *QuoteService) GetDepth(ctx context.Context, symbol string) (*Depth, error) {
	params := url.Values{}
	params.Set("symbol", symbol)

	var resp depthResponse
	if err := s.client.Call(ctx, endpointDepth, client.CallOpts{Params: params}, &resp); err != nil {
		return nil, fmt.Errorf("get depth: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetTicker returns tick-by-tick trades for a symbol.
func (s *QuoteService) GetTicker(ctx context.Context, symbol string, count int) ([]Ticker, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("count", strconv.Itoa(count))

	var resp tickerResponse
	if err := s.client.Call(ctx, endpointTicker, client.CallOpts{Params: params}, &resp); err != nil {
		return nil, fmt.Errorf("get ticker: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetRank returns a market ranking list.
func (s *QuoteService) GetRank(ctx context.Context, rankType string, market string, count int) ([]RankItem, error) {
	params := buildRankParams(rankType, market, count)

	var resp rankResponse
	if err := s.client.Call(ctx, endpointRank, client.CallOpts{Params: params}, &resp); err != nil {
		return nil, fmt.Errorf("get rank: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetOptionChain returns the option chain for an underlying symbol.
func (s *QuoteService) GetOptionChain(ctx context.Context, symbol string) ([]OptionChainItem, error) {
	params := url.Values{}
	params.Set("symbol", symbol)

	var resp optionChainResponse
	if err := s.client.Call(ctx, endpointOptionChain, client.CallOpts{Params: params}, &resp); err != nil {
		return nil, fmt.Errorf("get option chain: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// --- Helpers ---

// buildKLineParams constructs query parameters for the kline endpoint.
func buildKLineParams(symbol, period string, count int) url.Values {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("period", period)
	params.Set("count", strconv.Itoa(count))
	return params
}

// buildRankParams constructs query parameters for the rank endpoint.
func buildRankParams(rankType, market string, count int) url.Values {
	params := url.Values{}
	params.Set("rank_type", rankType)
	params.Set("market", market)
	params.Set("count", strconv.Itoa(count))
	return params
}

// checkCode validates the API response code and returns an error for non-zero codes.
func checkCode(code int, message string) error {
	if code != 0 {
		return fmt.Errorf("API error (code %d): %s", code, message)
	}
	return nil
}

// checkRetCode validates Futu API ret_code.
func checkRetCode(code int, msg string) error {
	if code != 0 {
		return fmt.Errorf("API error (ret_code %d): %s", code, msg)
	}
	return nil
}
