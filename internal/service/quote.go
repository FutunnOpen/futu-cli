package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/futunn/futu/internal/client"
)

// Endpoint names for quote API routes.
const (
	endpointStockQuote       = "quote.stock_quote"
	endpointMarketSnapshot   = "quote.market_snapshot"
	endpointCurKLine         = "quote.cur_kline"
	endpointHistoryKLine     = "quote.history_kline"
	endpointOrderBook        = "quote.order_book"
	endpointRTTicker         = "quote.rt_ticker"
	endpointRTData           = "quote.rt_data"
	endpointStockBasicInfo   = "quote.stock_basicinfo"
	endpointOptionChain      = "quote.option_chain"
	endpointOptionExpiration = "quote.option_expiration_date"
	endpointStockScreen      = "quote.stock_screen"
)

// QuoteService provides access to market data endpoints.
type QuoteService struct {
	client *client.Client
}

// NewQuoteService creates a new QuoteService backed by the given client.
func NewQuoteService(c *client.Client) *QuoteService {
	return &QuoteService{client: c}
}

// Call invokes a named BFF quote endpoint and returns the decoded envelope.
func (s *QuoteService) Call(ctx context.Context, endpoint string, opts client.CallOpts) (*BFFResult, error) {
	result, err := s.callBFF(ctx, endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", endpoint, err)
	}
	return result, nil
}

// CurKLineParams maps quote.cur_kline query parameters.
type CurKLineParams struct {
	Num          int
	KType        int
	AUType       int
	ExtendedTime int
	PredSide     int
}

// HistoryKLineParams maps quote.history_kline query parameters.
type HistoryKLineParams struct {
	Start        string
	End          string
	NextTime     int64
	KType        int
	AUType       int
	Num          int
	ExtendedTime int
	PredSide     int
}

// RTTickerParams maps quote.rt_ticker query parameters.
type RTTickerParams struct {
	Num     int
	Periods []string
}

// OptionChainParams maps quote.option_chain query parameters.
type OptionChainParams struct {
	Start           string
	End             string
	IndexOptionType int
	HasIndexType    bool
	FilterStandard  string
}

// --- Response types ---

// StockQuote represents a real-time quote from quote.stock_quote.
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

