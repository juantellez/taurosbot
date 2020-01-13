#!/usr/bin/env python3

import json, time, requests, threading, os, socketio, grpc, sys

from concurrent import futures

import balance_pb2, balance_pb2_grpc

balances = {}

print('reading credentials json file:',sys.argv[1])

with open(sys.argv[1], mode='r') as json_file:
  data = json.load(json_file)
  TAU_TOKEN = data['tauros']['token']
  TAU_EMAIL = data['tauros']['email']
  TAU_PWD = data['tauros']['password']
  BASE_URL = data['tauros']['base_api_url']
  WS = data['tauros']['websocket']
  GRPC_PORT = '2224' # data['tauros']['bal_port']

print('TAU_TOKEN=',TAU_TOKEN)
print('TAU_EMAIL=',TAU_EMAIL)
print('TAU_PWD=',TAU_PWD)
print('BASE_URL=',BASE_URL)
print('WS=',WS)

class BalancesServicer(balance_pb2_grpc.BalancesServiceServicer):
  def GetBalances(self, request, context):
    market=request.Market
    left=market.split('-')[0]
    right=market.split('-')[1]
    result = { #todo: just manage a combined balance of available+frozen
      'Right': {'Currency': right, 'Available': str(balances[right]['available']), 'Frozen': str(balances[right]['frozen'])}, 
      'Left': {'Currency': left, 'Available': str(balances[left]['available']), 'Frozen': str(balances[left]['frozen'])},
    }
    return balance_pb2.Balances(**result)

server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

balance_pb2_grpc.add_BalancesServiceServicer_to_server(BalancesServicer(), server)

print('Starting grpc server. Listening on port ',GRPC_PORT)
server.add_insecure_port('[::]:'+GRPC_PORT)
server.start()

# first load initial balances

headers = {
  'Authorization': f'Token {TAU_TOKEN}',
  'Content-Type': 'application/json',
}

response = requests.get(
  url=BASE_URL+'v1/data/listbalances/',
  headers=headers,
)

print(response.content)
wallets = response.json()['data']['wallets']

for w in wallets:
  balances[w['coin']] = { #todo: manage only a combined balance of available+frozen
    'available': float(w['balances']['available']),
    'frozen': float(w['balances']['frozen']),
  }
   
# print(balances)

# get jwt token necessary for socketio
response = requests.post(
  url=BASE_URL + 'v2/auth/signin/',
  headers={'Content-Type': 'application/json'},
  data=json.dumps({
    'email': TAU_EMAIL,
    'password': TAU_PWD,
    'device_name': "Bot",
    'unique_device_id': "f8c8a829-c1fa-405f-b9e3-0d50c7d2b9f0",
  }),
)

print(response.json()) #todo: process invalid login credencials
jwtToken = response.json()['payload']['token']

# print(jwtToken)

# start socketio connection

sio = socketio.Client(reconnection=True)
@sio.event
def connect():
  print('ws connected!')

@sio.on('notification')
def on_message(data):
    #print('new notification:')
    #print(data)
    if (data['type']=='TD'):
      print('==================================')
      received = float(data['object']['amount_received'])
      paid = float(data['object']['amount_paid'])
      if data['object']['side']=='SELL':
        to_bal = data['object']['right_coin']
        from_bal = data['object']['left_coin']
      else:
        to_bal = data['object']['left_coin']
        from_bal = data['object']['right_coin']
      print('NEW TRADE')
      print('coin %s balance %f' % (from_bal, balances[from_bal]['available']))
      print('coin %s balance %f' % (to_bal, blances[to_bal]['available']))
      print('received=%f paid=% f from %s to %s' % (received,paid,from_bal,to_bal))
      balances[from_bal]['available'] -= paid
      balances[to_bal]['available'] += received
      print('coin %s balance %f' % (from_bal, balances[from_bal]['available']))
      print('coin %s balance %f' % (to_bal, balances[to_bal]['available']))
    if (data['type']=='TR'):
      print('==================================')
      coin = data['object']['coin']
      print('coin %s balance %f' % (coin,balances[coin]['available']))
      amount = float(data['object']['amount'])
      if data['object']['type']=='deposit':
        print('received new deposit: %f' % amount)
      else:
        print('sent new withdrawal: %f' % amount)
        amount = amount * -1.0
      balances[coin]['available'] += amount
      print('coin %s balance %f' % (coin,balances[coin]['available']))
  
@sio.event
def disconnect():
  print('ws disconnected!')

sio.connect(WS+'?token='+jwtToken)

try:
  while True:
    time.sleep(86400)
except KeyboardInterrupt:
  server.stop(0)