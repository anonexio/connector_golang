// Package anonex provides a Go client for the AnonEx cryptocurrency exchange API.
package anonex

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is the REST API client for AnonEx.
type Client struct {
	APIKey     string
	APISecret  string
	BaseURL    string
	AuthMethod string // "hmac" or "basic"
	HTTPClient *http.Client
}

// NewClient creates a new AnonEx REST client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		BaseURL:    "https://api.anonex.io",
		AuthMethod: "hmac",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ClientOption configures the client.
type ClientOption func(*Client)

func WithAPIKey(key, secret string) ClientOption {
	return func(c *Client) { c.APIKey = key; c.APISecret = secret }
}
func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.BaseURL = url }
}
func WithAuthMethod(method string) ClientOption {
	return func(c *Client) { c.AuthMethod = method }
}

func (c *Client) signRequest(fullURL, body string) http.Header {
	nonce := strconv.FormatInt(time.Now().UnixMilli(), 10)
	message := c.APIKey + fullURL + body + nonce
	mac := hmac.New(sha256.New, []byte(c.APISecret))
	mac.Write([]byte(message))
	sig := hex.EncodeToString(mac.Sum(nil))
	h := http.Header{}
	h.Set("x-api-key", c.APIKey)
	h.Set("x-api-nonce", nonce)
	h.Set("x-api-sign", sig)
	return h
}

func (c *Client) request(method, path string, params map[string]string, data interface{}, auth bool) (json.RawMessage, error) {
	fullURL := c.BaseURL + path
	if len(params) > 0 {
		v := url.Values{}
		for k, val := range params {
			if val != "" {
				v.Set(k, val)
			}
		}
		if encoded := v.Encode(); encoded != "" {
			fullURL += "?" + encoded
		}
	}

	var bodyReader io.Reader
	bodyStr := ""
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		bodyStr = string(b)
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	if auth {
		if c.APIKey == "" || c.APISecret == "" {
			return nil, fmt.Errorf("API key and secret required")
		}
		if c.AuthMethod == "basic" {
			req.SetBasicAuth(c.APIKey, c.APISecret)
		} else {
			for k, vals := range c.signRequest(fullURL, bodyStr) {
				for _, val := range vals {
					req.Header.Set(k, val)
				}
			}
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for API error
	var errResp struct {
		Error *struct {
			Code        interface{} `json:"code"`
			Message     string      `json:"message"`
			Description string      `json:"description"`
		} `json:"error"`
	}
	if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != nil {
		return nil, fmt.Errorf("API error %v: %s - %s", errResp.Error.Code, errResp.Error.Message, errResp.Error.Description)
	}

	return json.RawMessage(respBody), nil
}

func (c *Client) get(path string, params map[string]string, auth bool) (json.RawMessage, error) {
	return c.request("GET", path, params, nil, auth)
}

func (c *Client) post(path string, data interface{}, auth bool) (json.RawMessage, error) {
	return c.request("POST", path, nil, data, auth)
}

// p is a helper to build param maps, filtering empty values.
func p(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] != "" && kv[i+1] != "0" {
			m[kv[i]] = kv[i+1]
		}
	}
	return m
}

// ========================
//  Public Endpoints
// ========================

func (c *Client) GetInfo() (json.RawMessage, error)    { return c.get("/api/v2/info", nil, false) }
func (c *Client) GetTime() (json.RawMessage, error)    { return c.get("/api/v2/time", nil, false) }
func (c *Client) GetSummary() (json.RawMessage, error) { return c.get("/api/v2/summary", nil, false) }

