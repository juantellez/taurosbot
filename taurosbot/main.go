package main //trading-bot

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	pb "git.vmo.mx/Tauros/tradingbot/proto"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	tau "git.vmo.mx/Tauros/tradingbot/taurosapi"
)

// myOrder to keep track of bot orders
type myOrder struct {
	Side   string
	Price  string
	Amount string
}

var myOrders struct {
	sync.RWMutex
	orders map[int64]*myOrder
}

type obItem struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
}

type bot struct {
	Side        string    //"buy" or "sell"
	Spread      float64   //how deep must the bid go for it to find the price
	Pct         float64   //percentage of the available balance that should be put in order
	MinInterval int       //minimum milliseconds to change
	MaxInterval int       //maximum milliseconds to change
	Quit        chan bool //channel used to stop the bot
}

type credentials struct {
	Tauros struct {
		Token string `json:"token"`
		TestingToken string `json:"testing_token"`
		Email string `json:"email"`
		Password string `json:"password"`
		Websocket string `json:"websocket"`
		BaseAPIUrl string `json:"base_api_url"`
		BalService string `json:"bal_service"`
		BalPort string `json:"bal_port"`
	} `json:"tauros"`
	OpenExchangeRates struct {
		Token string `json:"token"`
	} `json:"openexchangerates"`
	Gdax struct {
		APIToken string `json:"api_token"`
	} `json:"gdax"`
}

// bots configuration loaded from file
var bots struct {
	Market           string
	Bots             []bot
	Testing          bool
	TaurosToken      string
	TestingToken     string
	CoinbaseToken    string
	LogLevel         string
	BuyPct           float64
	SellPct          float64
	Spread           float64
	ExchangeModifier float64
	Email            string
	Password         string
}

// all current market data in a struct to be able to mux lock and lock
var marketData struct {
	sync.RWMutex
	currentAsk          float64
	currentBid          float64
	currentExchangeRate float64
	imbalance           float64
	buyBalance          float64
	sellBalance         float64
}

var balPort string
var balService string
var gdaxMarket string
var buySide string
var sellSide string
var tauMarket string
var quitExchangeRater chan bool
var gdaxDone chan bool
var wg sync.WaitGroup
var grpcGdaxConn *grpc.ClientConn
var grpcOxConn *grpc.ClientConn
var grpcBalConn *grpc.ClientConn
var getTicker = pb.NewTickerServiceClient(grpcGdaxConn)
var getSpreadPrice = pb.NewSpreadPriceServiceClient(grpcGdaxConn)
var getOxRate = pb.NewOxServiceClient(grpcOxConn)
var getTauBalances = pb.NewBalancesServiceClient(grpcBalConn)

var currentBidOrders struct {
	sync.RWMutex
	Items []obItem
}
var currentAskOrders struct {
	sync.RWMutex
	Items []obItem
}

func loadBotsFile(filename string) {
	log.Infof("Using bots configuration file: %s", filename)
	in, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Unable to load bots conf file: %v", err)
	}
	if err := json.Unmarshal(in, &bots); err != nil {
		log.Fatalf("Unable to unmarshal json file: %v", err)
	}
}

func loadCredentialsFile(filename string) {
	log.Infof("Using credentials file: %s", filename)
	var creds credentials
	in, err := ioutil.ReadFile(filename)
	if err!= nil {
		log.Fatalf("Unable to load credentials file: %v", err)
	}
	if err := json.Unmarshal(in, &creds); err != nil {
		log.Fatalf("Unable to unmarshal json file: %v", err)
	}
	bots.TaurosToken = creds.Tauros.Token
	bots.TestingToken = creds.Tauros.TestingToken
	bots.CoinbaseToken = creds.Gdax.APIToken
	balService = creds.Tauros.BalService
	balPort = "2224" //creds.Tauros.BalPort
}

func getExchangeRate() {
	res, err := getOxRate.GetOxRate(context.Background(), &pb.OxRequest{Currency: "MXN"})
	if err != nil {
		log.Fatalf("Unable to get exchange rate from ox grpc service: %v", err)
	}
	m, err := strconv.ParseFloat(res.Rate, 64)
	if err != nil {
		log.Errorf("Bad Rate %s unable to convert to float64: %v", res.Rate, err)
	}
	marketData.Lock()
	marketData.currentExchangeRate = m * bots.ExchangeModifier
	log.Infof("current exchange rate = %f", marketData.currentExchangeRate)
	marketData.Unlock()
}

