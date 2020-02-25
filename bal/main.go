package main // balances - maintain tauros balances

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	pb "git.vmo.mx/Tauros/tradingbot/proto"
	log "github.com/sirupsen/logrus"

	tau "git.vmo.mx/Tauros/tradingbot/taurosapi"
)

var balances struct {
	sync.RWMutex
	b map[string]*float64
}

type grpcServer struct{}

var apiToken = "2ce1cbe69b9108399bd80822b1dd2a564ed92358"
var testing = true

func (*grpcServer) GetBalance(ctx context.Context, req *pb.BalancesRequest) (*pb.Balances, error) {
	log.Infof("Get balance request invoked with %+v", req)
	balances.RLock()
	defer balances.RUnlock()
	m := strings.Split(req.Market, "-")
	leftCoin := strings.ToUpper(m[0])
	rightCoin := strings.ToUpper(m[1])
	leftBalance := fmt.Sprintf("%f", *balances.b[leftCoin])
	rightBalance := fmt.Sprintf("%f", *balances.b[rightCoin])
	return &pb.Balances{
		Right: &pb.Balance{
			Currency:  leftCoin,
			Available: leftBalance,
		},
		Left: &pb.Balance{
			Currency:  rightCoin,
			Available: rightBalance,
		},
	}, nil
}

func homeLink(w http.ResponseWriter, r *http.Request) {
	var err error
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error: %v", err)
	}
	log.Tracef("Req Body = %s", string(reqBody))
	var whMessage tau.TauWebHookMessage
	if err := json.Unmarshal(reqBody, &whMessage); err != nil {
		log.Errorf("Error unmarshal json req body from webhook: %v", err)
	}
	log.Printf("Received webhook: s", whMessage.Description)

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
	tau.Init(testing)
	w, err := tau.GetWebhooks(apiToken)
	if err != nil {
		log.Errorf("Error getting webhooks: %v", err)
	}
	log.Printf("Webhooks = %+v", w)
	tau.DeleteWebhooks(apiToken)
	webhookID, err := tau.CreateWebhook(tau.Webhook{
		Name:              "Testing-02",
		Endpoint:          "https://my.webhookrelay.com/v1/webhooks/55adaeff-f8f3-4449-aa83-c2c5ff335244",
		NotifyDeposit:     true,
		NotifyWithdrawal:  true,
		NotifyOrderPlaced: true,
		NotifyOrderFilled: true,
	}, apiToken)
	if err != nil {
		log.Errorf("Error creating webhook: %v", err)
	} else {
		log.Printf("Created webhook with id %d", webhookID)
	}
	//router := mux.NewRouter().StrictSlash(true)
	// router.HandleFunc("/", homeLink)
	//log.Print("Ready to receive POST requests at port 9090")
	//log.Fatal(http.ListenAndServe(":9090", router))
}