type stockQuoteData struct {
	QuoteList []StockQuote `json:"quote_list"`
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

// GetStockQuote returns real-time quotes for the given Futu-format codes.
func (s *QuoteService) GetStockQuote(ctx context.Context, codes []string) ([]StockQuote, error) {
	result, err := s.callBFF(ctx, endpointStockQuote, client.CallOpts{
		Body: map[string]any{"code_list": codes},
	})
	if err != nil {
		return nil, fmt.Errorf("get stock quote: %w", err)
	}
	var data stockQuoteData
	if err := unmarshalBFFData(result.Data, &data); err != nil {
		return nil, fmt.Errorf("decode stock quote: %w", err)
	}
	return data.QuoteList, nil
}

// GetMarketSnapshot returns market snapshots for the given Futu-format codes.
func (s *QuoteService) GetMarketSnapshot(ctx context.Context, codes []string) (*BFFResult, error) {
	result, err := s.callBFF(ctx, endpointMarketSnapshot, client.CallOpts{
		Body: map[string]any{"code_list": codes},
	})
	if err != nil {
		return nil, fmt.Errorf("get market snapshot: %w", err)
	}
	return result, nil
}

// GetCurKLine returns the latest N K-line bars for a symbol.
func (s *QuoteService) GetCurKLine(ctx context.Context, symbol string, params CurKLineParams) (*BFFResult, error) {
	query := url.Values{}
	query.Set("num", strconv.Itoa(params.Num))
	query.Set("ktype", strconv.Itoa(params.KType))
	query.Set("autype", strconv.Itoa(params.AUType))
	query.Set("extended_time", strconv.Itoa(params.ExtendedTime))
	query.Set("pred_side", strconv.Itoa(params.PredSide))
	result, err := s.callBFF(ctx, endpointCurKLine, client.CallOpts{
		PathParams: map[string]string{"symbol": symbol},
		Params:     query,
	})
	if err != nil {
		return nil, fmt.Errorf("get cur kline: %w", err)
	}
	return result, nil
}

// GetHistoryKLine returns historical K-line bars for a symbol.
func (s *QuoteService) GetHistoryKLine(ctx context.Context, symbol string, params HistoryKLineParams) (*BFFResult, error) {
	query := url.Values{}
	if len(params.Start) > 0 {
		query.Set("start", params.Start)
	}
	if len(params.End) > 0 {
		query.Set("end", params.End)
	}
	if params.NextTime != 0 {
		query.Set("next_time", strconv.FormatInt(params.NextTime, 10))
	}
	query.Set("ktype", strconv.Itoa(params.KType))
	query.Set("autype", strconv.Itoa(params.AUType))
	query.Set("num", strconv.Itoa(params.Num))
	query.Set("extended_time", strconv.Itoa(params.ExtendedTime))
	query.Set("pred_side", strconv.Itoa(params.PredSide))
	result, err := s.callBFF(ctx, endpointHistoryKLine, client.CallOpts{
		PathParams: map[string]string{"symbol": symbol},
		Params:     query,
	})
	if err != nil {
		return nil, fmt.Errorf("get history kline: %w", err)
	}
	return result, nil
}

// GetOrderBook returns the order book for a symbol.
func (s *QuoteService) GetOrderBook(ctx context.Context, code string, num int, numSet bool) (*BFFResult, error) {
	body := map[string]any{"code": code}
	if numSet {
		body["num"] = num
	}
	result, err := s.callBFF(ctx, endpointOrderBook, client.CallOpts{Body: body})
	if err != nil {
		return nil, fmt.Errorf("get order book: %w", err)
	}
	return result, nil
}

// GetRTTicker returns the latest tick trades for a symbol.
func (s *QuoteService) GetRTTicker(ctx context.Context, symbol string, params RTTickerParams) (*BFFResult, error) {
	query := url.Values{}
	query.Set("num", strconv.Itoa(params.Num))
	for _, period := range params.Periods {
		query.Add("period", period)
	}
	result, err := s.callBFF(ctx, endpointRTTicker, client.CallOpts{
		PathParams: map[string]string{"symbol": symbol},
		Params:     query,
	})
	if err != nil {
		return nil, fmt.Errorf("get rt ticker: %w", err)
	}
	return result, nil
}

// GetRTData returns intraday time-share data for a symbol.
func (s *QuoteService) GetRTData(ctx context.Context, symbol, section string) (*BFFResult, error) {
	query := url.Values{}
	query.Set("request_section", section)
	result, err := s.callBFF(ctx, endpointRTData, client.CallOpts{
		PathParams: map[string]string{"symbol": symbol},
		Params:     query,
	})
	if err != nil {
		return nil, fmt.Errorf("get rt data: %w", err)
	}
	return result, nil
}

// GetStockBasicInfo returns static security info for the given codes.
func (s *QuoteService) GetStockBasicInfo(ctx context.Context, codes []string) (*BFFResult, error) {
	result, err := s.callBFF(ctx, endpointStockBasicInfo, client.CallOpts{
		Body: map[string]any{"code_list": codes},
	})
	if err != nil {
		return nil, fmt.Errorf("get stock basicinfo: %w", err)
	}
	return result, nil
}

// GetOptionChain returns the option chain for an underlying symbol.
func (s *QuoteService) GetOptionChain(ctx context.Context, symbol string, params OptionChainParams) (*BFFResult, error) {
	query := url.Values{}
	if len(params.Start) > 0 {
		query.Set("start", params.Start)
	}
	if len(params.End) > 0 {
		query.Set("end", params.End)
	}
	if params.HasIndexType {
		query.Set("index_option_type", strconv.Itoa(params.IndexOptionType))
	}
	if len(params.FilterStandard) > 0 {
		query.Set("filter_standard", params.FilterStandard)
	}
	result, err := s.callBFF(ctx, endpointOptionChain, client.CallOpts{
		PathParams: map[string]string{"symbol": symbol},
		Params:     query,
	})
	if err != nil {
		return nil, fmt.Errorf("get option chain: %w", err)
	}
	return result, nil
}

// GetOptionExpirationDates returns future/past strike dates for an underlying.
func (s *QuoteService) GetOptionExpirationDates(ctx context.Context, symbol string, params OptionChainParams) ([]string, error) {
	query := url.Values{}
	if params.HasIndexType {
		query.Set("index_option_type", strconv.Itoa(params.IndexOptionType))
	}
	if len(params.FilterStandard) > 0 {
		query.Set("filter_standard", params.FilterStandard)
	}
	result, err := s.callBFF(ctx, endpointOptionExpiration, client.CallOpts{
		PathParams: map[string]string{"symbol": symbol},
		Params:     query,
	})
	if err != nil {
		return nil, fmt.Errorf("get option expiration: %w", err)
	}
	var payload struct {
		ExpirationList []struct {
			StrikeTime string `json:"strike_time"`
		} `json:"expiration_list"`
	}
	if err := unmarshalBFFData(result.Data, &payload); err != nil {
		return nil, fmt.Errorf("decode option expiration: %w", err)
	}
	dates := make([]string, 0, len(payload.ExpirationList))
	for _, item := range payload.ExpirationList {
		if item.StrikeTime != "" {
			dates = append(dates, item.StrikeTime)
		}
	}
	return dates, nil
}

const (
	rankPriceScale    = 1e9
	rankPercentScale  = 1e3
	rankTurnoverScale = 1e3

	rankPropPrice     = 2220
	rankPropChange    = 3102
	rankPropVolume    = 3104
	rankPropTurnover  = 3105
	rankPropAmplitude = 3103

	rankSortDesc = 2
	rankSortAsc  = 1
)

var rankMarketIDs = map[string]int{
	"HK": 1,
	"US": 2,
	"CN": 3,
}

// GetRank returns a market ranking list.
func (s *QuoteService) GetRank(ctx context.Context, rankType string, market string, count int) ([]RankItem, error) {
	marketID, ok := rankMarketIDs[strings.ToUpper(market)]
	if !ok {
		return nil, fmt.Errorf("unsupported market %q", market)
	}
	sortName, direction, err := rankSort(rankType)
	if err != nil {
		return nil, err
	}
	result, err := s.callBFF(ctx, endpointStockScreen, client.CallOpts{
		Body: rankScreenBody(marketID, count, sortName, direction),
	})
	if err != nil {
		return nil, fmt.Errorf("get rank: %w", err)
	}
	var payload struct {
		Items []screenRankItem `json:"items"`
	}
	if err := unmarshalBFFData(result.Data, &payload); err != nil {
		return nil, fmt.Errorf("decode rank: %w", err)
	}
	items := make([]RankItem, 0, len(payload.Items))
	for _, item := range payload.Items {
		items = append(items, item.toRankItem())
	}
	return items, nil
}

func rankSort(rankType string) (int, int, error) {
	switch rankType {
	case "volume":
		return rankPropVolume, rankSortDesc, nil
	case "turnover":
		return rankPropTurnover, rankSortDesc, nil
	case "gainers":
		return rankPropChange, rankSortDesc, nil
	case "losers":
		return rankPropChange, rankSortAsc, nil
	case "amplitude":
		return rankPropAmplitude, rankSortDesc, nil
	default:
		return 0, 0, fmt.Errorf("invalid rank type %q", rankType)
	}
}

func rankScreenBody(marketID, count, sortName, direction int) map[string]any {
	return map[string]any{
		"screen_queries": []any{
			map[string]any{
				"simple_field_query": map[string]any{
					"simple_field":      1,
					"screen_value_list": []int{marketID},
				},
			},
		},
		"retrieve_queries": []any{
			map[string]any{"simple_property": map[string]any{"name": rankPropPrice}},
			map[string]any{"cumulative_property": map[string]any{"name": rankPropChange, "days": 1}},
			map[string]any{"cumulative_property": map[string]any{"name": rankPropVolume, "days": 1}},
			map[string]any{"cumulative_property": map[string]any{"name": rankPropTurnover, "days": 1}},
		},
		"sort": map[string]any{
			"direction": direction,
			"cumulative_property": map[string]any{
				"name": sortName,
				"days": 1,
			},
		},
		"limit": count,
	}
}

type screenRankItem struct {
	Code    string            `json:"code"`
	Name    string            `json:"name"`
	SCName  string            `json:"sc_name"`
	Results []screenRankValue `json:"results"`
}

type screenRankValue struct {
	Simple     *screenPropertyResult `json:"simple_property_result"`
	Cumulative *screenPropertyResult `json:"cumulative_property_result"`
}

type screenPropertyResult struct {
	Property struct {
		Name int `json:"name"`
	} `json:"property"`
	Value screenNumber `json:"value"`
}

type screenNumber float64

func (n *screenNumber) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return err
		}
		*n = screenNumber(value)
		return nil
	}
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*n = screenNumber(value)
	return nil
}

func (item screenRankItem) toRankItem() RankItem {
	name := item.SCName
	if name == "" {
		name = item.Name
	}
	out := RankItem{Symbol: item.Code, Name: name}
	for _, result := range item.Results {
		prop := result.Simple
		if prop == nil {
			prop = result.Cumulative
		}
		if prop == nil {
			continue
		}
		value := float64(prop.Value)
		switch prop.Property.Name {
		case rankPropPrice:
			out.LastPrice = value / rankPriceScale
		case rankPropChange:
			out.ChangePct = value / rankPercentScale
		case rankPropVolume:
			out.Volume = int64(value)
		case rankPropTurnover:
			out.Turnover = value / rankTurnoverScale
		}
	}
	return out
}
