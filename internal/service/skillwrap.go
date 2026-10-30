package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/futunn/futu/internal/client"
)

const (
	endpointTechnicalUnusual  = "skillwrap.technical_unusual"
	endpointFinancialUnusual  = "skillwrap.financial_unusual"
	endpointDerivativeUnusual = "skillwrap.derivative_unusual"

	defaultUnusualTimeRange  = 7
	defaultUnusualLanguageID = 0
)

// UnusualRequest is the SkillWrap POST body (OpenD / SkillWrapAPI.proto field names).
type UnusualRequest struct {
	StockSymbol        string
	TimeRange          int
	LanguageID         int
	IndicatorFilters   []string
	AnalysisDimensions []string
}

// UnusualResult is the CLI JSON shape for unusual commands (T07).
type UnusualResult struct {
	RetType     int    `json:"retType"`
	ErrCode     int    `json:"errCode"`
	RetMsg      string `json:"retMsg"`
	TimeRange   string `json:"time_range"`
	Content     string `json:"content"`
	Method      string `json:"method"`
	StockSymbol string `json:"stock_symbol,omitempty"`
}

type skillWrapHTTPResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	TimeRange string `json:"time_range"`
	Content   string `json:"content"`
}

type skillWrapEnvelope struct {
	Status string              `json:"s"`
	Data   skillWrapHTTPResponse `json:"d"`
}

// SkillWrapService calls SkillWrap anomaly HTTP endpoints.
type SkillWrapService struct {
	client *client.Client
}

// NewSkillWrapService constructs a SkillWrapService.
func NewSkillWrapService(c *client.Client) *SkillWrapService {
	return &SkillWrapService{client: c}
}

// GetTechnicalUnusual calls get_technical_unusual.
func (s *SkillWrapService) GetTechnicalUnusual(ctx context.Context, req UnusualRequest) (*UnusualResult, error) {
	return s.postUnusual(ctx, endpointTechnicalUnusual, "get_technical_unusual", req)
}

// GetFinancialUnusual calls get_financial_unusual.
func (s *SkillWrapService) GetFinancialUnusual(ctx context.Context, req UnusualRequest) (*UnusualResult, error) {
	return s.postUnusual(ctx, endpointFinancialUnusual, "get_financial_unusual", req)
}

// GetDerivativeUnusual calls get_derivative_unusual.
func (s *SkillWrapService) GetDerivativeUnusual(ctx context.Context, req UnusualRequest) (*UnusualResult, error) {
	return s.postUnusual(ctx, endpointDerivativeUnusual, "get_derivative_unusual", req)
}

func (s *SkillWrapService) postUnusual(ctx context.Context, endpoint, method string, req UnusualRequest) (*UnusualResult, error) {
	body := newSkillWrapBody(req)
	var raw json.RawMessage
	if err := s.client.Call(ctx, endpoint, client.CallOpts{Body: body}, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	parsed, err := parseSkillWrapResponse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	if parsed.Code != 0 {
		return nil, fmt.Errorf("%s: api code %d: %s", method, parsed.Code, parsed.Message)
	}
	return &UnusualResult{
		RetType:     0,
		ErrCode:     parsed.Code,
		RetMsg:      parsed.Message,
		TimeRange:   parsed.TimeRange,
		Content:     parsed.Content,
		Method:      method,
		StockSymbol: req.StockSymbol,
	}, nil
}

func newSkillWrapBody(req UnusualRequest) map[string]any {
	timeRange := req.TimeRange
	if timeRange <= 0 {
		timeRange = defaultUnusualTimeRange
	}
	languageID := req.LanguageID
	body := map[string]any{
		"stock_symbol": req.StockSymbol,
		"time_range":   timeRange,
		"language_id":  languageID,
	}
	if len(req.IndicatorFilters) > 0 {
		body["indicator_filters"] = req.IndicatorFilters
	}
	if len(req.AnalysisDimensions) > 0 {
		body["analysis_dimensions"] = req.AnalysisDimensions
	}
	return body
}

func parseSkillWrapResponse(raw json.RawMessage) (*skillWrapHTTPResponse, error) {
	var flat skillWrapHTTPResponse
	if err := json.Unmarshal(raw, &flat); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if flat.Code != 0 || flat.Content != "" || flat.TimeRange != "" || flat.Message != "" {
		return &flat, nil
	}
	var wrapped skillWrapEnvelope
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return &flat, nil
	}
	if wrapped.Status != "" {
		return &wrapped.Data, nil
	}
	return &flat, nil
}
