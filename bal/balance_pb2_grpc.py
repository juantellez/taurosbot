# Generated by the gRPC Python protocol compiler plugin. DO NOT EDIT!
import grpc

import balance_pb2 as balance__pb2


class BalancesServiceStub(object):
  # missing associated documentation comment in .proto file
  pass

  def __init__(self, channel):
    """Constructor.

    Args:
      channel: A grpc.Channel.
    """
    self.GetBalances = channel.unary_unary(
        '/pb.BalancesService/GetBalances',
        request_serializer=balance__pb2.BalancesRequest.SerializeToString,
        response_deserializer=balance__pb2.Balances.FromString,
        )


class BalancesServiceServicer(object):
  # missing associated documentation comment in .proto file
  pass

  def GetBalances(self, request, context):
    # missing associated documentation comment in .proto file
    pass
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')


def add_BalancesServiceServicer_to_server(servicer, server):
  rpc_method_handlers = {
      'GetBalances': grpc.unary_unary_rpc_method_handler(
          servicer.GetBalances,
          request_deserializer=balance__pb2.BalancesRequest.FromString,
          response_serializer=balance__pb2.Balances.SerializeToString,
      ),
  }
  generic_handler = grpc.method_handlers_generic_handler(
      'pb.BalancesService', rpc_method_handlers)
  server.add_generic_rpc_handlers((generic_handler,))