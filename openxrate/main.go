package main // openxrate - Open Exchange Rate updater

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "git.vmo.mx/Tauros/tradingbot/proto"
)

type rate struct {
	Rates struct {
		MXN float64
	} `json:"rates"`
}

//todo put apiToken and credentials types in lib
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

var currentRate rate
var oxGrpcServer *grpc.Server
var mux sync.RWMutex
var oxToken string

func loadCredentialsFile(filename string) {
	log.Infof("Using credentials file: %s", filename)
	var creds credentials
	in, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Unable to load credentials file: %v", err)
	}
	if err := json.Unmarshal(in, &creds); err != nil {
		log.Fatalf("Unable to unmarshal json file: %v", err)
	}
	oxToken = creds.OpenexchangeratesToken
}

func (*grpcServer) GetOxRate(ctx context.Context, req *pb.OxRequest) (*pb.OxRate, error) {
	log.Infof("Get OxRate request invoked with %+v", req)
	mux.RLock()
	defer mux.RUnlock()
	return &pb.OxRate{
		Currency: req.Currency,
		Rate:     fmt.Sprintf("%f", currentRate.Rates.MXN),
	}, nil
}

func getRate(apiToken string) (rate, error) {
	httpReq, err := http.NewRequest("GET", "http://openexchangerates.org/api/latest.json?app_id="+apiToken, nil)
	if err != nil {
		return rate{}, fmt.Errorf("Unable to setup http request for openexchangerates: %v", err)
	}
	client := http.Client{Timeout: time.Second * 60}
	resp, err := client.Do(httpReq)
	if err != nil {
		return rate{}, fmt.Errorf("error reading response from openexchangerates: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return rate{}, fmt.Errorf("openexchangerate returned non 200 code: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return rate{}, fmt.Errorf("ioutil error: %v", err)
	}
	//log.Infof("body=%s", body)
	mxnRate := rate{}
	if err := json.Unmarshal(body, &mxnRate); err != nil {
		return rate{}, fmt.Errorf("Unable to unmarshal json from openexchangerates: %v", err)
	}

	log.Infof("mxnRate = %+v, rate=%f", mxnRate, mxnRate.Rates.MXN)
	return mxnRate, nil
}

func startGrpcServer(port string) {
	log.Info("Starting grpc server..")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to open listening port on %s, %v", port, err)
	}
	oxGrpcServer = grpc.NewServer()
	pb.RegisterOxServiceServer(oxGrpcServer, &grpcServer{})
	reflection.Register(oxGrpcServer)
	log.Infof("Done. Waiting for grpc requests on port %s...", port)
	err = oxGrpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("Unable to start listening for grpce: %v", err)
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
	logFormatter.LevelDesc = []string{"PANIC", "FATAL", "ERROR", "WARNI", "INFOR", "DEBUG", "TRACE"}
	log.SetFormatter(logFormatter)

	loadCredentialsFile(flag.Arg(0))
	go func() {
		var err error
		for {
			mux.Lock()
			currentRate, err = getRate(oxToken)
			mux.Unlock()
			if err != nil { //todo check if error is http timeout, if so try several times
				log.Fatalf("%v", err)
			}
			time.Sleep(time.Duration(15) * time.Minute)
		}
	}()
	go startGrpcServer("2223") //todo port in parameter

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
