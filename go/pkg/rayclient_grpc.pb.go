// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v5.26.1
// source: rayclient.proto

package grpc

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
	GlobalScheduler_Schedule_FullMethodName = "/ray.GlobalScheduler/schedule"
)

// GlobalSchedulerClient is the client API for GlobalScheduler service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GlobalSchedulerClient interface {
	Schedule(ctx context.Context, in *GlobalScheduleRequest, opts ...grpc.CallOption) (*StatusResponse, error)
}

type globalSchedulerClient struct {
	cc grpc.ClientConnInterface
}

func NewGlobalSchedulerClient(cc grpc.ClientConnInterface) GlobalSchedulerClient {
	return &globalSchedulerClient{cc}
}

func (c *globalSchedulerClient) Schedule(ctx context.Context, in *GlobalScheduleRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, GlobalScheduler_Schedule_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GlobalSchedulerServer is the server API for GlobalScheduler service.
// All implementations must embed UnimplementedGlobalSchedulerServer
// for forward compatibility
type GlobalSchedulerServer interface {
	Schedule(context.Context, *GlobalScheduleRequest) (*StatusResponse, error)
	mustEmbedUnimplementedGlobalSchedulerServer()
}

// UnimplementedGlobalSchedulerServer must be embedded to have forward compatible implementations.
type UnimplementedGlobalSchedulerServer struct {
}

func (UnimplementedGlobalSchedulerServer) Schedule(context.Context, *GlobalScheduleRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Schedule not implemented")
}
func (UnimplementedGlobalSchedulerServer) mustEmbedUnimplementedGlobalSchedulerServer() {}

// UnsafeGlobalSchedulerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GlobalSchedulerServer will
// result in compilation errors.
type UnsafeGlobalSchedulerServer interface {
	mustEmbedUnimplementedGlobalSchedulerServer()
}

func RegisterGlobalSchedulerServer(s grpc.ServiceRegistrar, srv GlobalSchedulerServer) {
	s.RegisterService(&GlobalScheduler_ServiceDesc, srv)
}

func _GlobalScheduler_Schedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GlobalScheduleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GlobalSchedulerServer).Schedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GlobalScheduler_Schedule_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GlobalSchedulerServer).Schedule(ctx, req.(*GlobalScheduleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// GlobalScheduler_ServiceDesc is the grpc.ServiceDesc for GlobalScheduler service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var GlobalScheduler_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ray.GlobalScheduler",
	HandlerType: (*GlobalSchedulerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "schedule",
			Handler:    _GlobalScheduler_Schedule_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rayclient.proto",
}

const (
	LocalScheduler_Schedule_FullMethodName = "/ray.LocalScheduler/Schedule"
)

// LocalSchedulerClient is the client API for LocalScheduler service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type LocalSchedulerClient interface {
	Schedule(ctx context.Context, in *ScheduleRequest, opts ...grpc.CallOption) (*ScheduleResponse, error)
}

type localSchedulerClient struct {
	cc grpc.ClientConnInterface
}

func NewLocalSchedulerClient(cc grpc.ClientConnInterface) LocalSchedulerClient {
	return &localSchedulerClient{cc}
}

func (c *localSchedulerClient) Schedule(ctx context.Context, in *ScheduleRequest, opts ...grpc.CallOption) (*ScheduleResponse, error) {
	out := new(ScheduleResponse)
	err := c.cc.Invoke(ctx, LocalScheduler_Schedule_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LocalSchedulerServer is the server API for LocalScheduler service.
// All implementations must embed UnimplementedLocalSchedulerServer
// for forward compatibility
type LocalSchedulerServer interface {
	Schedule(context.Context, *ScheduleRequest) (*ScheduleResponse, error)
	mustEmbedUnimplementedLocalSchedulerServer()
}

// UnimplementedLocalSchedulerServer must be embedded to have forward compatible implementations.
type UnimplementedLocalSchedulerServer struct {
}

func (UnimplementedLocalSchedulerServer) Schedule(context.Context, *ScheduleRequest) (*ScheduleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Schedule not implemented")
}
func (UnimplementedLocalSchedulerServer) mustEmbedUnimplementedLocalSchedulerServer() {}

// UnsafeLocalSchedulerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LocalSchedulerServer will
// result in compilation errors.
type UnsafeLocalSchedulerServer interface {
	mustEmbedUnimplementedLocalSchedulerServer()
}

func RegisterLocalSchedulerServer(s grpc.ServiceRegistrar, srv LocalSchedulerServer) {
	s.RegisterService(&LocalScheduler_ServiceDesc, srv)
}

func _LocalScheduler_Schedule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ScheduleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocalSchedulerServer).Schedule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: LocalScheduler_Schedule_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocalSchedulerServer).Schedule(ctx, req.(*ScheduleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// LocalScheduler_ServiceDesc is the grpc.ServiceDesc for LocalScheduler service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var LocalScheduler_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ray.LocalScheduler",
	HandlerType: (*LocalSchedulerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Schedule",
			Handler:    _LocalScheduler_Schedule_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rayclient.proto",
}

const (
	LocalObjStore_Store_FullMethodName = "/ray.LocalObjStore/Store"
	LocalObjStore_Get_FullMethodName   = "/ray.LocalObjStore/Get"
)

// LocalObjStoreClient is the client API for LocalObjStore service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type LocalObjStoreClient interface {
	Store(ctx context.Context, in *StoreRequest, opts ...grpc.CallOption) (*StatusResponse, error)
	Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error)
}

type localObjStoreClient struct {
	cc grpc.ClientConnInterface
}

func NewLocalObjStoreClient(cc grpc.ClientConnInterface) LocalObjStoreClient {
	return &localObjStoreClient{cc}
}

func (c *localObjStoreClient) Store(ctx context.Context, in *StoreRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, LocalObjStore_Store_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *localObjStoreClient) Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error) {
	out := new(GetResponse)
	err := c.cc.Invoke(ctx, LocalObjStore_Get_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LocalObjStoreServer is the server API for LocalObjStore service.
// All implementations must embed UnimplementedLocalObjStoreServer
// for forward compatibility
type LocalObjStoreServer interface {
	Store(context.Context, *StoreRequest) (*StatusResponse, error)
	Get(context.Context, *GetRequest) (*GetResponse, error)
	mustEmbedUnimplementedLocalObjStoreServer()
}

// UnimplementedLocalObjStoreServer must be embedded to have forward compatible implementations.
type UnimplementedLocalObjStoreServer struct {
}

func (UnimplementedLocalObjStoreServer) Store(context.Context, *StoreRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Store not implemented")
}
func (UnimplementedLocalObjStoreServer) Get(context.Context, *GetRequest) (*GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedLocalObjStoreServer) mustEmbedUnimplementedLocalObjStoreServer() {}

// UnsafeLocalObjStoreServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LocalObjStoreServer will
// result in compilation errors.
type UnsafeLocalObjStoreServer interface {
	mustEmbedUnimplementedLocalObjStoreServer()
}

func RegisterLocalObjStoreServer(s grpc.ServiceRegistrar, srv LocalObjStoreServer) {
	s.RegisterService(&LocalObjStore_ServiceDesc, srv)
}

func _LocalObjStore_Store_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocalObjStoreServer).Store(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: LocalObjStore_Store_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocalObjStoreServer).Store(ctx, req.(*StoreRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _LocalObjStore_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LocalObjStoreServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: LocalObjStore_Get_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LocalObjStoreServer).Get(ctx, req.(*GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// LocalObjStore_ServiceDesc is the grpc.ServiceDesc for LocalObjStore service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var LocalObjStore_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ray.LocalObjStore",
	HandlerType: (*LocalObjStoreServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Store",
			Handler:    _LocalObjStore_Store_Handler,
		},
		{
			MethodName: "Get",
			Handler:    _LocalObjStore_Get_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rayclient.proto",
}

const (
	Worker_Run_FullMethodName = "/ray.Worker/Run"
)

// WorkerClient is the client API for Worker service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WorkerClient interface {
	Run(ctx context.Context, in *RunRequest, opts ...grpc.CallOption) (*StatusResponse, error)
}

type workerClient struct {
	cc grpc.ClientConnInterface
}

func NewWorkerClient(cc grpc.ClientConnInterface) WorkerClient {
	return &workerClient{cc}
}

func (c *workerClient) Run(ctx context.Context, in *RunRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, Worker_Run_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WorkerServer is the server API for Worker service.
// All implementations must embed UnimplementedWorkerServer
// for forward compatibility
type WorkerServer interface {
	Run(context.Context, *RunRequest) (*StatusResponse, error)
	mustEmbedUnimplementedWorkerServer()
}

// UnimplementedWorkerServer must be embedded to have forward compatible implementations.
type UnimplementedWorkerServer struct {
}

func (UnimplementedWorkerServer) Run(context.Context, *RunRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Run not implemented")
}
func (UnimplementedWorkerServer) mustEmbedUnimplementedWorkerServer() {}

// UnsafeWorkerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WorkerServer will
// result in compilation errors.
type UnsafeWorkerServer interface {
	mustEmbedUnimplementedWorkerServer()
}

func RegisterWorkerServer(s grpc.ServiceRegistrar, srv WorkerServer) {
	s.RegisterService(&Worker_ServiceDesc, srv)
}

func _Worker_Run_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WorkerServer).Run(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Worker_Run_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WorkerServer).Run(ctx, req.(*RunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Worker_ServiceDesc is the grpc.ServiceDesc for Worker service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Worker_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ray.Worker",
	HandlerType: (*WorkerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Run",
			Handler:    _Worker_Run_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rayclient.proto",
}

const (
	GCSObj_NotifyOwns_FullMethodName      = "/ray.GCSObj/NotifyOwns"
	GCSObj_RequestLocation_FullMethodName = "/ray.GCSObj/RequestLocation"
)

// GCSObjClient is the client API for GCSObj service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GCSObjClient interface {
	NotifyOwns(ctx context.Context, in *NotifyOwnsRequest, opts ...grpc.CallOption) (*StatusResponse, error)
	RequestLocation(ctx context.Context, in *RequestLocationRequest, opts ...grpc.CallOption) (*StatusResponse, error)
}

type gCSObjClient struct {
	cc grpc.ClientConnInterface
}

func NewGCSObjClient(cc grpc.ClientConnInterface) GCSObjClient {
	return &gCSObjClient{cc}
}

func (c *gCSObjClient) NotifyOwns(ctx context.Context, in *NotifyOwnsRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, GCSObj_NotifyOwns_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gCSObjClient) RequestLocation(ctx context.Context, in *RequestLocationRequest, opts ...grpc.CallOption) (*StatusResponse, error) {
	out := new(StatusResponse)
	err := c.cc.Invoke(ctx, GCSObj_RequestLocation_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GCSObjServer is the server API for GCSObj service.
// All implementations must embed UnimplementedGCSObjServer
// for forward compatibility
type GCSObjServer interface {
	NotifyOwns(context.Context, *NotifyOwnsRequest) (*StatusResponse, error)
	RequestLocation(context.Context, *RequestLocationRequest) (*StatusResponse, error)
	mustEmbedUnimplementedGCSObjServer()
}

// UnimplementedGCSObjServer must be embedded to have forward compatible implementations.
type UnimplementedGCSObjServer struct {
}

func (UnimplementedGCSObjServer) NotifyOwns(context.Context, *NotifyOwnsRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NotifyOwns not implemented")
}
func (UnimplementedGCSObjServer) RequestLocation(context.Context, *RequestLocationRequest) (*StatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestLocation not implemented")
}
func (UnimplementedGCSObjServer) mustEmbedUnimplementedGCSObjServer() {}

// UnsafeGCSObjServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GCSObjServer will
// result in compilation errors.
type UnsafeGCSObjServer interface {
	mustEmbedUnimplementedGCSObjServer()
}

func RegisterGCSObjServer(s grpc.ServiceRegistrar, srv GCSObjServer) {
	s.RegisterService(&GCSObj_ServiceDesc, srv)
}

func _GCSObj_NotifyOwns_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NotifyOwnsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GCSObjServer).NotifyOwns(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GCSObj_NotifyOwns_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GCSObjServer).NotifyOwns(ctx, req.(*NotifyOwnsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GCSObj_RequestLocation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestLocationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GCSObjServer).RequestLocation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GCSObj_RequestLocation_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GCSObjServer).RequestLocation(ctx, req.(*RequestLocationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// GCSObj_ServiceDesc is the grpc.ServiceDesc for GCSObj service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var GCSObj_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ray.GCSObj",
	HandlerType: (*GCSObjServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "NotifyOwns",
			Handler:    _GCSObj_NotifyOwns_Handler,
		},
		{
			MethodName: "RequestLocation",
			Handler:    _GCSObj_RequestLocation_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rayclient.proto",
}

const (
	GCSFunc_RegisterFunc_FullMethodName = "/ray.GCSFunc/RegisterFunc"
	GCSFunc_FetchFunc_FullMethodName    = "/ray.GCSFunc/FetchFunc"
)

// GCSFuncClient is the client API for GCSFunc service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GCSFuncClient interface {
	RegisterFunc(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
	FetchFunc(ctx context.Context, in *FetchRequest, opts ...grpc.CallOption) (*FetchResponse, error)
}

type gCSFuncClient struct {
	cc grpc.ClientConnInterface
}

func NewGCSFuncClient(cc grpc.ClientConnInterface) GCSFuncClient {
	return &gCSFuncClient{cc}
}

func (c *gCSFuncClient) RegisterFunc(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error) {
	out := new(RegisterResponse)
	err := c.cc.Invoke(ctx, GCSFunc_RegisterFunc_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gCSFuncClient) FetchFunc(ctx context.Context, in *FetchRequest, opts ...grpc.CallOption) (*FetchResponse, error) {
	out := new(FetchResponse)
	err := c.cc.Invoke(ctx, GCSFunc_FetchFunc_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GCSFuncServer is the server API for GCSFunc service.
// All implementations must embed UnimplementedGCSFuncServer
// for forward compatibility
type GCSFuncServer interface {
	RegisterFunc(context.Context, *RegisterRequest) (*RegisterResponse, error)
	FetchFunc(context.Context, *FetchRequest) (*FetchResponse, error)
	mustEmbedUnimplementedGCSFuncServer()
}

// UnimplementedGCSFuncServer must be embedded to have forward compatible implementations.
type UnimplementedGCSFuncServer struct {
}

func (UnimplementedGCSFuncServer) RegisterFunc(context.Context, *RegisterRequest) (*RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterFunc not implemented")
}
func (UnimplementedGCSFuncServer) FetchFunc(context.Context, *FetchRequest) (*FetchResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchFunc not implemented")
}
func (UnimplementedGCSFuncServer) mustEmbedUnimplementedGCSFuncServer() {}

// UnsafeGCSFuncServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GCSFuncServer will
// result in compilation errors.
type UnsafeGCSFuncServer interface {
	mustEmbedUnimplementedGCSFuncServer()
}

func RegisterGCSFuncServer(s grpc.ServiceRegistrar, srv GCSFuncServer) {
	s.RegisterService(&GCSFunc_ServiceDesc, srv)
}

func _GCSFunc_RegisterFunc_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GCSFuncServer).RegisterFunc(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GCSFunc_RegisterFunc_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GCSFuncServer).RegisterFunc(ctx, req.(*RegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GCSFunc_FetchFunc_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GCSFuncServer).FetchFunc(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GCSFunc_FetchFunc_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GCSFuncServer).FetchFunc(ctx, req.(*FetchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// GCSFunc_ServiceDesc is the grpc.ServiceDesc for GCSFunc service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var GCSFunc_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ray.GCSFunc",
	HandlerType: (*GCSFuncServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterFunc",
			Handler:    _GCSFunc_RegisterFunc_Handler,
		},
		{
			MethodName: "FetchFunc",
			Handler:    _GCSFunc_FetchFunc_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "rayclient.proto",
}
