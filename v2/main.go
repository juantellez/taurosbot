package main // no longer just balances, but a new version of bot

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	tau "git.vmo.mx/Tauros/tradingbot/taurosapi"
	dec "github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pb "git.vmo.mx/Tauros/tradingbot/proto"
)

var grpcGdaxConn *grpc.ClientConn
var grpcOxConn *grpc.ClientConn
var getTicker = pb.NewTickerServiceClient(grpcGdaxConn)
var getSpreadPrice = pb.NewSpreadPriceServiceClient(grpcGdaxConn)
var getOxRate = pb.NewOxServiceClient(grpcOxConn)
var coins = []string{}

// ExchangeRate - keeps the current exchange rate
type ExchangeRate struct {
	*sync.RWMutex
	Rate float64 //I don't think decimal is necessary but maybe reconsider later
}

func (e *ExchangeRate) set(newRate float64) {
	//fatal error check for zero?
	e.Lock()
	e.Rate = newRate
	e.Unlock()
}

func (e *ExchangeRate) get() float64 {
	e.RLock()
	defer e.RUnlock()
	return e.Rate
}

var exchangeRate = &ExchangeRate{new(sync.RWMutex), 0.0}

func getMXNRate() float64 {
	res, err := getOxRate.GetOxRate(context.Background(), &pb.OxRequest{Currency: "MXN"})
	if err != nil {
		log.Fatalf("Unable to get exchange rate from ox grpc service: %v", err)
	}
	m, err := strconv.ParseFloat(res.Rate, 64)
	if err != nil {
		log.Errorf("Bad Rate %s unable to convert to float64: %v", res.Rate, err)
	}
	log.Infof("*** MXN exchange rate %f", m)
	return m
}

// exchangeRater - continually update the exchange rate.
func exchangeRater(quit chan bool, interval time.Duration) {
	exchangeRate.set(getMXNRate())
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			exchangeRate.set(getMXNRate())
		case <-quit:
			ticker.Stop()
			log.Info("stopping exchange rate service")
			wg.Done()
		}
	}
}

// Balances - type
type Balances struct {
	*sync.RWMutex
	balance map[string]dec.Decimal
}

// bal.update - update the balance of account for coin with amount
func (b *Balances) update(account string, coin string, amount string) {
	log.Infof("updating balance of %s coin %s with amount %s", account, coin, amount)
	key := account + coin
	a, _ := dec.NewFromString(amount)
	b.Lock()
	defer b.Unlock()
	if bal, exists := b.balance[key]; !exists {
		b.balance[key] = a
	} else {
		b.balance[key] = bal.Add(a)
	}
}

// bal.list - output the balances to screen
func (b *Balances) list() {
	b.RLock()
	defer b.RUnlock()
	for k := range bal.balance {
		log.Printf("balance[%s] = %s", k, bal.balance[k].String())
	}
	log.Print("============================")
}

//bal.json() - return json of all balances
func (b *Balances) json() []byte {
	b.RLock()
	defer b.RUnlock()
	type bal struct {
		Coin   string `json:"coin"`
		Amount string `json:"amount"`
	}
	type acc struct {
		Account  string `json:"account"`
		Balances []bal  `json:"balances"`
	}
	var jsonBalances []acc
	for _, a := range apiTokens {
		//log.Printf("account=%s", a)
		var b1 []bal
		for _, c := range coins {
			b1 = append(b1, bal{c, b.balance[a.Account+c].String()})
		}
		jsonBalances = append(jsonBalances, acc{a.Account, b1})
	}
	//log.Printf("jsonBalances=%+v", jsonBalances)
	j, _ := json.MarshalIndent(jsonBalances, "   ", " ")
	return j
}

func (b Balances) available(account string, coin string) dec.Decimal {
	b.RLock()
	defer b.RUnlock()
	return b.balance[account+coin]
}

var bal = &Balances{new(sync.RWMutex), make(map[string]dec.Decimal)}
var wg sync.WaitGroup

// Bot - data of one bot.
type Bot struct {
	ID           int64     `json:"id"`
	Account      string    `json:"account"`
	Market       string    `json:"market"`
	Side         string    `json:"side"`          //"buy" or "sell"
	TickerSource string    `json:"ticker_source"` //not yet used, normally "gdax"
	Spread       int64     `json:"spread"`
	Pct          float64   `json:"pct"`           //percentage of total available balance destined for orders.
	OrderID      int64     `json:"order_id"`      //current order id placed by this bot
	Price        string    `json:"price"`         //current price of this bot's order
	Amount       string    `json:"amount"`        //current amount of this bot's order
	ErrorMsg     string    `json:"error_message"` //last current error message
	Active       bool      `json:"active"`        //is the bot active or not
	MinInterval  int       `json:"min_interval"`  //mininum interval in ms before changing order
	MaxInterval  int       `json:"max_interval"`  //maximum interval in ms before changing order
	Bias         float64   `json:"bias"`          //how much should the price be biased toward buy <-> sell
	MinVariance  float64   `json:"min_variance"`  //how much the price has to change before changing the order
	Quit         chan bool `json:"-"`             // channel to notify the bot to quit
}

