## Requisites
1. api token from Tauros
2. api token from openexchangerates.org
3. Go version 1.11.x
4. Docker and Docker Compose
5. Linux environment although it could run in Windows too, but I haven't tested it.
6. Protobuf v3.5 installed with go plugins:
```bash
wget https://github.com/google/protobuf/releases/download/v3.5.1/protoc-3.5.1-linux-x86_64.zip
sudo mkdir /usr/local/protobuf
sudo unzip -j protoc-3.5.1-linux-x86_64.zip -d /usr/local/protobuf
sudo chmod -R 755 /usr/local/protobuf
export PATH=$PATH:/usr/local/protobuf/bin 
go get -u github.com/golang/protobuf/proto
go get -u github.com/golang/protobuf/protoc-gen-go
```
7. Python >3.6 (needed for socket.io to get notification of new trades and deposits) in the bal folder is the README to setup python. Soon Tauros will update its websocket service using native sockets instead of socket.io. I will then remove the python code and implement a golang version of this service.
## steps to runeval
```
Create a bots directory to have the bots and credentials json files, and copy the files to the docker volume, before running.
# 
# make all
# 
# docker-compose up
```

## TODO
1. http api interface to tradingbot to stop and start bots without restarting and eliminate json bot configuration files, and do other live changes.
2. Use json config file for openexchange rate instead of env

## Sample bot configuration JSON file:
```json
{
  "Market": "BTC-MXN", //market 
  "Bots": [
{
  "Side": "buy", 
  "Spread": 40, // the minimum spread it looks for in coinbase 
  "Pct": 0.1, // percentage of assigned balanced to this market used for this bot
  "MinInterval": 5000, // chooses a random interval between min and maxwait intervals before updating order
  "MaxInterval": 10000 //
},
{
  "Side": "buy",
  "Spread": 200,
  "Pct": 0.3,
  "MinInterval": 20000,
  "MaxInterval": 30000
},
{
  "Side": "sell",
  "Spread": 40,
  "Pct": 0.05,
  "MinInterval": 5000,
  "MaxInterval": 10000
},
{
  "Side": "sell",
  "Spread": 100,
  "Pct": 0.25,
  "MinInterval": 5000,
  "MaxInterval": 10000
},
//... add as many bots you like here
],
"Testing": false, //if true runs the bots in staging of tauros exchange
"LogLevel": "Info", //log level of the bot daemon according to logrus go library
"BuyPct":0.3, // balance assigned to this market on the buy side of all available
"SellPct": 0.3, //balance assigned to this market on the sell side of all available
"Spread": 0.005, //minimum spread between buy and sell of the bots
"ExchangeModifier": 1.005 //factor applied to exchange rate (set to 1.0 if none)
}
```

## Sample credentials JSON file:
```json
{
    "tauros" : {
        "token" : "your tauros api token",
        "testing_token" : "your tauros api testing token",
        "email": "yourtaurosaccount@email.com",
        "password": "your tauros account pwd",
        "websocket": "wss://private-ws.coinbtr.com",
        "base_api_url": "https://api.tauros.io/api/",
        "bal_service": "docker service name"
    },
    "openexchangerates" : {
        "token" : "openexchangerates.com api token (free for low usage)"
    },
    "gdax" : {
        "api_token": "not yet used"
    }
}
```