package main // testapi

import (
	"log"
	//"os"
	//"os/signal"
	//"syscall"
	"context"

	pb "git.vmo.mx/Tauros/tradingbot/proto"
	"google.golang.org/grpc"
)

var grpcConn *grpc.ClientConn
var getTauBalances = pb.NewBalancesServiceClient(grpcConn)

func getBalances(market string) {
	res, err := getTauBalances.GetBalances(context.Background(), &pb.BalancesRequest{Market: market})
	if err != nil {
		log.Fatalf("Unable to get balances from balances grpc service: %v", err)
	}
	log.Printf("left available: %s right available: %s",res.Left.Available,res.Right.Available)
}

func main() {
	var err error
	grpcConn, err = grpc.Dial("localhost:2224", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect to GDAX grpc service at localhost:2224")
	}
	defer grpcConn.Close()
	getTauBalances = pb.NewBalancesServiceClient(grpcConn)
  getBalances("BTC-MXN")
}