// BotUpdate - parts of the bot that can be updated on the fly, otherwise delete the bot and add new
type BotUpdate struct {
	ID          int64   `json:"id"` //required
	Spread      int64   `json:"spread"`
	Pct         float64 `json:"pct"`
	MinInterval int     `json:"min_interval"`
	MaxInterval int     `json:"max_interval"`
	Bias        float64 `json:"bias"`
	MinVariance float64 `json:"min_variance"`
}

// Bots - type
type Bots struct {
	*sync.RWMutex
	lastID int64
	bots   map[int64]*Bot
}

// bots.add(Bot) - add one bot and start it
func (b *Bots) add(newBot Bot) int64 {
	b.Lock()
	newBot.ID = b.lastID
	newBot.Quit = make(chan bool)
	log.Infof("Adding bot %+v", newBot)
	b.bots[b.lastID] = &newBot
	b.lastID++
	b.Unlock()
	if newBot.Active {
		wg.Add(1)
		go b.run(newBot.ID, newBot.Quit)
	}
	return newBot.ID
}

// bots.delete(ID) - delete one bot
func (b *Bots) delete(ID int64) {
	log.Infof("deleting bot ID %d", ID)
	b.Lock()
	defer b.Unlock()
	b.bots[ID].Quit <- true
	delete(b.bots, ID)
}

// bots.getJSON(ID) returns json of bot ID
func (b *Bots) getJSON(ID int64) []byte {
	b.RLock()
	defer b.RUnlock()
	j, _ := json.MarshalIndent(b.bots[ID], "   ", " ")
	return j
}

func (b *Bots) getJSONAll() []byte {
	b.RLock()
	defer b.RUnlock()
	j, _ := json.MarshalIndent(b.bots, "   ", " ")
	return j
}

// bots.deactivate(ID) - deactivate one bot
func (b Bots) deactivate(ID int64) {
	log.Infof("deactivating bot ID %d", ID)
	b.Lock()
	defer b.Unlock()
	b.bots[ID].Quit <- true
	b.bots[ID].Active = false
}

// bots.activate(ID) - activate one bot
func (b Bots) activate(ID int64) {
	log.Infof("activating bot ID %d", ID)
	b.Lock()
	defer b.Unlock()
	b.bots[ID].Active = true
	wg.Add(1)
	go b.run(ID, b.bots[ID].Quit) //not sure if this will work due to mutex not yet unlocked
}

func (b Bots) update(botUpdate BotUpdate) (err error) {
	b.Lock()
	defer b.Unlock()
	_, exists := b.bots[botUpdate.ID]
	if !exists {
		return fmt.Errorf("bot.update error, bot ID %d not found", botUpdate.ID)
	}
	//todo: check if reflect is better?
	if botUpdate.Spread > 0.0 {
		b.bots[botUpdate.ID].Spread = botUpdate.Spread
	}
	if botUpdate.Pct > 0.0 {
		b.bots[botUpdate.ID].Pct = botUpdate.Pct
	}
	if botUpdate.MinInterval > 0 {
		b.bots[botUpdate.ID].MinInterval = botUpdate.MinInterval
	}
	if botUpdate.MaxInterval > 0 {
		b.bots[botUpdate.ID].MaxInterval = botUpdate.MaxInterval
	}
	if botUpdate.Bias > 0.0 {
		b.bots[botUpdate.ID].Bias = botUpdate.Bias
	}
	if botUpdate.MinVariance > 0.0 {
		b.bots[botUpdate.ID].MinVariance = botUpdate.MinVariance
	}
	return nil
}

// bots.save - save all the bots to a json file.
func (b Bots) save() {
	var bots []Bot
	filename := "bots.json"
	b.RLock()
	defer b.RUnlock()
	for _, b := range b.bots {
		bots = append(bots, *b)
	}
	json, err := json.MarshalIndent(bots, " ", " ")
	if err != nil {
		log.Fatalf("Unable to marshal bots to json: %v", err)
	}
	if err := ioutil.WriteFile(filename, json, 0644); err != nil {
		log.Fatalf("Unable to save bots to file %s: %$v", filename, err)
	}
	log.Infof("bots saved to file %s", filename)
}

// bots.list list to log output
func (b Bots) list() {
	b.RLock()
	defer b.RUnlock()
	var s string
	for _, b := range b.bots {
		json, _ := json.MarshalIndent(b, "  ", "  ")
		s = s + "\n" + string(json)
	}
	if s == "" {
		s = "No bots are loaded..."
	}
	log.Printf("%s \n lastID=%d", s, b.lastID)
}

