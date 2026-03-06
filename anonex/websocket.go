package anonex

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage represents a JSON-RPC 2.0 message.
type WSMessage struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	Method  string          `json:"method"`
	Params  interface{}     `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	ID      int64           `json:"id,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// WSHandler is a callback for WebSocket messages.
type WSHandler func(WSMessage)

// WebSocketClient is the WebSocket client for AnonEx.
type WebSocketClient struct {
	APIKey           string
	APISecret        string
	WSURL            string
	Reconnect        bool
	ReconnectInterval time.Duration

	conn              *websocket.Conn
	msgID             int64
	handlers          map[string][]WSHandler
	responseHandlers  map[int64]WSHandler
	mu                sync.RWMutex
	done              chan struct{}
	shouldReconnect   bool
}

// NewWebSocketClient creates a new WebSocket client.
func NewWebSocketClient(opts ...WSOption) *WebSocketClient {
	ws := &WebSocketClient{
		WSURL:             "wss://api.anonex.io",
		Reconnect:         true,
		ReconnectInterval: 5 * time.Second,
		handlers:          make(map[string][]WSHandler),
		responseHandlers:  make(map[int64]WSHandler),
		done:              make(chan struct{}),
		shouldReconnect:   true,
	}
	for _, opt := range opts {
		opt(ws)
	}
	return ws
}

// WSOption configures the WebSocket client.
type WSOption func(*WebSocketClient)

func WithWSAuth(key, secret string) WSOption {
	return func(ws *WebSocketClient) { ws.APIKey = key; ws.APISecret = secret }
}
func WithWSURL(url string) WSOption {
	return func(ws *WebSocketClient) { ws.WSURL = url }
}

// On registers an event handler. Events: ticker, orderbook, trades, candles,
// reports, balances, transfers, connected, disconnected, error, message
func (ws *WebSocketClient) On(event string, handler WSHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.handlers[event] = append(ws.handlers[event], handler)
}

func (ws *WebSocketClient) emit(event string, msg WSMessage) {
	ws.mu.RLock()
	handlers := ws.handlers[event]
	ws.mu.RUnlock()
	for _, h := range handlers {
		go h(msg)
	}
}

func (ws *WebSocketClient) nextID() int64 {
	return atomic.AddInt64(&ws.msgID, 1)
}

// Connect connects to the WebSocket server.
func (ws *WebSocketClient) Connect() error {
	ws.shouldReconnect = true
	return ws.doConnect()
}

func (ws *WebSocketClient) doConnect() error {
	conn, _, err := websocket.DefaultDialer.Dial(ws.WSURL, nil)
	if err != nil {
		return fmt.Errorf("websocket connect failed: %w", err)
	}
	ws.conn = conn
	ws.emit("connected", WSMessage{})

	go ws.readLoop()
	return nil
}

func (ws *WebSocketClient) readLoop() {
	for {
		_, raw, err := ws.conn.ReadMessage()
		if err != nil {
			ws.emit("disconnected", WSMessage{})
			if ws.Reconnect && ws.shouldReconnect {
				time.Sleep(ws.ReconnectInterval)
				ws.doConnect()
			}
			return
		}

		var msg WSMessage
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}

		ws.emit("message", msg)

		// Response handlers
		if msg.ID > 0 {
			ws.mu.Lock()
			if h, ok := ws.responseHandlers[msg.ID]; ok {
				delete(ws.responseHandlers, msg.ID)
				ws.mu.Unlock()
				go h(msg)
			} else {
				ws.mu.Unlock()
			}
		}

		// Route by method
		switch msg.Method {
		case "ticker":
			ws.emit("ticker", msg)
		case "snapshotOrderbook", "updateOrderbook":
			ws.emit("orderbook", msg)
		case "snapshotTrades", "updateTrades":
			ws.emit("trades", msg)
		case "snapshotCandles", "updateCandles":
			ws.emit("candles", msg)
		case "report":
			ws.emit("reports", msg)
		case "balancereport":
			ws.emit("balances", msg)
		case "transferreport":
			ws.emit("transfers", msg)
		case "pong":
			ws.emit("pong", msg)
		}
	}
}

// Disconnect closes the WebSocket connection.
func (ws *WebSocketClient) Disconnect() {
	ws.shouldReconnect = false
	if ws.conn != nil {
		ws.conn.Close()
	}
}

