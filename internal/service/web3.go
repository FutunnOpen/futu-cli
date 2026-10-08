package service

import (
	"context"
	"fmt"
	"net/url"

	"github.com/FutunnOpen/futu-cli/internal/client"
)

// Endpoint names for Web3 API routes.
const (
	endpointWalletBalance = "web3.wallet.balance"
	endpointSwapQuote     = "web3.swap.quote"
	endpointSwapExecute   = "web3.swap.execute"
	endpointTransfer      = "web3.transfer"
)

// Web3Service provides access to Web3 wallet and swap endpoints.
type Web3Service struct {
	client *client.Client
}

// NewWeb3Service creates a new Web3Service backed by the given client.
func NewWeb3Service(c *client.Client) *Web3Service {
	return &Web3Service{client: c}
}

// --- Response types ---

// TokenBalance represents a single token balance in a wallet.
type TokenBalance struct {
	Token    string  `json:"token"`
	Balance  float64 `json:"balance"`
	ValueUSD float64 `json:"value_usd"`
}

// SwapQuote represents a quoted price for a token swap.
type SwapQuote struct {
	FromToken   string  `json:"from_token"`
	ToToken     string  `json:"to_token"`
	FromAmt     float64 `json:"from_amount"`
	ToAmt       float64 `json:"to_amount"`
	Rate        float64 `json:"rate"`
	PriceImpact float64 `json:"price_impact"`
}

// SwapResult represents the outcome of an executed swap.
type SwapResult struct {
	TxHash    string  `json:"tx_hash"`
	Status    string  `json:"status"`
	FromToken string  `json:"from_token"`
	ToToken   string  `json:"to_token"`
	FromAmt   float64 `json:"from_amount"`
	ToAmt     float64 `json:"to_amount"`
}

// TransferResult represents the outcome of a token transfer.
type TransferResult struct {
	TxHash string `json:"tx_hash"`
	Status string `json:"status"`
}

// --- API response wrappers ---

type walletBalanceResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		List []TokenBalance `json:"list"`
	} `json:"data"`
}

type swapQuoteResponse struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    SwapQuote `json:"data"`
}

type swapExecuteResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    SwapResult `json:"data"`
}

type transferResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    TransferResult `json:"data"`
}

// --- Request bodies ---

type swapQuoteReq struct {
	FromToken string `json:"from_token"`
	ToToken   string `json:"to_token"`
	Amount    string `json:"amount"`
}

type swapExecuteReq struct {
	FromToken string `json:"from_token"`
	ToToken   string `json:"to_token"`
	Amount    string `json:"amount"`
}

type transferReq struct {
	To     string `json:"to"`
	Token  string `json:"token"`
	Amount string `json:"amount"`
}

// --- Service methods ---

// GetBalance returns all token balances for the given wallet address.
func (s *Web3Service) GetBalance(ctx context.Context, address string) ([]TokenBalance, error) {
	params := url.Values{}
	params.Set("address", address)

	var resp walletBalanceResponse
	if err := s.client.Call(ctx, endpointWalletBalance, client.CallOpts{Params: params}, &resp); err != nil {
		return nil, fmt.Errorf("get wallet balance: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return resp.Data.List, nil
}

// GetSwapQuote requests a price quote for swapping tokens.
func (s *Web3Service) GetSwapQuote(ctx context.Context, from, to, amount string) (*SwapQuote, error) {
	body := swapQuoteReq{
		FromToken: from,
		ToToken:   to,
		Amount:    amount,
	}

	var resp swapQuoteResponse
	if err := s.client.Call(ctx, endpointSwapQuote, client.CallOpts{Body: body}, &resp); err != nil {
		return nil, fmt.Errorf("get swap quote: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// ExecuteSwap executes a token swap with the given parameters.
func (s *Web3Service) ExecuteSwap(ctx context.Context, from, to, amount string) (*SwapResult, error) {
	body := swapExecuteReq{
		FromToken: from,
		ToToken:   to,
		Amount:    amount,
	}

	var resp swapExecuteResponse
	if err := s.client.Call(ctx, endpointSwapExecute, client.CallOpts{Body: body}, &resp); err != nil {
		return nil, fmt.Errorf("execute swap: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Transfer sends tokens to the specified address.
func (s *Web3Service) Transfer(ctx context.Context, toAddr, token, amount string) (*TransferResult, error) {
	body := transferReq{
		To:     toAddr,
		Token:  token,
		Amount: amount,
	}

	var resp transferResponse
	if err := s.client.Call(ctx, endpointTransfer, client.CallOpts{Body: body}, &resp); err != nil {
		return nil, fmt.Errorf("transfer: %w", err)
	}
	if err := checkCode(resp.Code, resp.Message); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}
