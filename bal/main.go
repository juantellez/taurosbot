package main // balances - maintain tauros balances

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	tau "git.vmo.mx/Tauros/tradingbot/taurosapi"
)

type balances struct {
	sync.Mutex
	balance map[string]float64
}

func (b *balances) updateBalances(account string, coin string, amount float64) {
	key := account + coin
	b.Lock()
	defer b.Unlock()
	if bal, exists := b.balance[key]; !exists {
		b.balance[key] = amount
	} else {
		b.balance[key] = bal + amount
	}
}

var bal balances

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

type grpcServer struct{}

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

func homeLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error: %v", err)
	}
	log.Tracef("Req Body = %s", string(reqBody))
	var whMessage tau.TauWebHookMessage
	if err := json.Unmarshal(reqBody, &whMessage); err != nil {
		log.Errorf("Error unmarshal json req body from webhook: %v", err)
	}
	log.Tracef("Received webhook from %s Description: %s", vars["apikey"], whMessage.Description)
	var account string
	//find account
	for _, t := range apiTokens {
		if t.APIToken[4:10] == vars["apikey"] {
			account = t.Account
			break
		}
	}
	if account == "" {
		log.Fatalf("received webhook of invalid account: [%s]", account)
	}
	switch whMessage.Type {
	case "TR": // Deposit, Withdrawal
		bal.updateBalances(account, whMessage.Object.LeftCoin, strconv.ParseFloat(whMessage.Object.AmountReceived, 64))
	case "TD": // Market taker trade executed
	case "OP": // Market maker order placed
	case "OF": // Market maker order filled
	}
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

	loadCredentialsFile(flag.Arg(0))
	tau.Init(isStaging)

	//remove current webhooks and add new webhooks
	for _, t := range apiTokens {
		log.Infof("deleting and recreating webhooks for %s using apitoken %s", t.Account, t.APIToken)
		err := tau.DeleteWebhooks(t.APIToken)
		if err != nil {
			log.Fatalf("%v", err)
		}
		webhookID, err := tau.CreateWebhook(tau.Webhook{
			Name:              "Bot",
			Endpoint:          baseWebhookURL + "/" + t.APIToken[4:10], //just use a part of apikey
			NotifyDeposit:     true,
			NotifyWithdrawal:  true,
			NotifyOrderPlaced: true,
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
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/{apikey}", homeLink)
	log.Print("Ready to receive POST requests at port 9090")
	log.Fatal(http.ListenAndServe(":9090", router))
}

/*
https://my.webhookrelay.com/v1/webhooks/55adaeff-f8f3-4449-aa83-c2c5ff335244/de8064
 export RELAY_KEY=234efedd-9036-48b7-b1c1-41c2d5f8b4ce
 export RELAY_SECRET=JhIdkzUNweFN
*/
