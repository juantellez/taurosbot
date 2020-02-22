package taurosapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// TauWsObject - Tauros Websocket message "object"
type TauWsObject struct {
	Amount         string `json:"amount"`
	AmountPaid     string `json:"amount_paid"`
	AmountReceived string `json:"amount_received"`
	ClosedAt       string `json:"closed_at"`
	CreatedAt      string `json:"created_at"`
	FeeAmountPaid  string `json:"fee_amount_paid"`
	FeeDecimal     string `json:"fee_decimal"`
	FeePercent     string `json:"fee_percent"`
	Filled         string `json:"filled"`
	ID             int64  `json:"id"`
	InitialAmount  string `json:"initial_amount"`
	InitialValue   string `json:"initial_value"`
	IsOpen         bool   `json:"is_open"`
	LeftCoin       string `json:"left_coin"`
	Market         string `json:"market"`
	Price          string `json:"price"`
	RightCoin      string `json:"right_coin"`
	Side           string `json:"side"`
	Value          string `json:"value"`
}

// TauWebHookObject - Taures Webhook message "object"
type TauWebHookObject struct {
	Market          string `json:"market"`
	Side            string `json:"side"`
	InitialAmount   string `json:"initial_amount"`
	Filled          string `json:"filled"`
	Value           string `json:"value"`
	InitialValue    string `json:"initial_value"`
	Price           string `json:"price"`
	FeeDecimal      string `json:"fee_decimal"` //todo: issue to correct too much data and overlapping names
	FeePercent      string `json:"fee_percent"`
	FeeAmountPaid   string `json:"fee_amount_paid"`
	IsOpen          bool   `json:"is_open"`
	AmountPaid      string `json:"amount_paid"`
	AmountReceived  string `json:"amount_received"`
	CreatedAt       string `json:"created_at"`
	ClosedAt        string `json:"closed_at"`
	LeftCoin        string `json:"left_coin"`
	RightCoin       string `json:"right_coin"`
	LeftCoinIcon    string `json:"left_coin_icon"`
	RightCoinIcon   string `json:"right_coin_icon"`
	Sender          string `json:"sender"`
	Receiver        string `json:"receiver"`
	Coin            string `json:"coin"`
	CoinName        string `json:"coin_name"`
	CoinIcon        string `json:"coin_icon"`
	Amount          string `json:"amount"`
	TxID            string `json:"txId"` //todo: github issue correcting json format to "tx_id"
	Confirmed       bool   `json:"confirmed"`
	ConfirmedAt     string `json:"confirmed_at"`
	IsInnerTransfer bool   `json:"is_innerTransfer"` //todo: issue to correct json name to is_inner_transfer
	Address         string `json:"address"`
	ExplorerLink    string `json:"explorer_link"`
	FeeAmount       string `json:"fee_amount"`
	TotalAmount     string `json:"total_amount"`
	Type            string `json:"type"`
	Description     string `json:"description"`
	ID              int64  `json:"id"`
}

// TauWsMessage - Tauros Websocket message header
type TauWsMessage struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Date        string      `json:"date"`
	Object      TauWsObject `json:"object"`
}

// TauWebHookMessage - Tauros POST message received via webhooks
type TauWebHookMessage struct { //todo: unify this with TauWsMessage
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Type        string           `json:"type"`
	Date        string           `json:"date"`
	Object      TauWebHookObject `json:"object"`
}

