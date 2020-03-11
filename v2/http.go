package main // bot - http related parts

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	tau "git.vmo.mx/Tauros/tradingbot/taurosapi"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func webhooksLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error: %v", err)
	}
	log.Infof("Req Body = %s", string(reqBody))
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
		log.Fatalf("received webhook of invalid account: [%s]", vars["apikey"])
	}
	log.Printf("Balances of %s BEFORE webhook %s:", account, whMessage.Type)
	bal.list()
	switch whMessage.Type {
	case "TR": // Deposit, Withdrawal, Transfer sent and received
		prefix := "" //negative to subtract
		if whMessage.Object.Type == "withdrawal" {
			prefix = "-"
		}
		bal.update(account, whMessage.Object.Coin, prefix+whMessage.Object.TotalAmount)
	case "OF": //market maker order fill (trade) executed
		bal.update(account, whMessage.Object.LeftCoin, whMessage.Object.TradeAmountReceived)
		bal.update(account, whMessage.Object.RightCoin, "-"+whMessage.Object.TradeAmountPaid)
	case "TD": //
		bal.update(account, whMessage.Object.LeftCoin, whMessage.Object.AmountReceived)
		bal.update(account, whMessage.Object.RightCoin, "-"+whMessage.Object.AmountPaid)
	default:
		log.Errorf("Unknown webhook message type: %s", whMessage.Type)
	}
	log.Print("balances AFTER webhook:")
	bal.list()
}

func pingLink(w http.ResponseWriter, r *http.Request) {
	//return {succeess: true, message: "ok!", data: null}
}

func getBotLink(w http.ResponseWriter, r *http.Request)      {}
func deleteBotLink(w http.ResponseWriter, r *http.Request)   {}
func postBotLink(w http.ResponseWriter, r *http.Request)     {}
func putBotLink(w http.ResponseWriter, r *http.Request)      {}
func getBotsLink(w http.ResponseWriter, r *http.Request)     {}
func getBalancesLink(w http.ResponseWriter, r *http.Request) {}
func getTickersLink(w http.ResponseWriter, r *http.Request)  {}
func getBotPauseLink(w http.ResponseWriter, r *http.Request) {}

func startRouter(srv *http.Server) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/webhooks/{apikey}", webhooksLink).Methods("POST")
	router.HandleFunc("/ping", pingLink).Methods("GET")
	router.HandleFunc("/bot/{botid}", getBotLink).Methods("GET")
	router.HandleFunc("/bot/{botid}", deleteBotLink).Methods("DELETE")
	router.HandleFunc("/bot", postBotLink).Methods("POST")
	router.HandleFunc("/bot/{botid}", putBotLink).Methods("PUT")
	router.HandleFunc("/bots", getBotsLink).Methods("GET")
	router.HandleFunc("/balances", getBalancesLink).Methods("GET")
	router.HandleFunc("/tickers", getTickersLink).Methods("GET")
	router.HandleFunc("/bot/pause/{botid}", getBotPauseLink).Methods("GET")

	srv.Handler = router
	go func() {
		log.Print("Ready to receive POST requests at port 9090")
		if err := srv.ListenAndServe(); err != nil { //":9090", router); err != nil {
			log.Errorf("http server error %v", err)
		}
	}()
}
