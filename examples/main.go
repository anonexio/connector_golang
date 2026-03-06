package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/anonex/anonex-go/anonex"
)

func main() {
	// ========================================
	//  Public REST API
	// ========================================

	client := anonex.NewClient()

	info, _ := client.GetInfo()
	fmt.Println("Info:", string(info))

	serverTime, _ := client.GetTime()
	fmt.Println("Time:", string(serverTime))

	assets, _ := client.GetAssets(map[string]string{"limit": "5"})
	fmt.Println("Assets:", string(assets)[:100], "...")

	btc, _ := client.GetAssetInfo(map[string]string{"ticker": "BTC"})
	fmt.Println("BTC:", string(btc)[:100], "...")

	markets, _ := client.GetMarketList()
	fmt.Println("Markets:", string(markets)[:100], "...")

	ticker, _ := client.GetTicker("BTC_USDT")
	fmt.Println("Ticker:", string(ticker))

	ob, _ := client.GetMarketOrderbook(map[string]string{"symbol": "BTC/USDT", "limit": "5"})
	fmt.Println("Orderbook:", string(ob)[:100], "...")

	// ========================================
	//  Authenticated REST API
	// ========================================

	// authClient := anonex.NewClient(anonex.WithAPIKey("KEY", "SECRET"))
	// balances, _ := authClient.GetBalances(map[string]string{"tickerlist": "BTC,ETH"})
	// order, _ := authClient.CreateOrder(map[string]interface{}{
	//     "symbol": "BTC/USDT", "side": "buy", "type": "limit",
	//     "quantity": "0.001", "price": "50000",
	// })

	// ========================================
	//  WebSocket
	// ========================================

	ws := anonex.NewWebSocketClient()

	ws.On("connected", func(msg anonex.WSMessage) {
		fmt.Println("WS Connected!")
		ws.SubscribeTicker("BTC/USDT", nil)
	})

	ws.On("ticker", func(msg anonex.WSMessage) {
		data, _ := json.Marshal(msg)
		fmt.Println("Ticker:", string(data)[:100])
	})

	ws.On("disconnected", func(msg anonex.WSMessage) {
		fmt.Println("WS Disconnected")
	})

	if err := ws.Connect(); err != nil {
		fmt.Println("WS Error:", err)
		return
	}

	time.Sleep(10 * time.Second)
	ws.Disconnect()

	fmt.Println("Done!")
}
