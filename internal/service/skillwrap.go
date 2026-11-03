package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/futunn/futu/internal/client"
	apierr "github.com/futunn/futu/internal/errors"
)

const (
	endpointTechnicalUnusual  = "skillwrap.technical_unusual"
	endpointFinancialUnusual  = "skillwrap.financial_unusual"
	endpointDerivativeUnusual = "skillwrap.derivative_unusual"

	defaultUnusualTimeRange = 7
	maxUnusualTimeRange     = 365
)

// UnusualRequest is the Open API unusual-summary request.
// Symbol is a path parameter; the remaining fields are the JSON body.
type UnusualRequest struct {
	StockSymbol        string
	TimeRange          int
	LanguageID         int
	IndicatorFilters   []string
	AnalysisDimensions []string
}

// UnusualResult is the CLI JSON shape for unusual commands.
type UnusualResult struct {
	RetType     int    `json:"retType"`
	ErrCode     int    `json:"errCode"`
	RetMsg      string `json:"retMsg"`
	TimeRange   string `json:"time_range"`
	Content     string `json:"content"`
	Method      string `json:"method"`
	StockSymbol string `json:"stock_symbol,omitempty"`
}

type skillWrapData struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	TimeRange string `json:"time_range"`
	Content   string `json:"content"`
}

// SkillWrapService calls the quote unusual-summary endpoints.
type SkillWrapService struct {
	client *client.Client
}

// NewSkillWrapService constructs a SkillWrapService.
func NewSkillWrapService(c *client.Client) *SkillWrapService {
	return &SkillWrapService{client: c}
}

// GetTechnicalUnusual calls POST /api/v1.0/quote/{symbol}/unusual/technical.
func (s *SkillWrapService) GetTechnicalUnusual(ctx context.Context, req UnusualRequest) (*UnusualResult, error) {
	return s.postUnusual(ctx, endpointTechnicalUnusual, "get_technical_unusual", req)
}

// GetFinancialUnusual calls POST /api/v1.0/quote/{symbol}/unusual/capital.
func (s *SkillWrapService) GetFinancialUnusual(ctx context.Context, req UnusualRequest) (*UnusualResult, error) {
	return s.postUnusual(ctx, endpointFinancialUnusual, "get_financial_unusual", req)
}

// GetDerivativeUnusual calls POST /api/v1.0/quote/{symbol}/unusual/derivatives.
func (s *SkillWrapService) GetDerivativeUnusual(ctx context.Context, req UnusualRequest) (*UnusualResult, error) {
	return s.postUnusual(ctx, endpointDerivativeUnusual, "get_derivative_unusual", req)
}

func (s *SkillWrapService) postUnusual(ctx context.Context, endpoint, method string, req UnusualRequest) (*UnusualResult, error) {
	symbol := strings.TrimSpace(req.StockSymbol)
	if symbol == "" {
		return nil, fmt.Errorf("%s: symbol is required", method)
	}
	body, err := newSkillWrapBody(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	var env bffEnvelope
	if err := s.client.Call(ctx, endpoint, client.CallOpts{
		Body:       body,
		PathParams: map[string]string{"symbol": symbol},
	}, &env); err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	var bffErr *apierr.BFFError
	if env.Error != nil {
		bffErr = &apierr.BFFError{Code: env.Error.Code, Message: env.Error.Message}
	}
	if err := apierr.FromBFF(env.RetCode, env.RetMsg, bffErr); err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	data, err := parseSkillWrapData(env.Data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	retMsg := data.Message
	if retMsg == "" {
		retMsg = env.RetMsg
	}
	return &UnusualResult{
		RetType:     env.RetCode,
		ErrCode:     data.Code,
		RetMsg:      retMsg,
		TimeRange:   data.TimeRange,
		Content:     data.Content,
		Method:      method,
		StockSymbol: symbol,
	}, nil
}

func newSkillWrapBody(req UnusualRequest) (map[string]any, error) {
	timeRange := req.TimeRange
	if timeRange <= 0 {
		timeRange = defaultUnusualTimeRange
	}
	if timeRange > maxUnusualTimeRange {
		return nil, fmt.Errorf("time_range must be between 1 and %d", maxUnusualTimeRange)
	}
	body := map[string]any{
		"time_range":  timeRange,
		"language_id": req.LanguageID,
	}
	if len(req.IndicatorFilters) > 0 {
		body["indicator_filters"] = req.IndicatorFilters
	}
	if len(req.AnalysisDimensions) > 0 {
		body["analysis_dimensions"] = req.AnalysisDimensions
	}
	return body, nil
}

func parseSkillWrapData(raw json.RawMessage) (*skillWrapData, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return &skillWrapData{}, nil
	}
	var data skillWrapData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &data, nil
}
