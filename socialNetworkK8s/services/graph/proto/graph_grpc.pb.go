// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.12.4
// source: services/graph/proto/graph.proto

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
	Graph_GetFollowers_FullMethodName      = "/graph.Graph/GetFollowers"
	Graph_GetFollowees_FullMethodName      = "/graph.Graph/GetFollowees"
	Graph_Follow_FullMethodName            = "/graph.Graph/Follow"
	Graph_Unfollow_FullMethodName          = "/graph.Graph/Unfollow"
	Graph_FollowWithUname_FullMethodName   = "/graph.Graph/FollowWithUname"
	Graph_UnfollowWithUname_FullMethodName = "/graph.Graph/UnfollowWithUname"
)

// GraphClient is the client API for Graph service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GraphClient interface {
	GetFollowers(ctx context.Context, in *GetFollowersRequest, opts ...grpc.CallOption) (*GraphGetResponse, error)
	GetFollowees(ctx context.Context, in *GetFolloweesRequest, opts ...grpc.CallOption) (*GraphGetResponse, error)
	Follow(ctx context.Context, in *FollowRequest, opts ...grpc.CallOption) (*GraphUpdateResponse, error)
	Unfollow(ctx context.Context, in *UnfollowRequest, opts ...grpc.CallOption) (*GraphUpdateResponse, error)
	FollowWithUname(ctx context.Context, in *FollowWithUnameRequest, opts ...grpc.CallOption) (*GraphUpdateResponse, error)
	UnfollowWithUname(ctx context.Context, in *UnfollowWithUnameRequest, opts ...grpc.CallOption) (*GraphUpdateResponse, error)
}

type graphClient struct {
	cc grpc.ClientConnInterface
}

func NewGraphClient(cc grpc.ClientConnInterface) GraphClient {
	return &graphClient{cc}
}

func (c *graphClient) GetFollowers(ctx context.Context, in *GetFollowersRequest, opts ...grpc.CallOption) (*GraphGetResponse, error) {
	out := new(GraphGetResponse)
	err := c.cc.Invoke(ctx, Graph_GetFollowers_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *graphClient) GetFollowees(ctx context.Context, in *GetFolloweesRequest, opts ...grpc.CallOption) (*GraphGetResponse, error) {
	out := new(GraphGetResponse)
	err := c.cc.Invoke(ctx, Graph_GetFollowees_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *graphClient) Follow(ctx context.Context, in *FollowRequest, opts ...grpc.CallOption) (*GraphUpdateResponse, error) {
	out := new(GraphUpdateResponse)
	err := c.cc.Invoke(ctx, Graph_Follow_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *graphClient) Unfollow(ctx context.Context, in *UnfollowRequest, opts ...grpc.CallOption) (*GraphUpdateResponse, error) {
	out := new(GraphUpdateResponse)
	err := c.cc.Invoke(ctx, Graph_Unfollow_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *graphClient) FollowWithUname(ctx context.Context, in *FollowWithUnameRequest, opts ...grpc.CallOption) (*GraphUpdateResponse, error) {
	out := new(GraphUpdateResponse)
	err := c.cc.Invoke(ctx, Graph_FollowWithUname_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *graphClient) UnfollowWithUname(ctx context.Context, in *UnfollowWithUnameRequest, opts ...grpc.CallOption) (*GraphUpdateResponse, error) {
	out := new(GraphUpdateResponse)
	err := c.cc.Invoke(ctx, Graph_UnfollowWithUname_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GraphServer is the server API for Graph service.
// All implementations must embed UnimplementedGraphServer
// for forward compatibility
type GraphServer interface {
	GetFollowers(context.Context, *GetFollowersRequest) (*GraphGetResponse, error)
	GetFollowees(context.Context, *GetFolloweesRequest) (*GraphGetResponse, error)
	Follow(context.Context, *FollowRequest) (*GraphUpdateResponse, error)
	Unfollow(context.Context, *UnfollowRequest) (*GraphUpdateResponse, error)
	FollowWithUname(context.Context, *FollowWithUnameRequest) (*GraphUpdateResponse, error)
	UnfollowWithUname(context.Context, *UnfollowWithUnameRequest) (*GraphUpdateResponse, error)
	mustEmbedUnimplementedGraphServer()
}

// UnimplementedGraphServer must be embedded to have forward compatible implementations.
type UnimplementedGraphServer struct {
}

func (UnimplementedGraphServer) GetFollowers(context.Context, *GetFollowersRequest) (*GraphGetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetFollowers not implemented")
}
func (UnimplementedGraphServer) GetFollowees(context.Context, *GetFolloweesRequest) (*GraphGetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetFollowees not implemented")
}
func (UnimplementedGraphServer) Follow(context.Context, *FollowRequest) (*GraphUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Follow not implemented")
}
func (UnimplementedGraphServer) Unfollow(context.Context, *UnfollowRequest) (*GraphUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Unfollow not implemented")
}
func (UnimplementedGraphServer) FollowWithUname(context.Context, *FollowWithUnameRequest) (*GraphUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FollowWithUname not implemented")
}
func (UnimplementedGraphServer) UnfollowWithUname(context.Context, *UnfollowWithUnameRequest) (*GraphUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnfollowWithUname not implemented")
}
func (UnimplementedGraphServer) mustEmbedUnimplementedGraphServer() {}

// UnsafeGraphServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GraphServer will
// result in compilation errors.
type UnsafeGraphServer interface {
	mustEmbedUnimplementedGraphServer()
}

func RegisterGraphServer(s grpc.ServiceRegistrar, srv GraphServer) {
	s.RegisterService(&Graph_ServiceDesc, srv)
}

func _Graph_GetFollowers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetFollowersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GraphServer).GetFollowers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Graph_GetFollowers_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GraphServer).GetFollowers(ctx, req.(*GetFollowersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Graph_GetFollowees_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetFolloweesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GraphServer).GetFollowees(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Graph_GetFollowees_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GraphServer).GetFollowees(ctx, req.(*GetFolloweesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Graph_Follow_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FollowRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GraphServer).Follow(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Graph_Follow_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GraphServer).Follow(ctx, req.(*FollowRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Graph_Unfollow_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnfollowRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GraphServer).Unfollow(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Graph_Unfollow_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GraphServer).Unfollow(ctx, req.(*UnfollowRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Graph_FollowWithUname_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FollowWithUnameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GraphServer).FollowWithUname(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Graph_FollowWithUname_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GraphServer).FollowWithUname(ctx, req.(*FollowWithUnameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Graph_UnfollowWithUname_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnfollowWithUnameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GraphServer).UnfollowWithUname(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Graph_UnfollowWithUname_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GraphServer).UnfollowWithUname(ctx, req.(*UnfollowWithUnameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Graph_ServiceDesc is the grpc.ServiceDesc for Graph service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Graph_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "graph.Graph",
	HandlerType: (*GraphServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetFollowers",
			Handler:    _Graph_GetFollowers_Handler,
		},
		{
			MethodName: "GetFollowees",
			Handler:    _Graph_GetFollowees_Handler,
		},
		{
			MethodName: "Follow",
			Handler:    _Graph_Follow_Handler,
		},
		{
			MethodName: "Unfollow",
			Handler:    _Graph_Unfollow_Handler,
		},
		{
			MethodName: "FollowWithUname",
			Handler:    _Graph_FollowWithUname_Handler,
		},
		{
			MethodName: "UnfollowWithUname",
			Handler:    _Graph_UnfollowWithUname_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "services/graph/proto/graph.proto",
}