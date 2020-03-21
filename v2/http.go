package main // bot - http related parts

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	tau "git.vmo.mx/Tauros/tradingbot/taurosapi"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func respJSON(w http.ResponseWriter, success bool, message string, data string) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{
  "success":"%t",
  "message":"%s",
  "data":"%s"
}`, success, message, data)))
}

func webhooksLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Infof("POST /webhooks/%s", vars["apikey"])
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error: %v", err)
		return //todo: return 50x
	}
	//log.Infof("Req Body = %s", string(reqBody))
	var whMessage tau.TauWebHookMessage
	if err := json.Unmarshal(reqBody, &whMessage); err != nil {
		log.Errorf("Error unmarshal json req body from webhook: %v", err)
	}
	//log.Infof("Received webhook type %s Description: %s", whMessage.Type, whMessage.Description)
	var account string
	//find account
	for a, t := range apiTokens {
		if t[4:10] == vars["apikey"] {
			account = a
			break
		}
	}
	if account == "" {
		log.Fatalf("received webhook of invalid account: [%s]", vars["apikey"])
	}
	if whMessage.Type == "OC" || whMessage.Type == "OP" {
		return // ignore, bug from the service
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
	log.Info("GET /ping")
	respJSON(w, true, "ok!", "")
}

func getBalancesLink(w http.ResponseWriter, r *http.Request) {
	log.Info("GET /balances")
	respJSON(w, true, "ok!", string(bal.json()))
}

func getBotLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	varsBotID := vars["botid"]
	log.Infof("GET /bot/%s", varsBotID)
	botID, err := strconv.ParseInt(varsBotID, 10, 64)
	if err != nil {
		respJSON(w, false, "Unable to parse integer: "+varsBotID, "")
		return
	}
	botJSON := string(bots.getJSON(botID))
	if botJSON == "null" {
		respJSON(w, false, "bot id "+varsBotID+" not found", "")
		return
	}
	respJSON(w, true, "ok!", botJSON)
}

func deleteBotLink(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	varsBotID := vars["botid"]
	log.Infof("DELETE /bot/%s", varsBotID)
	botID, err := strconv.ParseInt(varsBotID, 10, 64)
	if err != nil {
		respJSON(w, false, "unable to parse integer: "+varsBotID, "")
		return
	}
	bots.delete(botID) //todo check if exists?
	respJSON(w, true, "ok!", "")
}

func postBotLink(w http.ResponseWriter, r *http.Request) {
	log.Info("POST /bot")
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error: %v", err)
		return //todo return http status 50x
	}
	//todo: check for valid bot data: valid account, valid market, positive spread, pct, etc
	var newbot Bot
	if err := json.Unmarshal(reqBody, &newbot); err != nil {
		log.Errorf("Error unmarshal json req body to bot: %v", err)
		return //todo: return http error
	}
	botID := bots.add(newbot)
	respJSON(w, true, "ok!", fmt.Sprintf(`{"id":"%d"}`, botID))
}

func putBotLink(w http.ResponseWriter, r *http.Request) {
	log.Info("PUT /bot")
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Error: %v", err)
		return //todo return http status error
	}
	//todo: do basic parameter checks
	var botUpdate BotUpdate
	if err := json.Unmarshal(reqBody, &botUpdate); err != nil {
		log.Errorf("Error unmarshal json req body to updateBot: %v", err)
		return //todo send failed
	}
	if err := bots.update(botUpdate); err != nil {
		log.Errorf("Unable to update bot %v", err)
		return
	}
	respJSON(w, true, "ok!", "")
}

func getBotsLink(w http.ResponseWriter, r *http.Request) {
	log.Info("GET /bots")
	botsJSON := string(bots.getJSONAll())
	respJSON(w, true, "ok!", botsJSON)
}

func getTickersLink(w http.ResponseWriter, r *http.Request) {
	//todo
}

func getBotPauseLink(w http.ResponseWriter, r *http.Request) {
	varsBotID := mux.Vars(r)["botid"]
	log.Infof("GET /bot/pause/%s", varsBotID)
	botID, err := strconv.ParseInt(varsBotID, 10, 64)
	if err != nil {
		respJSON(w, false, "unable to parse integer: "+varsBotID, "")
	}
	if err := bots.deactivate(botID); err != nil {
		respJSON(w, false, err.Error(), "")
		return
	}
	respJSON(w, true, "ok!", "")
}

func getBotUnpauseLink(w http.ResponseWriter, r *http.Request) {
	varsBotID := mux.Vars(r)["botid"]
	log.Infof("GET /bot/unpause/%s", varsBotID)
	botID, err := strconv.ParseInt(varsBotID, 10, 64)
	if err != nil {
		respJSON(w, false, "unable to parse integer: "+varsBotID, "")
	}
	if err := bots.activate(botID); err != nil {
		respJSON(w, false, err.Error(), "")
		return
	}
	respJSON(w, true, "ok!", "")
}
func getOrdersLink(w http.ResponseWriter, r *http.Request) {
	log.Info(" GET /orders")
	ordersJSON := string(orders.json())
	respJSON(w, true, "ok!", ordersJSON)
}

func startRouter(srv *http.Server) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/webhooks/{apikey}", webhooksLink).Methods("POST") //update balances
	router.HandleFunc("/ping", pingLink).Methods("GET")                   //ping service
	router.HandleFunc("/bot/{botid}", getBotLink).Methods("GET")          //get bot data
	router.HandleFunc("/bot/{botid}", deleteBotLink).Methods("DELETE")    //delete bot
	router.HandleFunc("/bot", postBotLink).Methods("POST")                //add new bot
	router.HandleFunc("/bot", putBotLink).Methods("PUT")                  //update bot data
	router.HandleFunc("/balances", getBalancesLink).Methods("GET")
	router.HandleFunc("/bots", getBotsLink).Methods("GET")
	router.HandleFunc("/bot/pause/{botid}", getBotPauseLink).Methods("GET")
	router.HandleFunc("/bot/unpause/{botid}", getBotUnpauseLink).Methods("GET")
	router.HandleFunc("/orders", getOrdersLink).Methods("GET")

	//todo:
	router.HandleFunc("/tickers", getTickersLink).Methods("GET")

	srv.Handler = router
	go func() {
		log.Print("Ready to receive POST requests at port 9090")
		if err := srv.ListenAndServe(); err != nil { //":9090", router); err != nil {
			log.Errorf("http server error %v", err)
		}
	}()
}
