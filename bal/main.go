package main // balances - maintain tauros balances

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
	"sync"
	"syscall"
	"time"

	tau "git.vmo.mx/Tauros/tradingbot/taurosapi"
	dec "github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

// Balances - type
type Balances struct {
	*sync.RWMutex
	balance map[string]dec.Decimal
}

// bal.update - update the balance for coin with amount
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

var bal = &Balances{new(sync.RWMutex), make(map[string]dec.Decimal)}
var wg sync.WaitGroup

// Bot - data of one bot.
type Bot struct {
	ID           int64     `json:"id"`
	Account      string    `json:"account"`
	Market       string    `json:"market"`
	Side         string    `json:"json"` //"buy" or "sell"
	TickerSource string    `json:"ticker_source"`
	Spread       int64     `json:"spread"`
	Pct          float32   `json:"pct"`           //percentage of total available balance destined for orders.
	OrderID      int64     `json:"order_id"`      //current order id placed by this bot
	Price        string    `json:"price"`         //current price of this bot's order
	Amount       string    `json:"amount"`        //current amount of this bot's order
	ErrorMsg     string    `json:"error_message"` //last current error message
	Active       bool      `json:"active"`        //is the bot active or not
	MinInterval  int       `json:"min_interval"`  //mininum interval in ms before changing order
	MaxInterval  int       `json:"max_interval"`  //maximum interval in ms before changing order
	Bias         float32   `json:"bias"`          //how much should the price be biased toward buy <-> sell
	MinVariance  float32   `json:"min_variance"`  //how much the price has to change before changing the order
	Quit         chan bool `json:"-"`             // channel to notify the bot to quit
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
	//todo: stop before if running
	delete(b.bots, ID)
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
	log.Infof("loading bots: %+v", bots)
	for _, newBot := range bots {
		log.Infof("adding bot %+v", newBot)
		b.add(newBot)
	}
}

func (b *Bots) run(ID int64, quit chan bool) {
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
			log.Infof("=== End of ticker of %d reached check order with amount %s here ====", ID, b.bots[ID].Amount)
			// new order, check for changes goes here
			b.bots[ID].Amount = fmt.Sprintf("%8.2f", rand.Float64())
			if b.bots[ID].Active {
				ticker = time.NewTicker(time.Duration(minInt+rand.Intn(maxInt)) * time.Millisecond)

			}
			b.Unlock()
		case <-quit: //not sure if this is the correct way to do this
			ticker.Stop()
			log.Infof("Stopping bot ID %d", ID)
			wg.Done()
		}
	}
}
func (b *Bots) stop() {
	// save anyway here?
	for _, b := range b.bots {
		b.Quit <- true
	}
	wg.Wait()
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
	tau.Init(isStaging)

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

	bots.restore()
	bots.list()
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
	bots.stop()
	bots.save()
}

/*
https://my.webhookrelay.com/v1/webhooks/55adaeff-f8f3-4449-aa83-c2c5ff335244/de8064
 export RELAY_KEY=234efedd-9036-48b7-b1c1-41c2d5f8b4ce
 export RELAY_SECRET=JhIdkzUNweFN
*/
