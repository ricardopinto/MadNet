// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

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

// BootNodeClient is the client API for BootNode service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BootNodeClient interface {
	KnownNodes(ctx context.Context, in *BootNodeRequest, opts ...grpc.CallOption) (*BootNodeResponse, error)
}

type bootNodeClient struct {
	cc grpc.ClientConnInterface
}

func NewBootNodeClient(cc grpc.ClientConnInterface) BootNodeClient {
	return &bootNodeClient{cc}
}

func (c *bootNodeClient) KnownNodes(ctx context.Context, in *BootNodeRequest, opts ...grpc.CallOption) (*BootNodeResponse, error) {
	out := new(BootNodeResponse)
	err := c.cc.Invoke(ctx, "/proto.BootNode/KnownNodes", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BootNodeServer is the server API for BootNode service.
// All implementations should embed UnimplementedBootNodeServer
// for forward compatibility
type BootNodeServer interface {
	KnownNodes(context.Context, *BootNodeRequest) (*BootNodeResponse, error)
}

// UnimplementedBootNodeServer should be embedded to have forward compatible implementations.
type UnimplementedBootNodeServer struct {
}

func (UnimplementedBootNodeServer) KnownNodes(context.Context, *BootNodeRequest) (*BootNodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method KnownNodes not implemented")
}

// UnsafeBootNodeServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BootNodeServer will
// result in compilation errors.
type UnsafeBootNodeServer interface {
	mustEmbedUnimplementedBootNodeServer()
}

func RegisterBootNodeServer(s grpc.ServiceRegistrar, srv BootNodeServer) {
	s.RegisterService(&BootNode_ServiceDesc, srv)
}

func _BootNode_KnownNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BootNodeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BootNodeServer).KnownNodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.BootNode/KnownNodes",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BootNodeServer).KnownNodes(ctx, req.(*BootNodeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BootNode_ServiceDesc is the grpc.ServiceDesc for BootNode service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BootNode_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.BootNode",
	HandlerType: (*BootNodeServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "KnownNodes",
			Handler:    _BootNode_KnownNodes_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/bootnode.proto",
}