package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	dec "github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"

	tau "git.vmo.mx/Tauros/tradingbot/taurosapi"
)

// Order - an order placed in the exchange
type Order struct {
	ID     int64       `json:"id"`
	Market string      `json:"market"`
	Side   string      `json:"side"`
	Price  dec.Decimal `json:"price"`
	Amount dec.Decimal `json:"amount"`
}

// Orders - all the current orders placed in the exchange
type Orders struct {
	*sync.RWMutex
	Order map[int64]Order
}

var orders = &Orders{new(sync.RWMutex), make(map[int64]Order)}

func (o *Orders) add(market string, side string, price string, amount string, apiToken string) (int64, error) {
	o.Lock()
	defer o.Unlock()
	p, _ := dec.NewFromString(price)
	a, _ := dec.NewFromString(amount)
	orderInfo := fmt.Sprintf("%s s:%4s p:%s a:%s", time.Now().Format("2006-01-02 15:04:05"), side, price, amount)
	orderID, err := tau.PlaceOrder(tau.Message{
		Market: market,
		Amount: amount,
		Side:   side,
		Type:   "limit",
		Price:  price,
	}, apiToken)
	if err != nil {
		log.Errorf("Unable to place new order %s: %v", orderInfo, err)
		return 0, err
	}
	o.Order[orderID] = *&Order{
		ID:     orderID,
		Market: market,
		Side:   side,
		Price:  p,
		Amount: a,
	}
	return orderID, nil
}

// SortOrders - slice to sort the orders to find min bids and max asks made in the exchange
type SortOrders []Order

// Orders.json - get json of currently placed orders
func (o *Orders) json() []byte {
	o.RLock()
	defer o.RUnlock()
	var so SortOrders
	for _, o := range o.Order {
		so = append(so, o)
	}
	b, _ := json.MarshalIndent(so, "   ", " ")
	return b
}

func (o *Orders) delete(id int64, apiToken string) error {
	o.Lock()
	defer o.Unlock()
	if err := tau.CloseOrder(id, apiToken); err != nil {
		return err
	}
	delete(o.Order, id)
	//uddate balances

	return nil
}

func (o *Orders) list() {
	o.RLock()
	defer o.RUnlock()
	for id, o := range o.Order {
		log.Printf("ID %d: m: %s s: %s p: %s a: %s", id, o.Market, o.Side, o.Price, o.Amount)
	}
}

// Orders.sort() returns slice of all orders of one market and side ordered by price
func (o *Orders) sort(market, side string) SortOrders {
	o.RLock()
	defer o.RUnlock()
	var so SortOrders
	for _, o := range o.Order {
		if o.Market == market && o.Side == side {
			so = append(so, o)
		}
	}
	sort.Slice(so, func(i, j int) bool {
		if side == "sell" { //sort ascending
			return so[i].Price.LessThan(so[j].Price)
		} //sort descending
		return so[i].Price.GreaterThan(so[j].Price)
	})
	return so
}

func (o *Orders) getLowestBid(market, side string) Order {
	o.RLock()
	defer o.RUnlock()
	orders := o.sort(market, "sell")
	return orders[0]
}
