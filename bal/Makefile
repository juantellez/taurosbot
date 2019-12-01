.PHONY: pb env

pb:
	python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. balance.proto

lnpb:
	ln -s ../proto/balance.proto balance.proto

