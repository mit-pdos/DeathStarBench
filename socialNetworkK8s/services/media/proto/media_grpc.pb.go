// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.12.4
// source: services/media/proto/media.proto

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

const (
	MediaStorage_StoreMedia_FullMethodName = "/media.MediaStorage/StoreMedia"
	MediaStorage_ReadMedia_FullMethodName  = "/media.MediaStorage/ReadMedia"
)

// MediaStorageClient is the client API for MediaStorage service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MediaStorageClient interface {
	StoreMedia(ctx context.Context, in *StoreMediaRequest, opts ...grpc.CallOption) (*StoreMediaResponse, error)
	ReadMedia(ctx context.Context, in *ReadMediaRequest, opts ...grpc.CallOption) (*ReadMediaResponse, error)
}

type mediaStorageClient struct {
	cc grpc.ClientConnInterface
}

func NewMediaStorageClient(cc grpc.ClientConnInterface) MediaStorageClient {
	return &mediaStorageClient{cc}
}

func (c *mediaStorageClient) StoreMedia(ctx context.Context, in *StoreMediaRequest, opts ...grpc.CallOption) (*StoreMediaResponse, error) {
	out := new(StoreMediaResponse)
	err := c.cc.Invoke(ctx, MediaStorage_StoreMedia_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *mediaStorageClient) ReadMedia(ctx context.Context, in *ReadMediaRequest, opts ...grpc.CallOption) (*ReadMediaResponse, error) {
	out := new(ReadMediaResponse)
	err := c.cc.Invoke(ctx, MediaStorage_ReadMedia_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MediaStorageServer is the server API for MediaStorage service.
// All implementations must embed UnimplementedMediaStorageServer
// for forward compatibility
type MediaStorageServer interface {
	StoreMedia(context.Context, *StoreMediaRequest) (*StoreMediaResponse, error)
	ReadMedia(context.Context, *ReadMediaRequest) (*ReadMediaResponse, error)
	mustEmbedUnimplementedMediaStorageServer()
}

// UnimplementedMediaStorageServer must be embedded to have forward compatible implementations.
type UnimplementedMediaStorageServer struct {
}

func (UnimplementedMediaStorageServer) StoreMedia(context.Context, *StoreMediaRequest) (*StoreMediaResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StoreMedia not implemented")
}
func (UnimplementedMediaStorageServer) ReadMedia(context.Context, *ReadMediaRequest) (*ReadMediaResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReadMedia not implemented")
}
func (UnimplementedMediaStorageServer) mustEmbedUnimplementedMediaStorageServer() {}

// UnsafeMediaStorageServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MediaStorageServer will
// result in compilation errors.
type UnsafeMediaStorageServer interface {
	mustEmbedUnimplementedMediaStorageServer()
}

func RegisterMediaStorageServer(s grpc.ServiceRegistrar, srv MediaStorageServer) {
	s.RegisterService(&MediaStorage_ServiceDesc, srv)
}

func _MediaStorage_StoreMedia_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreMediaRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MediaStorageServer).StoreMedia(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MediaStorage_StoreMedia_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MediaStorageServer).StoreMedia(ctx, req.(*StoreMediaRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _MediaStorage_ReadMedia_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadMediaRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MediaStorageServer).ReadMedia(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: MediaStorage_ReadMedia_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MediaStorageServer).ReadMedia(ctx, req.(*ReadMediaRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// MediaStorage_ServiceDesc is the grpc.ServiceDesc for MediaStorage service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MediaStorage_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "media.MediaStorage",
	HandlerType: (*MediaStorageServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "StoreMedia",
			Handler:    _MediaStorage_StoreMedia_Handler,
		},
		{
			MethodName: "ReadMedia",
			Handler:    _MediaStorage_ReadMedia_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "services/media/proto/media.proto",
}