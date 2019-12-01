.PHONY: pb

pb:
	protoc --go_out=plugins=grpc:. -I . -I /usr/local/protobuf/include --proto_path . *.proto && sed -i 's/const _/\/\//g' *.pb.go

