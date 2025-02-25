// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: proto/v1beta/sink.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// SinkServiceClient is the client API for SinkService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type SinkServiceClient interface {
	Send(ctx context.Context, opts ...grpc.CallOption) (SinkService_SendClient, error)
}

type sinkServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewSinkServiceClient(cc grpc.ClientConnInterface) SinkServiceClient {
	return &sinkServiceClient{cc}
}

func (c *sinkServiceClient) Send(ctx context.Context, opts ...grpc.CallOption) (SinkService_SendClient, error) {
	stream, err := c.cc.NewStream(ctx, &SinkService_ServiceDesc.Streams[0], "/proto.v1beta.SinkService/Send", opts...)
	if err != nil {
		return nil, err
	}
	x := &sinkServiceSendClient{stream}
	return x, nil
}

type SinkService_SendClient interface {
	Send(*SendRequest) error
	CloseAndRecv() (*SendResponse, error)
	grpc.ClientStream
}

type sinkServiceSendClient struct {
	grpc.ClientStream
}

func (x *sinkServiceSendClient) Send(m *SendRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *sinkServiceSendClient) CloseAndRecv() (*SendResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(SendResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// SinkServiceServer is the server API for SinkService service.
// All implementations must embed UnimplementedSinkServiceServer
// for forward compatibility
type SinkServiceServer interface {
	Send(SinkService_SendServer) error
	mustEmbedUnimplementedSinkServiceServer()
}

// UnimplementedSinkServiceServer must be embedded to have forward compatible implementations.
type UnimplementedSinkServiceServer struct {
}

func (UnimplementedSinkServiceServer) Send(SinkService_SendServer) error {
	return status.Errorf(codes.Unimplemented, "method Send not implemented")
}
func (UnimplementedSinkServiceServer) mustEmbedUnimplementedSinkServiceServer() {}

// UnsafeSinkServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to SinkServiceServer will
// result in compilation errors.
type UnsafeSinkServiceServer interface {
	mustEmbedUnimplementedSinkServiceServer()
}

func RegisterSinkServiceServer(s grpc.ServiceRegistrar, srv SinkServiceServer) {
	s.RegisterService(&SinkService_ServiceDesc, srv)
}

func _SinkService_Send_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(SinkServiceServer).Send(&sinkServiceSendServer{stream})
}

type SinkService_SendServer interface {
	SendAndClose(*SendResponse) error
	Recv() (*SendRequest, error)
	grpc.ServerStream
}

type sinkServiceSendServer struct {
	grpc.ServerStream
}

func (x *sinkServiceSendServer) SendAndClose(m *SendResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *sinkServiceSendServer) Recv() (*SendRequest, error) {
	m := new(SendRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// SinkService_ServiceDesc is the grpc.ServiceDesc for SinkService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var SinkService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.v1beta.SinkService",
	HandlerType: (*SinkServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Send",
			Handler:       _SinkService_Send_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "proto/v1beta/sink.proto",
}
