package main // gdax service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	pb "git.vmo.mx/Tauros/tradingbot/proto"
	ws "github.com/gorilla/websocket"
	gdax "github.com/preichenberger/go-coinbasepro/v2"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/yasushi-saito/rbtree"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// orderbook item, to keep track of the coinbase orderbook
type obItem struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
}

var wsDialer ws.Dialer
var wsConn *ws.Conn

type orderbook struct {
	*sync.RWMutex
	Asks *rbtree.Tree
	Bids *rbtree.Tree
}

var orderbooks map[string]orderbook

var markets = []string{"BTC-USD", "LTC-USD", "BCH-USD", "XLM-USD", "DASH-USD"}  //todo get from command line args

func compareOrders(a, b rbtree.Item) int {
	return b.(obItem).Price.Cmp(a.(obItem).Price)
}

func (o orderbook) updateBid(price string, amount string) {
	log.Debugf("updateBid %s %s",price,amount)
	p, _ := decimal.NewFromString(price)
	a, _ := decimal.NewFromString(amount)
	bid := obItem{
		Price:  p,
		Amount: a,
	}
	o.Bids.DeleteWithKey(bid)
	if a.IsZero() {
		return
	}
	o.Bids.Insert(bid)
}

func (o orderbook) updateAsk(price string, amount string) {
	log.Debugf("updateAsk %s %s",price,amount)
	p, _ := decimal.NewFromString(price)
	a, _ := decimal.NewFromString(amount)
	ask := obItem{
		Price:  p,
		Amount: a,
	}
	o.Asks.DeleteWithKey(ask)
	if a.IsZero() {
		return
	}
	o.Asks.Insert(ask)
}

func (o orderbook) getBidSpreadPrice(amount decimal.Decimal) float64 {
	o.RLock()
	defer o.RUnlock()
	iter := o.Bids.Min()
	bidAmount := iter.Item().(obItem).Amount
	for ; !iter.Limit() && bidAmount.LessThan(amount); iter = iter.Next() {
		bidAmount = bidAmount.Add(iter.Item().(obItem).Amount)
	}
	r, _ := iter.Item().(obItem).Price.Float64()
	return r
}

func (o orderbook) getAskSpreadPrice(amount decimal.Decimal) float64 {
	o.RLock()
	defer o.RUnlock()
	iter := o.Asks.Max()
	askAmount := iter.Item().(obItem).Amount
	for ; !iter.NegativeLimit() && askAmount.LessThan(amount); iter = iter.Prev() {
		askAmount = askAmount.Add(iter.Item().(obItem).Amount)
	}
	r, _ := iter.Item().(obItem).Price.Float64()
	return r
}

func (o orderbook) getTicker() (maxBid decimal.Decimal, minAsk decimal.Decimal) {
	var mb, ma decimal.Decimal
	o.RLock()
	if o.Bids.Len() > 0 {
		mb = o.Bids.Min().Item().(obItem).Price
	}
	if o.Asks.Len() > 0 {
		ma = o.Asks.Max().Item().(obItem).Price
	}
	o.RUnlock()
	return mb, ma
}

//todo: create a markets available grpc service.

func (o orderbook) reset() {
	o.Lock() 
	o.Asks = rbtree.NewTree(compareOrders)
	o.Bids = rbtree.NewTree(compareOrders)
	o.Unlock()
}

type grpcServer struct{}

var gdaxGrpcServer *grpc.Server

func (*grpcServer) GetSpreadPrice(ctx context.Context, req *pb.SpreadPriceRequest) (*pb.SpreadPrice, error) {
	//log.Infof("Get Spread Price request invoked with %+v", req)
	market := strings.ToUpper(req.Market)
	side := strings.ToLower(req.Side)
	if _, ok := orderbooks[req.Market]; !ok {
		return &pb.SpreadPrice{}, errors.New("Invalid market specified in call to GetSpreadPrice service: " + market)
	}
	depth, err := decimal.NewFromString(req.Depth)
	if err != nil {
		return &pb.SpreadPrice{}, err
	}
	if side == "buy" {
		return &pb.SpreadPrice{
			Market: market,
			Price:  fmt.Sprintf("%f", orderbooks[market].getBidSpreadPrice(depth)),
		}, nil
	}
	if side == "sell" {
		return &pb.SpreadPrice{
			Market: market,
			Price:  fmt.Sprintf("%f", orderbooks[market].getAskSpreadPrice(depth)),
		}, nil
	}
	return &pb.SpreadPrice{}, errors.New("Invalid SpreadPriceRequest side, must be 'buy' or 'sell' side: " + side)
}

func (*grpcServer) GetTicker(ctx context.Context, req *pb.TickerRequest) (*pb.Ticker, error) {
	//log.Infof("Get Ticker request invoked with %+v", req)
	market := strings.ToUpper(req.Market)
	if _, ok := orderbooks[market]; !ok {
		return &pb.Ticker{}, errors.New("Invalid market specified in call to GetTicker grpc")
	}
	maxBid, minAsk := orderbooks[req.Market].getTicker()
	log.Infof("Ticker maxBid=%s minAsk=%s", maxBid.String(), minAsk.String())
	if maxBid.GreaterThanOrEqual(minAsk) {
		log.Fatal("GetTicker: maxBid cannot be greater or equal to minAsk")
	}
	return &pb.Ticker{
		Market: req.Market,
		MaxBid: maxBid.String(),
		MinAsk: minAsk.String(),
	}, nil
}