func getGdaxTicker() (maxBid, minAsk decimal.Decimal) {
	res, err := getTicker.GetTicker(context.Background(), &pb.TickerRequest{Market: gdaxMarket})
	if err != nil {
		log.Fatalf("Unable to get ticker from gdax grpc service: %v", err)
	}
	var mb, ma decimal.Decimal
	mb, err = decimal.NewFromString(res.MaxBid)
	if err != nil {
		log.Fatalf("Bad Ticker MaxBid, unable %s to convert to decimal: %v", res.MaxBid, err)
	}
	ma, err = decimal.NewFromString(res.MinAsk)
	if err != nil {
		log.Fatalf("Bad Ticker MinAsk, unable to convert %s to decimal:%v", res.MinAsk, err)
	}
	return mb, ma
}

func getDepthPrice(side string, depth float64) float64 { //todo: refactor all naming "spread" to "depth"
	res, err := getSpreadPrice.GetSpreadPrice(context.Background(), &pb.SpreadPriceRequest{
		Market: gdaxMarket,
		Side:   side,
		Depth:  fmt.Sprintf("%f", depth),
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

func getBalances() (buyBal, sellBal float64) {
	res, err := getTauBalances.GetBalances(context.Background(), &pb.BalancesRequest{Market: bots.Market})
	if err != nil {
		log.Fatalf("Unable to get balances from balances grpc service: %v", err)
	}
	sellAvailable, err:=strconv.ParseFloat(res.Left.Available, 64)
	if err!= nil {
		log.Fatalf("Bad Left.Available, unable to convert %s to float64: %v", res.Left.Available, err)
	}
	sellFrozen, err := strconv.ParseFloat(res.Left.Frozen, 64)
	if err!= nil {
		log.Fatalf("Bad Left.Frozen, unable to convert %s to float64: %v", res.Left.Frozen, err)
	}
	buyAvailable, err := strconv.ParseFloat(res.Right.Available, 64)
	if err!= nil {
		log.Fatalf("Bad Right.Available, unable to convert %s to float64: %v", res.Right.Available, err)
	}
	buyFrozen, err := strconv.ParseFloat(res.Right.Frozen, 64)
	if err!= nil {
		log.Fatalf("Bad Right.Frozen, unable to convert %s to float64: %v", res.Right.Frozen, err)
	}
	//log.Infof("grpcBal result - buyAvailable=%f buyFrozen=%f, sellAvailable=%f sellFrozen=%f",buyAvailable,buyFrozen,sellAvailable,sellFrozen)
	return buyAvailable+buyFrozen, sellAvailable+sellFrozen //todo: this result should come from the grpc service itself
}

func updateBalances() {
	buyAvailable, sellAvailable := getBalances()
	if marketData.currentExchangeRate == 0.0 {
			log.Fatalf("Update Balances -> Current Exchange Rate cannot be zero")
	}
	maxBid, minAsk := getGdaxTicker()
	price, _ := decimal.Avg(maxBid, minAsk).Float64()
	buyAvailable = buyAvailable/ (price * marketData.currentExchangeRate)
	marketData.imbalance = 0.5
	if sellAvailable > 0.0 {
		marketData.imbalance = buyAvailable*bots.BuyPct / (buyAvailable*bots.BuyPct + sellAvailable*bots.SellPct)
	}
	if marketData.buyBalance != buyAvailable*bots.BuyPct {
		log.Infof("Old buyBalance: %f, New buybalance: %f", marketData.buyBalance, buyAvailable*bots.BuyPct)
		marketData.buyBalance = buyAvailable*bots.BuyPct
	}
	if marketData.sellBalance != sellAvailable*bots.SellPct {
		log.Infof("Old sellbalance: %f, new sellbalance: %f", marketData.sellBalance,sellAvailable*bots.SellPct)
		marketData.sellBalance = sellAvailable*bots.SellPct
	}
}

func addOrder(orderID int64, amount string, side string, price string) int64 {
	var err error
	myOrders.Lock()
	defer myOrders.Unlock()
	o := fmt.Sprintf("side=%s price=%s amount=%s orderID=%d", side, price, amount, orderID)

	//check order parameters
	log.Tracef("Adding order side=%s price=%s amount=%s orderID=%d", side, price, amount, orderID)
	a, _ := strconv.ParseFloat(amount, 64)
	if a<=0.0 { 
		log.Errorf("Cannot place an order with amount 0 or negative: %s"+o)
		return orderID
	}
	p, _ := strconv.ParseFloat(price, 64)
	if p<=0.0 {
		log.Errorf("Cannot place an order with price 0 or negative: %s"+o)
	}
	if (p*a<=5.0) && (sellSide=="mxn")  { //todo: get minimum amount order from Tauros API
		log.Warnf("Cannot place an order of less than $5 pesos: %s"+o)
  }
	
	//check if this order is already posted
	for _, o := range myOrders.orders {
		if o.Price == price && o.Side == side && o.Amount == amount {
			log.Tracef("Order %d did not change skipping", orderID)
			return orderID
		}
	}
	//check if this order will cause a self trade, if so cancel opposing orders before posting
	for i, o := range myOrders.orders {
		//log.Infof("side=%4s o.Side=%4s price=%s, o.Price=%s id=%d", side, o.Side, price, o.Price, i)
		if (side == "buy" && o.Side == "sell" && price >= o.Price) || (side == "sell" && o.Side == "buy" && price <= o.Price) {
			log.Infof("Preventing self trade - closing order #%d", i)
			if err := tau.CloseOrder(i); err != nil {
				log.Errorf("Unable to delete possible self trade order #%d - %v", i, err)
			}
			delete(myOrders.orders, i)
		}
	}
	//delete old bot order in current orderbooks before adding a new one
	if orderID != 0 && myOrders.orders[orderID] != nil {
		delete(myOrders.orders, orderID)
		if err:=tau.CloseOrder(orderID); err != nil {
			log.Errorf("Unable to delete previous bot order #%d, %v, %s", orderID,err,o)
			//this can happen if a trade was filled.
		}
	}
	log.Infof("New order %s", o)
	orderID, err = tau.PlaceOrder(tau.Message{
		Market: tauMarket,
		Amount: amount,
		Side:   side,
		Type:   "limit",
		Price:  price,
	})
	if err != nil {
		log.Fatalf("Unable to place new order %s: %v buyBalance=%f, sellBalance=%f",o,err,marketData.buyBalance,marketData.sellBalance)
	}
	//keep track of all orders made
	myOrders.orders[orderID] = &myOrder{
		Side:   side,
		Price:  price,
		Amount: amount,
	}
	return orderID
}

func runBot(b bot) {
	log.Infof("Starting bot: side %4s, spread %f, pct %f, interval %d-%d ...", b.Side, b.Spread, b.Pct, b.MinInterval, b.MaxInterval)
	var orderID int64
	var available, price float64
	var orderAmount, orderSide, orderPrice string
	if b.MinInterval >= b.MaxInterval {
		log.Fatalf("MinInterval (%d) cannot be greater than MaxInterval (%d)", b.MinInterval, b.MaxInterval)
	}
	ticker := time.NewTicker(time.Duration(b.MinInterval+rand.Intn(b.MaxInterval-b.MinInterval)) * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			ticker.Stop()
			marketData.RLock()
			updateBalances()
			if b.Side == "buy" {
				available = marketData.buyBalance 
				if available > 0.0 {
					price = getDepthPrice("buy", b.Spread) * marketData.currentExchangeRate * (1 - (bots.Spread * marketData.imbalance))
					orderAmount = fmt.Sprintf("%.8f", available*b.Pct)
					orderSide = "buy"
					orderPrice = fmt.Sprintf("%.8f", price)
				} else {
					log.Warn("no balance available for buying %s",buySide)
				}
			} else {
				available = marketData.sellBalance
				if available > 0.0 {
					price = getDepthPrice("sell", b.Spread) * marketData.currentExchangeRate * (1 + (bots.Spread * (1 - marketData.imbalance)))
					orderAmount = fmt.Sprintf("%.8f", available*b.Pct)
					orderSide = "sell"
					orderPrice = fmt.Sprintf("%.8f", price)
				} else {
					log.Warn("no balance available for selling %s",sellSide)
				}
			}
			marketData.RUnlock()
			orderID = addOrder(orderID, orderAmount, orderSide, orderPrice)
			ticker = time.NewTicker(time.Duration(b.MinInterval+rand.Intn(b.MaxInterval-b.MinInterval)) * time.Millisecond)
		case <-b.Quit:
			ticker.Stop()
			log.Infof("Stopping bot: side %4s, spread %f, pct %f, interval %d-%d ...", b.Side, b.Spread, b.Pct, b.MinInterval, b.MaxInterval)
			if orderID != 0 {
				if tau.CloseOrder(orderID) != nil {
					log.Warnf("Unable to close order #%d", orderID)
				}
				myOrders.Lock()
				delete(myOrders.orders, orderID)
				myOrders.Unlock()
			}
			wg.Done()
		}
	}
}

// logformatter.Format this is needed because the log outputs incorrectly in Docker-Compose
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
	logFormatter.LevelDesc = []string{"PANIC", "FATAL", "ERROR", "WARNI", "INFOR", "DEBUG","TRACE"}
	log.SetFormatter(logFormatter)
	myOrders.orders = make(map[int64]*myOrder)

	loadBotsFile(flag.Arg(0))
	loadCredentialsFile(flag.Arg(1))

	log.Info("Subscribing to gdax service at gdax:2222")
	grpcGdaxConn, err := grpc.Dial("gdax:2222", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect to GDAX grpc service at localhost:2222")
	}
	defer grpcGdaxConn.Close()
	getTicker = pb.NewTickerServiceClient(grpcGdaxConn)
	getSpreadPrice = pb.NewSpreadPriceServiceClient(grpcGdaxConn)

	log.Info("Subscribing to openexchange service at ox:2223")
	grpcOxConn, err := grpc.Dial("ox:2223", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect to Ox grpc service at localhost:2223")
	}
	defer grpcOxConn.Close() //probably not needed
	getOxRate = pb.NewOxServiceClient(grpcOxConn)

	log.Info("subscribing to balance service at "+balService+":"+balPort)
	grpcBalConn, err := grpc.Dial(balService+":"+balPort, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect to Bal grpc service at localhost:%s",balPort)
	}
	getTauBalances = pb.NewBalancesServiceClient(grpcBalConn)

	log.Printf("bots file loglevel =%s", bots.LogLevel)
	if loglevel, err := (log.ParseLevel(bots.LogLevel)); err != nil {
		log.Warn(`Incorrect LogLevel especified, must be "Panic", "Fatal", "Error", "Warn", "Info", "Debug" or "Trace"`)
	} else {
		log.SetLevel(loglevel)
	}
	log.Infof("Testing: %t TestingToken: %s TaurosToken: %s",bots.Testing,bots.TestingToken,bots.TaurosToken)
	if bots.Testing {
		tau.Init(true, bots.TestingToken)
	} else {
		tau.Init(false, bots.TaurosToken)
	}
	//todo: check total pct of buy and sell is <=1.0
	m := strings.Split(bots.Market, "-")
	buySide = strings.ToLower(m[0])
	sellSide = strings.ToLower(m[1])
	tauMarket = buySide + "-" + sellSide
	if sellSide == "mxn" {
		gdaxMarket = strings.ToUpper(buySide) + "-USD"
	} else {
		gdaxMarket = strings.ToUpper(buySide) + "-" + strings.ToUpper(sellSide)
	}

	log.Printf("Market = %s buySide = %s sellSide = %s", bots.Market, buySide, sellSide)
	getExchangeRate()
	log.Infof("Exchange rate is %f", marketData.currentExchangeRate)
	log.Info("Launching Exchange Rate updater")
	quitExchangeRater = make(chan bool, 2)
	go func() {
		ticker := time.NewTicker(time.Duration(5) * time.Minute)
		for {
			select {
			case <-ticker.C:
				getExchangeRate()
			case <-quitExchangeRater:
				ticker.Stop()
				log.Info("stopping exchange rate updater")
				wg.Done()
			}
		}
	}()

	log.Info("Ok, starting bots")
	if err := tau.CloseAllOrders(); err != nil {
		log.Errorf("Tauros Error closing all orders: %v", err)
	}

	// start bots
	for i := range bots.Bots { //first set up quit channel
		bots.Bots[i].Quit = make(chan bool)
	}

	for i, b := range bots.Bots {
		log.Infof("starting bot %d", i)
		go runBot(b)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Warnf("SIGTERM received, ending Tauros trading bots...")
//	log.SetLevel(log.TraceLevel)
	for i := range bots.Bots {
		wg.Add(1)
		log.Infof("quitting bot %d", i)
		bots.Bots[i].Quit <- true
	}

	log.Info("quitting exchange rater...")
	wg.Add(1)
	quitExchangeRater <- true
	wg.Wait() //not working?
}