/*

// export RELAY_KEY=234efedd-9036-48b7-b1c1-41c2d5f8b4ce
// export RELAY_SECRET=JhIdkzUNweFN
==== deposit ====

{
   "title":"New LTC deposit",
   "description":"You have received 0.01 LTC via Blockchain",
   "type":"TR",
   "date":"2020-02-20 22:16:35.527432+00:00",
   "object":{
      "sender":null,
      "receiver":"david@montebit.com",
      "coin":"LTC",
      "coin_name":"Test Litecoin",
      "coin_icon":"https://staging-static.coinbtr.com/media/coins/LTC.png",
      "amount":"0.01",
      "txId":"594600c67f76ff6be6d6fcda658bc28f2dc88026cd2ec6e182216e873ad97762",
      "confirmed":true,
      "confirmed_at":"2020-02-20 22:16:35.439214+00:00",
      "is_innerTransfer":false,
      "address":"QTs9KtU7eX2KeQvuWD1wLQgSncSSt9bs4X",
      "explorer_link":"https://chain.so/tx/LTCTEST/594600c67f76ff6be6d6fcda658bc28f2dc88026cd2ec6e182216e873ad97762",
      "fee_amount":"0",
      "total_amount":"0.01",
      "type":"deposit",
      "description":null,
      "id":820
   }
}
==== withdrawal ====
{
	"title":"New LTC withdrawal",
	"description":"You have sent 0.1 LTC via Blockchain",
	"type":"TR",
	"date":"2020-02-20 22:33:08.808652+00:00",
	"object":{
		 "sender":"david@montebit.com",
		 "receiver":null,
		 "coin":"LTC",
		 "coin_name":"Test Litecoin",
		 "coin_icon":"https://staging-static.coinbtr.com/media/coins/LTC.png",
		 "amount":"0.1",
		 "txId":"19f547ce5576a9cf58bfc53f84c895cffb29f8fd2d581edd6f64d3c7f007d4a1",
		 "confirmed":true,
		 "confirmed_at":"2020-02-20 22:33:08.747393+00:00",
		 "is_innerTransfer":false,
		 "address":"tltc1qas0p2206mn52lg9vqkld4swczlczr39ga4e8la",
		 "explorer_link":"https://chain.so/tx/LTCTEST/19f547ce5576a9cf58bfc53f84c895cffb29f8fd2d581edd6f64d3c7f007d4a1",
		 "fee_amount":"0.0001",
		 "total_amount":"0.1001",
		 "type":"withdrawal",
		 "description":null,
		 "id":821
	}
}
=== trade ===
{
   "title":"New trade",
   "description":"New trade in BTC-MXN orderbook",
   "type":"TD",
   "date":"2020-02-20 22:52:48.287919+00:00",
   "object":{
      "market":"BTC-MXN",
      "side":"BUY",
      "amount_paid":"20000.1",
      "amount_received":"0.0996005",
      "price":"200000.00",
      "fee_amount":"0.0004",
      "created_at":"2020-02-20 22:52:48.010093+00:00",
      "left_coin":"BTC",
      "right_coin":"MXN",
      "left_coin_icon":"https://staging-static.coinbtr.com/media/coins/BTC_GFDG7yC.png",
      "right_coin_icon":"https://staging-static.coinbtr.com/media/coins/MXN_ZDfVVtQ.png",
      "filled_as":"taker",
      "closed_at":"2020-02-20 22:52:48.010093+00:00"
   }
}
==== new order =====
{
   "title":"New order placed",
   "description":"Your SELL order has been placed in BTC-MXN orderbook",
   "type":"OP",
   "date":"2020-02-20 23:17:31.506377+00:00",
   "object":{
      "market":"BTC-MXN",
      "side":"SELL",
      "amount":"0.2",
      "initial_amount":"0.2",
      "filled":"0",
      "value":"39800",
      "initial_value":"39800",
      "price":"199000.00",
      "fee_decimal":"0.00200000",
      "fee_percent":"0.20000000",
      "fee_amount_paid":"0",
      "is_open":true,
      "amount_paid":"0",
      "amount_received":"0",
      "created_at":"2020-02-20 23:17:31.381756+00:00",
      "closed_at":"2020-02-14 16:12:52.275009+00:00",
      "left_coin":"BTC",
      "right_coin":"MXN",
      "left_coin_icon":"https://staging-static.coinbtr.com/media/coins/BTC_GFDG7yC.png",
      "right_coin_icon":"https://staging-static.coinbtr.com/media/coins/MXN_ZDfVVtQ.png",
      "destination_wallet_available_balance":"137461.37",
      "origin_wallet_frozen_balance":"0",
      "id":108074
   }
}
==== order filled (new maker trade) ====
{
   "title":"Order filled",
   "description":"Your SELL order has been partially filled",
   "type":"OF",
   "date":"2020-02-20 23:47:27.901840+00:00",
   "object":{
      "market":"BTC-MXN",
      "side":"SELL",
      "amount":"0.1",
      "initial_amount":"0.2",
      "filled":"0.1",
      "value":"19900",
      "initial_value":"39800",
      "price":"199000.00",
      "fee_decimal":"0.00200000",
      "fee_percent":"0.20000000",
      "fee_amount_paid":"39.8",
      "is_open":true,
      "amount_paid":"0.1",
      "amount_received":"19860.2",
      "created_at":"2020-02-20 23:17:31.381756+00:00",
      "closed_at":"2020-02-14 16:12:52.275009+00:00",
      "left_coin":"BTC",
      "right_coin":"MXN",
      "left_coin_icon":"https://staging-static.coinbtr.com/media/coins/BTC_GFDG7yC.png",
      "right_coin_icon":"https://staging-static.coinbtr.com/media/coins/MXN_ZDfVVtQ.png",
      "destination_wallet_available_balance":"137421.57",
      "origin_wallet_frozen_balance":"0",
      "id":108074
   }
}

==== tauros transfer withdrawal ====
{
   "title":"New MXN withdrawal",
   "description":"You have sent 1000 MXN via Tauros Transfer\u00ae",
   "type":"TR",
   "date":"2020-02-21 00:20:09.730328+00:00",
   "object":{
      "sender":"david@montebit.com",
      "receiver":"salvadormlnz@gmail.com",
      "coin":"MXN",
      "coin_name":"Pesos Mexicanos",
      "coin_icon":"https://staging-static.coinbtr.com/media/coins/MXN_ZDfVVtQ.png",
      "amount":"1000",
      "txId":null,
      "confirmed":true,
      "confirmed_at":"2020-02-21 00:20:09.360990+00:00",
      "is_innerTransfer":true,
      "address":"",
      "explorer_link":null,
      "fee_amount":"0",
      "total_amount":"1000",
      "type":"withdrawal",
      "description":"prueba",
      "id":822
   }
}
==== tauros transfer deposit ====
{
   "title":"New MXN deposit",
   "description":"You have received 1000 MXN via Tauros Transfer\u00ae",
   "type":"TR",
   "date":"2020-02-21 00:27:33.046671+00:00",
   "object":{
      "sender":"salvadormlnz@gmail.com",
      "receiver":"david@montebit.com",
      "coin":"MXN",
      "coin_name":"Pesos Mexicanos",
      "coin_icon":"https://staging-static.coinbtr.com/media/coins/MXN_ZDfVVtQ.png",
      "amount":"1000",
      "txId":null,
      "confirmed":true,
      "confirmed_at":"2020-02-21 00:27:29.858271+00:00",
      "is_innerTransfer":true,
      "address":"",
      "explorer_link":null,
      "fee_amount":"0",
      "total_amount":"1000",
      "type":"deposit",
      "description":"prueba",
      "id":823
   }
}
*/