// Assets
func (c *Client) GetAssets(params map[string]string) (json.RawMessage, error)      { return c.get("/api/v2/asset/getlist", params, false) }
func (c *Client) GetAssetInfo(params map[string]string) (json.RawMessage, error)   { return c.get("/api/v2/asset/info", params, false) }
func (c *Client) GetAssetChart(ticker string, params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/asset/getsimplechart/"+ticker, params, false) }

// Markets
func (c *Client) GetMarketList() (json.RawMessage, error)     { return c.get("/api/v2/market/getlist", nil, false) }
func (c *Client) GetMarketListFull() (json.RawMessage, error) { return c.get("/api/v2/market/listfull", nil, false) }
func (c *Client) GetMarketsPaginated(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/market/list", params, false) }
func (c *Client) GetMarketInfo(params map[string]string) (json.RawMessage, error)  { return c.get("/api/v2/market/info", params, false) }
func (c *Client) GetCandles(params map[string]string) (json.RawMessage, error)     { return c.get("/api/v2/market/candles", params, false) }
func (c *Client) GetMarketOrderbook(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/market/orderbook", params, false) }
func (c *Client) GetMarketTrades(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/market/trades", params, false) }
func (c *Client) GetMarkets(params map[string]string) (json.RawMessage, error)     { return c.get("/api/v2/markets", params, false) }
func (c *Client) GetPairs() (json.RawMessage, error)   { return c.get("/api/v2/pairs", nil, false) }
func (c *Client) GetTicker(symbol string) (json.RawMessage, error) { return c.get("/api/v2/ticker/"+symbol, nil, false) }
func (c *Client) GetTickers() (json.RawMessage, error) { return c.get("/api/v2/tickers", nil, false) }
func (c *Client) GetOrderbook(params map[string]string) (json.RawMessage, error)   { return c.get("/api/v2/orderbook", params, false) }
func (c *Client) GetOrderSnapshot(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/orders/snapshot", params, false) }
func (c *Client) GetTrades(params map[string]string) (json.RawMessage, error)      { return c.get("/api/v2/trades", params, false) }

// Pools
func (c *Client) GetPoolList() (json.RawMessage, error)     { return c.get("/api/v2/pool/getlist", nil, false) }
func (c *Client) GetPoolListFull() (json.RawMessage, error) { return c.get("/api/v2/pool/listfull", nil, false) }
func (c *Client) GetPoolsPaginated(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/pool/list", params, false) }
func (c *Client) GetPoolInfo(params map[string]string) (json.RawMessage, error)    { return c.get("/api/v2/pool/info", params, false) }
func (c *Client) GetPoolTrades(params map[string]string) (json.RawMessage, error)  { return c.get("/api/v2/pool/trades", params, false) }
func (c *Client) GetPoolTickers() (json.RawMessage, error)  { return c.get("/api/v2/pooltickers", nil, false) }
func (c *Client) GetPoolTicker(symbol string) (json.RawMessage, error) { return c.get("/api/v2/poolticker/"+symbol, nil, false) }

// Misc
func (c *Client) GetAccountByAddress(addr string) (json.RawMessage, error) { return c.get("/api/v2/getaccountbyaddress/"+addr, nil, false) }

// ========================
//  Private Endpoints
// ========================

func (c *Client) GetBalances(params map[string]string) (json.RawMessage, error)  { return c.get("/api/v2/balances", params, true) }
func (c *Client) GetTradingFees() (json.RawMessage, error)    { return c.get("/api/v2/tradingfees", nil, true) }
func (c *Client) GetDepositAddress(ticker string) (json.RawMessage, error) { return c.get("/api/v2/getdepositaddress/"+ticker, nil, true) }
func (c *Client) GetDeposits(params map[string]string) (json.RawMessage, error)  { return c.get("/api/v2/getdeposits", params, true) }
func (c *Client) GetWithdrawals(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/getwithdrawals", params, true) }
func (c *Client) CreateWithdrawal(data map[string]interface{}) (json.RawMessage, error) { return c.post("/api/v2/createwithdrawal", data, true) }
func (c *Client) GetTransfers(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/gettransfers", params, true) }
func (c *Client) CreateTransfer(data map[string]interface{}) (json.RawMessage, error) { return c.post("/api/v2/createtransfer", data, true) }
func (c *Client) FindTransaction(id string) (json.RawMessage, error) { return c.get("/api/v2/findtransaction/"+id, nil, true) }
func (c *Client) CreateOrder(data map[string]interface{}) (json.RawMessage, error) { return c.post("/api/v2/createorder", data, true) }
func (c *Client) CreateTriggerOrder(data map[string]interface{}) (json.RawMessage, error) { return c.post("/api/v2/createtriggerorder", data, true) }
func (c *Client) CancelOrder(data map[string]interface{}) (json.RawMessage, error) { return c.post("/api/v2/cancelorder", data, true) }
func (c *Client) CancelAllOrders(data map[string]interface{}) (json.RawMessage, error) { return c.post("/api/v2/cancelallorders", data, true) }
func (c *Client) GetOrder(orderID string) (json.RawMessage, error) { return c.get("/api/v2/getorder/"+orderID, nil, true) }
func (c *Client) GetOrderWithTrades(orderID string) (json.RawMessage, error) { return c.get("/api/v2/getorderwithtrades/"+orderID, nil, true) }
func (c *Client) GetAccountOrders(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/account/orders", params, true) }
func (c *Client) GetOrders(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/getorders", params, true) }
func (c *Client) GetPoolLiquidity(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/account/pliquidity", params, true) }
func (c *Client) GetAccountTrades(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/account/trades", params, true) }
func (c *Client) GetMyTrades(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/gettrades", params, true) }
func (c *Client) GetTradesSince(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/gettradessince", params, true) }
func (c *Client) GetMyPoolTrades(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/getpooltrades", params, true) }
func (c *Client) GetPoolTradesSince(params map[string]string) (json.RawMessage, error) { return c.get("/api/v2/getpooltradessince", params, true) }
