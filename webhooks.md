
## New deposit
```json
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
```
## withdrawal
```json
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
```
## trade
```json
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
```
## new order
```json
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
```
## order filled (new maker trade)
```json
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
```
## tauros transfer withdrawal
```json
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
```
## tauros transfer deposit
```json
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
```