// Send sends a JSON-RPC message. Returns the message ID.
func (ws *WebSocketClient) Send(method string, params interface{}, callback WSHandler) int64 {
	id := ws.nextID()
	msg := map[string]interface{}{"method": method, "params": params, "id": id}
	if callback != nil {
		ws.mu.Lock()
		ws.responseHandlers[id] = callback
		ws.mu.Unlock()
	}
	if ws.conn != nil {
		ws.conn.WriteJSON(msg)
	}
	return id
}

// Public methods
func (ws *WebSocketClient) Ping()                                        { ws.Send("ping", nil, nil) }
func (ws *WebSocketClient) SubscribeTicker(symbol string, cb WSHandler)  { ws.Send("subscribeTicker", map[string]string{"symbol": symbol}, cb) }
func (ws *WebSocketClient) SubscribeOnlyTickers(symbols []string, cb WSHandler) { ws.Send("subscribeOnlyTickers", map[string]interface{}{"symbols": symbols}, cb) }
func (ws *WebSocketClient) UnsubscribeTicker(symbol string)              { ws.Send("unsubscribeTicker", map[string]string{"symbol": symbol}, nil) }
func (ws *WebSocketClient) SubscribeOrderbook(symbol string, limit int, cb WSHandler) { ws.Send("subscribeOrderbook", map[string]interface{}{"symbol": symbol, "limit": limit}, cb) }
func (ws *WebSocketClient) UnsubscribeOrderbook(symbol string)           { ws.Send("unsubscribeOrderbook", map[string]string{"symbol": symbol}, nil) }
func (ws *WebSocketClient) SubscribeTrades(symbol string, cb WSHandler)  { ws.Send("subscribeTrades", map[string]string{"symbol": symbol}, cb) }
func (ws *WebSocketClient) UnsubscribeTrades(symbol string)              { ws.Send("unsubscribeTrades", map[string]string{"symbol": symbol}, nil) }
func (ws *WebSocketClient) SubscribeCandles(symbol string, period int, cb WSHandler) { ws.Send("subscribeCandles", map[string]interface{}{"symbol": symbol, "period": period}, cb) }
func (ws *WebSocketClient) UnsubscribeCandles(symbol string, period int) { ws.Send("unsubscribeCandles", map[string]interface{}{"symbol": symbol, "period": period}, nil) }
func (ws *WebSocketClient) GetAsset(ticker string, cb WSHandler)         { ws.Send("getAsset", map[string]string{"ticker": ticker}, cb) }
func (ws *WebSocketClient) GetAssets(cb WSHandler)                       { ws.Send("getAssets", nil, cb) }
func (ws *WebSocketClient) GetMarket(symbol string, cb WSHandler)        { ws.Send("getMarket", map[string]string{"symbol": symbol}, cb) }
func (ws *WebSocketClient) GetMarkets(cb WSHandler)                      { ws.Send("getMarkets", nil, cb) }

// Authenticated methods
func (ws *WebSocketClient) Login(cb WSHandler) {
	ws.Send("login", map[string]string{"algo": "BASIC", "pKey": ws.APIKey, "sKey": ws.APISecret}, cb)
}
func (ws *WebSocketClient) GetTradingBalance(cb WSHandler)         { ws.Send("getTradingBalance", nil, cb) }
func (ws *WebSocketClient) GetBalanceValues(cb WSHandler)          { ws.Send("getBalanceValues", nil, cb) }
func (ws *WebSocketClient) SubscribeReports(cb WSHandler)          { ws.Send("subscribeReports", nil, cb) }
func (ws *WebSocketClient) SubscribeSubAccountReports(cb WSHandler) { ws.Send("subscribeSubAccountReports", nil, cb) }
func (ws *WebSocketClient) SubscribeBalances(cb WSHandler)         { ws.Send("subscribeBalances", nil, cb) }
func (ws *WebSocketClient) SubscribeTransfers(cb WSHandler)        { ws.Send("subscribeTransfers", nil, cb) }
func (ws *WebSocketClient) NewOrder(params map[string]interface{}, cb WSHandler) { ws.Send("newOrder", params, cb) }
func (ws *WebSocketClient) NewTriggerOrder(params map[string]interface{}, cb WSHandler) { ws.Send("newTriggerOrder", params, cb) }
func (ws *WebSocketClient) CancelWSOrder(id, orderType string, cb WSHandler) { ws.Send("cancelOrder", map[string]string{"id": id, "type": orderType}, cb) }
func (ws *WebSocketClient) GetWSOrders(params map[string]string, cb WSHandler) { ws.Send("getOrders", params, cb) }
func (ws *WebSocketClient) GetWSTrades(params map[string]interface{}, cb WSHandler) { ws.Send("getTrades", params, cb) }