// bots.restore - restore all the bots from json file and start them
func (b *Bots) restore() {
	var bots []Bot
	filename := "bots.json"
	log.Infof("Restoring bots previously saved in %s", filename)
	in, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Warnf("no bots.json file found, starting afresh with no saved bots")
		return
	}
	if err := json.Unmarshal(in, &bots); err != nil {
		log.Infof("Bad bots.json file: %v, starting afresh with no saved bots")
		return
	}
	for _, newBot := range bots {
		b.add(newBot)
	}
}

func getGdaxTicker(market string) (maxBid, minAsk, price dec.Decimal) {
	//convert mxn market to usd for gdax
	m := strings.Split(market, "-")
	if m[1] == "MXN" {
		market = m[0] + "-" + "USD"
	}
	res, err := getTicker.GetTicker(context.Background(), &pb.TickerRequest{Market: market})
	if err != nil {
		log.Fatalf("Unable to get ticker from gdax grpc service: %v", err)
	}
	var mb, ma dec.Decimal
	mb, err = dec.NewFromString(res.MaxBid)
	if err != nil { //todo: eliminate checking this once we are sure it is working
		log.Fatalf("Bad Ticker MaxBid, unable %s to convert to decimal: %v", res.MaxBid, err)
	}
	ma, err = dec.NewFromString(res.MinAsk)
	if err != nil {
		log.Fatalf("Bad Ticker MinAsk, unable to convert %s to decimal:%v", res.MinAsk, err)
	}
	return mb, ma, dec.Avg(mb, ma)
}

func getDepthPrice(market string, side string, depth int64) float64 { //todo: refactor all naming "spread" to "depth"
	m := strings.Split(market, "-")
	if m[1] == "MXN" {
		market = m[0] + "-" + "USD"
	}
	res, err := getSpreadPrice.GetSpreadPrice(context.Background(), &pb.SpreadPriceRequest{
		Market: market,
		Side:   side,
		Depth:  strconv.FormatInt(depth, 10),
	})
	if err != nil {
		log.Fatalf("Unable to get depth price from gdax grpc service: %v", err)
	}
	price, err := strconv.ParseFloat(res.Price, 64)
	if err != nil {
		log.Fatalf("Bad Depth Price, unable to convert %s to float64: %v", res.Price, err)
	}
	return price
}

func (b *Bots) run(ID int64, quit chan bool) { //the meaty part
	log.Infof("starting running bot %d", ID)
	b.RLock()
	minInt := b.bots[ID].MinInterval
	maxInt := b.bots[ID].MaxInterval
	b.RUnlock()
	ticker := time.NewTicker(time.Duration(minInt+rand.Intn(maxInt)) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			ticker.Stop()
			b.Lock()
			bot := b.bots[ID]
			m := strings.Split(bot.Market, "-")
			buySide := m[0]
			sellSide := m[1]
			_, _, marketPrice := getGdaxTicker(bot.Market)
			var available float64
			if bot.Side == "buy" {
				available, _ = bal.available(bot.Account, sellSide).Div(marketPrice).Float64()
			} else {
				available, _ = bal.available(bot.Account, buySide).Float64()
			}
			bot.Amount = fmt.Sprintf("%.8f", available*bot.Pct)
			//todo: set bias according to available balances and bot.Bias
			price := getDepthPrice(bot.Market, bot.Side, bot.Spread)
			if sellSide == "MXN" {
				price = price * exchangeRate.get()
			}
			bot.Price = fmt.Sprintf("%.8f", price)
			log.Infof("Bot %d order M=%s S=%s A=%s P=%s", bot.ID, bot.Market, bot.Side, bot.Amount, bot.Price)
			b.bots[ID] = bot
			b.Unlock()
			ticker = time.NewTicker(time.Duration(minInt+rand.Intn(maxInt)) * time.Millisecond)
		case <-quit:
			ticker.Stop()
			log.Infof("Stopping bot ID %d", ID)
			wg.Done()
		}
	}
}
func (b *Bots) stop() {
	b.RLock()
	for _, b := range b.bots {
		if b.Active {
			b.Quit <- true
		}
	}
	b.RUnlock()
}

var bots = &Bots{new(sync.RWMutex), 0, make(map[int64]*Bot)}

type apiToken struct {
	Account  string `json:"account"`
	APIToken string `json:"api_token"`
}

type credentials struct {
	IsStaging              bool       `json:"is_staging"`
	APITokens              []apiToken `json:"tauros_tokens"`
	OpenexchangeratesToken string     `json:"openexchangerates_token"`
	GdaxToken              string     `json:"gdax_token"`
	BaseWebhookURL         string     `json:"base_webhook_url"`
}

var apiTokens []apiToken
var isStaging bool
var baseWebhookURL string

