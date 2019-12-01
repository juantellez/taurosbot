.PHONY: tb ox gdax docker all testdockertb
tb:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/tb taurosbot/*.go

ox:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/ox openxrate/*.go

gdax:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o bin/gdax gdax/*.go

dockertb:
	docker build -f Dockerfile.tb -t taurosbot/tb .

dockerox:
	docker build -f Dockerfile.ox -t taurosbot/ox .

dockergdax:
	docker build -f Dockerfile.gdax -t taurosbot/gdax .

dockerbal:
	docker build -f Dockerfile.bal -t taurosbot/bal .

all: tb ox gdax dockertb dockerox dockergdax dockerbal