func startGrpcServer(port string) {
	log.Info("Starting grpc server..")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to open listening port on %s, %v", port, err)
	}
	gdaxGrpcServer = grpc.NewServer()
	pb.RegisterTickerServiceServer(gdaxGrpcServer, &grpcServer{})
	pb.RegisterSpreadPriceServiceServer(gdaxGrpcServer, &grpcServer{})
	reflection.Register(gdaxGrpcServer)
	log.Infof("Done. Waiting for grpc requests at port %s...",port)
	err = gdaxGrpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("Unable to start listening for grpc on port %s: %v", port, err)
	}
}

func gdaxSubscribe() {
	var err error
	//todo: pause grpc service until the initial orderbook is loaded
	if wsConn != nil {
		wsConn.Close()
		for _, o := range orderbooks {
			o.reset()
		}
	}
	maxBid, minAsk := orderbooks["BTC-USD"].getTicker()
	log.Infof("==== after BTC-USD orderbook reset: maxBid=%s minAsk=%s",maxBid.String(),minAsk.String())

	wsConn, _, err = wsDialer.Dial("wss://ws-feed.pro.coinbase.com", nil)
	log.Info("Connecting to coinbase websocket...")
	if err != nil {
		log.Fatalf("general websocket error %v", err)
	}
	log.Info("Connected")
	subscribe := gdax.Message{
		Type: "subscribe",
		Channels: []gdax.MessageChannel{
			gdax.MessageChannel{
				Name:       "heartbeat",
				ProductIds: markets,
			},
			gdax.MessageChannel{
				Name:       "level2",
				ProductIds: markets,
			},
			gdax.MessageChannel{
				Name:       "matches",
				ProductIds: markets,
			},
		},
	}

	log.Info("Subscribing to gdax websocket...")
	if err := wsConn.WriteJSON(subscribe); err != nil {
		log.Fatalf("websocket subscribe error: %v", err)
	}
	log.Info("Done")
}
type logFormatter struct {
	TimestampFormat string
	LevelDesc       []string
}
func (f *logFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := fmt.Sprintf(entry.Time.Format(f.TimestampFormat))
	return []byte(fmt.Sprintf("%s %s %s\n", f.LevelDesc[entry.Level], timestamp, entry.Message)), nil
}

func main() {

	logFormatter := new(logFormatter)
	logFormatter.TimestampFormat = "2006-01-02 15:04:05"
	logFormatter.LevelDesc = []string{"PANIC", "FATAL", "ERROR", "WARNI", "INFOR", "DEBUG","TRACE"}
	log.SetFormatter(logFormatter)

	orderbooks = make(map[string]orderbook)
	for _, m := range markets {
		orderbooks[m] = orderbook{
			RWMutex: &sync.RWMutex{},
			Bids:    rbtree.NewTree(compareOrders),
			Asks:    rbtree.NewTree(compareOrders),
		}
	}

	gdaxSubscribe()

	go readGdax()
	go startGrpcServer("2222") //todo port in parameter
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Warnf("SIGTERM received, ending Tauros trading bots...")
	//os.Exit(0)
}

func readGdax() {
	for true {
		message := gdax.Message{}
		if err := wsConn.ReadJSON(&message); err != nil {
			log.Warnf("websocket error reading gdax message: %v", err)
			gdaxSubscribe() // try to restart todo: delete all trees first
		}
		market := message.ProductID
		if message.Type == "snapshot" {
			bids := message.Bids
			log.Infof("Processing snapshot of %s, %d bids...", market, len(bids))
			for _, bid := range bids {
				orderbooks[market].updateBid(bid.Price, bid.Size)
			}
			asks := message.Asks
			log.Infof("Processing snapshot of %s, %d asks...", market, len(asks))
			for _, ask := range asks {
				orderbooks[market].updateAsk(ask.Price, ask.Size)
			}
			log.Info("Done processing snapshots")
		}
		if message.Type == "l2update" {
			orderbooks[market].Lock()
			for _, change := range message.Changes {
				if change.Side == "sell" {
					orderbooks[market].updateAsk(change.Price, change.Size)
				} else {
					orderbooks[market].updateBid(change.Price, change.Size)
				}
			}
			orderbooks[market].Unlock()
			maxBid, minAsk := orderbooks[market].getTicker()
			log.Tracef("Ticker maxBid=%s minAsk=%s", maxBid.String(), minAsk.String())
			if maxBid.GreaterThanOrEqual(minAsk) {
				log.Warnf("l2update: maxBid (%s) cannot be greater or equal to minAsk(%s)", maxBid.String(),minAsk.String())
				log.Fatalf("l2update message: %v",message.Changes)
				//todo: I keep getting this random error, what can I do?
			}
		}
		if message.Type == "match" {
			message.Price = fixPrice(message.Price)
			//	log.Infof("match===: %s %4s p: %7s a: %12s", market, message.Side, message.Price, message.Size)
		}
	}
}

func fixPrice(price string) string {
	p, _ := decimal.NewFromString(price)
	return p.String()
}
