.PHONY: tb ox gdax docker all testdockertb
tb:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/tb tb/*.go

ox:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/ox openxrate/*.go

gdax:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/gdax gdax/*.go

dockertb: tb
	docker build -f Dockerfile.tb -t taurosbot/tb .

dockerox:
	docker build -f Dockerfile.ox -t taurosbot/ox .

dockergdax:
	docker build -f Dockerfile.gdax -t taurosbot/gdax .

all: tb ox gdax dockertb dockerox dockergdax 
