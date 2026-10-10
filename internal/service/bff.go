package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/futunn/futu/internal/client"
	apierr "github.com/futunn/futu/internal/errors"
)

type bffErrorBlock struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type bffEnvelope struct {
	RetCode    int             `json:"ret_code"`
	RetMsg     string          `json:"ret_msg"`
	Data       json.RawMessage `json:"data"`
	Pagination json.RawMessage `json:"pagination"`
	Error      *bffErrorBlock  `json:"error"`
	Skipped    json.RawMessage `json:"skipped"`
}

// BFFResult is a decoded BFF envelope payload.
type BFFResult struct {
	Data       json.RawMessage
	Pagination json.RawMessage
	Skipped    json.RawMessage
}

func (s *QuoteService) callBFF(ctx context.Context, endpoint string, opts client.CallOpts) (*BFFResult, error) {
	var env bffEnvelope
	if err := s.client.Call(ctx, endpoint, opts, &env); err != nil {
		return nil, err
	}
	var bffErr *apierr.BFFError
	if env.Error != nil {
		bffErr = &apierr.BFFError{Code: env.Error.Code, Message: env.Error.Message}
	}
	if err := apierr.FromBFF(env.RetCode, env.RetMsg, bffErr); err != nil {
		return nil, err
	}
	return &BFFResult{
		Data:       env.Data,
		Pagination: env.Pagination,
		Skipped:    env.Skipped,
	}, nil
}

func unmarshalBFFData[T any](raw json.RawMessage, dest *T) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, dest)
}

func checkCode(code int, message string) error {
	if code != 0 {
		return fmt.Errorf("API error (code %d): %s", code, message)
	}
	return nil
}