// Message - main message struct
type Message struct {
	ID       int64  `json:"id"`
	Market   string `json:"market"`
	Amount   string `json:"amount"`
	Side     string `json:"side"`
	Type     string `json:"type"`
	Price    string `json:"price"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Order - order message struct
type Order struct {
	ID            int64       `json:"order_id"`
	Market        string      `json:"market"`
	Side          string      `json:"side"`
	Amount        json.Number `json:"amount,Number"`
	InitialAmount json.Number `json:"initial_amount,Number"`
	Filled        json.Number `json:"filled,Number"`
	Value         json.Number `json:"value,Number"`
	InitialValue  json.Number `json:"initial_value,Number"`
	Price         json.Number `json:"price,Number"`
	CreatedAt     string      `json:"created_at"`
}

// Coin - available coins
type Coin struct {
	Coin                  string      `json:"coin"`
	MinWithdrawal         json.Number `json:"min_withdraw"`
	FeeWithdrawal         json.Number `json:"fee_withdraw"`
	ConfirmationsRequired int         `json:"confirmations_required"`
}

// Balance - available balances
type Balance struct {
	Coin     string `json:"coin"`
	CoinName string `json:"coin_name"`
	Address  string `json:"address"`
	Balances struct {
		Available json.Number `json:"available"`
		Pending   json.Number `json:"pending"`
		Frozen    json.Number `json:"frozen"`
	} `json:"balances"`
}

var apiURL = "https://api.tauros.io"

var apiToken string

// GetCoins - get all available coins handled by the exchange
func GetCoins() (coins []Coin, error error) {
	var c = []Coin{}
	var d struct {
		Crypto []Coin `json:"crypto"`
	}
	jsonData, err := doTauRequest(1, "GET", "data/coins", nil)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(jsonData, &d); err != nil {
		return c, err
	}
	return d.Crypto, nil
}

// GetBalances - get available balances of the user
func GetBalances() (balances []Balance, error error) {
	var b []Balance
	var w struct {
		Wallets []Balance `json:"wallets"`
	}
	jsonData, err := doTauRequest(1, "GET", "data/listbalances", nil)
	if err != nil {
		return b, err
	}
	if err := json.Unmarshal(jsonData, &w); err != nil {
		return b, err
	}
	return w.Wallets, nil
}

// GetDepositAddress - get the deposit address of the user for the specified coin
func GetDepositAddress(coin string) (address string, error error) {
	jsonData, err := doTauRequest(1, "GET", "data/getdepositaddress?coin="+coin, nil)
	if err != nil {
		return "", fmt.Errorf("TauDepositAddress-> %v", err)
	}
	var d struct {
		Coin    string `json:"coin"`
		Address string `json:"address"`
	}
	if err := json.Unmarshal(jsonData, &d); err != nil {
		return "", fmt.Errorf("TauDepositAddress-> %v", err)
	}
	return d.Address, nil
}

// PlaceOrder - add a new order
func PlaceOrder(order Message) (orderID int64, error error) {
	jsonData, err := doTauRequest(1, "POST", "trading/placeorder/", &order)
	if err != nil {
		return 0, fmt.Errorf("PlaceOrder-> %v", err)
	}
	var d struct {
		ID int64 `json:"id"`
	}
	//log.Tracef("jsonData=%s", string(jsonData))
	if err := json.Unmarshal(jsonData, &d); err != nil {
		return 0, fmt.Errorf("PlaceOrder-> unmarshaling jsonData %v", err)
	}
	//d.ID = rand.Int63n(10000000)
	log.Tracef("tauapi: add order %d", d.ID)
	return d.ID, nil
}

// GetOpenOrders - get all open orders by the user
func GetOpenOrders() (orders []Order, error error) {
	jsonData, err := doTauRequest(1, "GET", "trading/myopenorders/", nil)
	if err != nil {
		return nil, fmt.Errorf("GetOpenOrders->%v", err)
	}
	log.Tracef("jsonData=%s", string(jsonData))
	if err := json.Unmarshal(jsonData, &orders); err != nil {
		return nil, fmt.Errorf("GetOpenOrders->%v", err)
	}
	return orders, nil
}

// CloseAllOrders - close all currently open orders
func CloseAllOrders() error {
	log.Info("closing all orders...")
	orders, err := GetOpenOrders()
	if err != nil {
		return fmt.Errorf("CloseAllOrders ->%v", err)
	}
	for _, o := range orders {
		if err := CloseOrder(o.ID); err != nil {
			return fmt.Errorf("CloseAllOrders Deleting Order %d ->%v", o.ID, err)
		}
	}
	return nil
}

// CloseOrder - close the order specified by the order ID
func CloseOrder(orderID int64) error {
	var m Message
	m.ID = orderID
	log.Tracef("tauapi: del Order %d", orderID)
	_, err := doTauRequest(1, "POST", "trading/closeorder/", &m)
	if err != nil {
		return fmt.Errorf("CloseOrder->%v", err)
	}
	return nil
}

// Login - simulate a login to get the jwt token
func Login(email string, password string) (jwtToken string, err error) {
	var m Message
	m.Email = email
	m.Password = password
	jsonData, err := doTauRequest(2, "POST", "auth/signin/", &m)
	if err != nil {
		return "", fmt.Errorf("Login->%v", err)
	}
	var d struct {
		Token     string `json:"token"`
		TwoFactor bool   `json:"two_factor"`
	}
	if err := json.Unmarshal(jsonData, &d); err != nil {
		return "", fmt.Errorf("Login->%v", err)
	}
	return d.Token, nil
}

func doTauRequest(version int, reqType string, tauService string, message *Message) (msgdata json.RawMessage, error error) {
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("doTauRequest-> Error trying to json marshal tauMessage: %v", err)
	}
	log.Tracef("reqType: [%s], tauService: [%s] message: %+v", reqType, tauService, jsonMsg)
	var httpReq *http.Request
	var b []byte
	if reqType != "GET" {
		if b, err = json.Marshal(message); err != nil {
			return nil, fmt.Errorf("doTauRequest-> Error on body marshal: %v", err)
		}
	}
	log.Tracef("url=%s token=%s", apiURL+"/api/v1/"+tauService, apiToken)
	apiVersion := fmt.Sprintf("v%1d", version)
	httpReq, err = http.NewRequest(reqType, apiURL+"/api/"+apiVersion+"/"+tauService, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("doTauRequest-> Error on http.NewRequest: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Token "+apiToken)
	client := http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("doTauRequest-> Error reading response: %v", err)
	}
	defer resp.Body.Close()
	//todo: check StatusCode
	body, err := ioutil.ReadAll(resp.Body)
	log.Tracef("resp body=%s", string(body))
	if err != nil {
		return nil, fmt.Errorf("doTauRequest-> Error ioutil body: %v", err)
	}
	var respJSON struct {
		Success bool            `json:"success"`
		Message json.RawMessage `json:"msg"`
		Data    json.RawMessage `json:"data"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(body, &respJSON); err != nil {
		return nil, fmt.Errorf("doTauRequest-> Unmarshall error: %v", err)
	}
	if !respJSON.Success {
		msg := string(respJSON.Message)
		if msg == "" {
			msg = string(body)
		}
		if strings.Contains(msg, "Invalid token") {
			msg += " Token=" + apiToken
		}
		return nil, fmt.Errorf("doTauRequest-> Unsuccess message %s", msg)
	}
	if version == 1 {
		return respJSON.Data, err
	}
	return respJSON.Payload, err
}

//Init start the tauros api
func Init(testing bool, token string) {
	if testing {
		apiURL = "https://api.staging.tauros.io"
	}
	apiToken = token
}