func loadCredentialsFile(filename string) {
	log.Infof("Using credentials file: %s", filename)
	var creds credentials
	in, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Unable to load credentials file: %v", err)
	}
	if err := json.Unmarshal(in, &creds); err != nil {
		log.Fatalf("unable to unmarshal json file: %v", err)
	}
	isStaging = creds.IsStaging
	apiTokens = creds.APITokens
	baseWebhookURL = creds.BaseWebhookURL
}

//todo put logFormatter in lib
type logFormatter struct {
	TimestampFormat string
	LevelDesc       []string
}

func (f *logFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := fmt.Sprintf(entry.Time.Format(f.TimestampFormat))
	return []byte(fmt.Sprintf("%s %s %s\n", f.LevelDesc[entry.Level], timestamp, entry.Message)), nil
}

func main() {
	flag.Parse()

	logFormatter := new(logFormatter)
	logFormatter.TimestampFormat = "2006-01-02 15:04:05"
	logFormatter.LevelDesc = []string{"PANIC", "FATAL", "ERROR", "WARNI", "INFOR", "DEBUG", "TRACE"}
	log.SetFormatter(logFormatter)
	log.SetLevel(log.InfoLevel)
	//log.SetLevel(log.TraceLevel)

	loadCredentialsFile(flag.Arg(0))

	log.Info("Subscribing to gdax service at gdax:2222")
	grpcGdaxConn, err := grpc.Dial("localhost:2222", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect to GDAX grpc service at localhost:2222")
	}
	defer grpcGdaxConn.Close()
	getTicker = pb.NewTickerServiceClient(grpcGdaxConn)
	getSpreadPrice = pb.NewSpreadPriceServiceClient(grpcGdaxConn)

	log.Info("Subscribing to openexchange service at ox:2223")
	grpcOxConn, err := grpc.Dial("localhost:2223", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect to Ox grpc service at localhost:2223")
	}
	defer grpcOxConn.Close() //probably not needed
	getOxRate = pb.NewOxServiceClient(grpcOxConn)

	//start the exchangeRater service
	quitExchangeRater := make(chan bool)
	wg.Add(1)
	go exchangeRater(quitExchangeRater, time.Duration(5)*time.Minute)

	tau.Init(isStaging)

	// get a list of supported coins and sort it
	if c, err := tau.GetCoins(); err != nil {
		log.Fatalf("Unable to get available coins")
	} else {
		for _, coin := range c {
			coins = append(coins, coin.Coin)
		}
	}
	sort.Strings(coins)

	//get balances, remove all orders, remove current webhooks and add new webhooks
	for _, t := range apiTokens {
		log.Infof("closing all orders for %s", t.Account)
		if err := tau.CloseAllOrders(t.APIToken); err != nil {
			log.Fatalf("Error closing all orders: %v", err)
		}
		log.Infof("getting balances for %s", t.Account)
		if wallets, err := tau.GetBalances(t.APIToken); err != nil {
			log.Fatalf("Error getting balances: %v", err)
		} else {
			for _, w := range wallets {
				log.Tracef("Balance of %s:%s = %s", t.Account, w.Coin, string(w.Balances.Available))
				bal.update(t.Account, w.Coin, string(w.Balances.Available))
			}
		}

		log.Infof("deleting and recreating webhooks for %s using apitoken %s", t.Account, t.APIToken)
		if err := tau.DeleteWebhooks(t.APIToken); err != nil {
			log.Fatalf("Error deleting webhooks %v", err)
		}
		webhookID, err := tau.CreateWebhook(tau.Webhook{
			Name:              "Bot",
			Endpoint:          baseWebhookURL + "/webhooks/" + t.APIToken[4:10], //just use a part of apikey
			NotifyDeposit:     true,
			NotifyWithdrawal:  true,
			NotifyOrderPlaced: false, //we will internally keep track of balances
			NotifyOrderFilled: true,
			NotifyTrade:       true,
			IsActive:          true,
		}, t.APIToken)
		if err != nil {
			log.Errorf("Error creating webhook: %v", err)
		} else {
			log.Printf("Created webhook with id %d", webhookID)
		}
	}

	//start bots
	bots.restore()
	//bots.list()

	//start http server
	srv := &http.Server{
		Addr:         "0.0.0.0:9090",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}
	startRouter(srv)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Warnf("SIGTERM received, ending tauros trading bots...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	log.Infof("shutting down http server")
	srv.Shutdown(ctx)
	quitExchangeRater <- true
	bots.stop()
	bots.save()
}

/*
https://my.webhookrelay.com/v1/webhooks/55adaeff-f8f3-4449-aa83-c2c5ff335244/de8064
 export RELAY_KEY=234efedd-9036-48b7-b1c1-41c2d5f8b4ce
 export RELAY_SECRET=JhIdkzUNweFN
*/
