# AnonEx Go Connector

Official Go client for the AnonEx cryptocurrency exchange API.

## Installation

```bash
go get github.com/anonexio/connector_golang
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/anonexio/connector_golang"
)

func main() {
    // Public data
    client := anonex.NewClient()
    markets, _ := client.GetMarketList()
    fmt.Println(string(markets))

    // Authenticated
    client = anonex.NewClient(anonex.WithAPIKey("KEY", "SECRET"))
    balances, _ := client.GetBalances(nil)
    fmt.Println(string(balances))
}
```

## Authentication

```go
// HMAC-SHA256 (default)
client := anonex.NewClient(anonex.WithAPIKey("key", "secret"))

// Basic Auth
client := anonex.NewClient(anonex.WithAPIKey("key", "secret"), anonex.WithAuthMethod("basic"))
```

## REST API

All methods return `(json.RawMessage, error)`. Unmarshal into your own structs as needed.

### Public Endpoints

```go
client.GetInfo()
client.GetTime()
client.GetSummary()
client.GetAssets(map[string]string{"limit": "50"})
client.GetAssetInfo(map[string]string{"ticker": "BTC"})
client.GetAssetChart("BTC", map[string]string{"interval": "1D"})
client.GetMarketList()
client.GetMarketListFull()
client.GetMarketsPaginated(map[string]string{"base": "USDT"})
client.GetMarketInfo(map[string]string{"symbol": "BTC/USDT"})
client.GetCandles(map[string]string{"symbol": "BTC/USDT", "resolution": "60"})
client.GetMarketOrderbook(map[string]string{"symbol": "BTC/USDT", "limit": "50"})
client.GetMarketTrades(map[string]string{"symbol": "BTC/USDT"})
client.GetMarkets(nil)
client.GetPairs()
client.GetTicker("BTC_USDT")
client.GetTickers()
client.GetOrderbook(map[string]string{"ticker_id": "BTC_USDT"})
client.GetPoolList()
client.GetPoolInfo(map[string]string{"symbol": "BTC/USDT"})
client.GetPoolTickers()
client.GetAccountByAddress("ADDRESS")
```

### Private Endpoints

```go
client.GetBalances(map[string]string{"tickerlist": "BTC,ETH"})
client.GetTradingFees()
client.GetDepositAddress("BTC")
client.GetDeposits(map[string]string{"ticker": "BTC", "limit": "50"})
client.GetWithdrawals(map[string]string{"limit": "50"})
client.CreateWithdrawal(map[string]interface{}{"ticker": "BTC", "address": "bc1q...", "quantity": "0.01"})
client.CreateTransfer(map[string]interface{}{"ticker": "USDT", "accountid": "user@email.com", "quantity": "100"})
client.CreateOrder(map[string]interface{}{"symbol": "BTC/USDT", "side": "buy", "type": "limit", "quantity": "0.001", "price": "50000"})
client.CancelOrder(map[string]interface{}{"id": "ORDER_ID"})
client.CancelAllOrders(map[string]interface{}{"symbol": "BTC/USDT", "side": "all"})
client.GetOrder("ORDER_ID")
client.GetOrderWithTrades("ORDER_ID")
client.GetAccountOrders(map[string]string{"symbol": "BTC/USDT", "status": "active"})
client.GetAccountTrades(map[string]string{"symbol": "BTC/USDT"})
```

## WebSocket API

```go
ws := anonex.NewWebSocketClient()

ws.On("ticker", func(msg anonex.WSMessage) {
    fmt.Println("Ticker:", msg.Method)
})

ws.On("orderbook", func(msg anonex.WSMessage) {
    fmt.Println("Orderbook update")
})

ws.Connect()
ws.SubscribeTicker("BTC/USDT", nil)
ws.SubscribeOrderbook("BTC/USDT", 20, nil)
ws.SubscribeTrades("BTC/USDT", nil)
```

### Authenticated WebSocket

```go
ws := anonex.NewWebSocketClient(anonex.WithWSAuth("KEY", "SECRET"))

ws.On("reports", func(msg anonex.WSMessage) {
    fmt.Println("Order report:", msg)
})

ws.Connect()
ws.Login(func(msg anonex.WSMessage) {
    ws.SubscribeReports(nil)
    ws.NewOrder(map[string]interface{}{
        "symbol": "BTC/USDT", "side": "buy", "type": "limit",
        "quantity": "0.001", "price": "50000",
    }, nil)
})
```

### Events

`connected`, `disconnected`, `message`, `ticker`, `orderbook`, `trades`, `candles`, `reports`, `balances`, `transfers`, `pong`
