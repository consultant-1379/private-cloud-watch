// Code generated by protoc-gen-go. DO NOT EDIT.
// source: srv_yurt.proto

package cruxgen

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

func init() { proto.RegisterFile("srv_yurt.proto", fileDescriptor_3dce354fadd8b998) }

var fileDescriptor_3dce354fadd8b998 = []byte{
	// 188 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2b, 0x2e, 0x2a, 0x8b,
	0xaf, 0x2c, 0x2d, 0x2a, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x4f, 0x2e, 0x2a, 0xad,
	0x48, 0x4f, 0xcd, 0x93, 0xe2, 0x2a, 0xc8, 0xcc, 0x4b, 0x87, 0x08, 0x4a, 0x89, 0x82, 0x14, 0x65,
	0xa4, 0x26, 0xe6, 0x94, 0x64, 0x24, 0x67, 0xa4, 0x26, 0x67, 0x43, 0x85, 0x79, 0x8b, 0x53, 0x8b,
	0xca, 0x32, 0x93, 0x53, 0x21, 0x5c, 0xa3, 0x19, 0x8c, 0x5c, 0x2c, 0x91, 0xa5, 0x45, 0x25, 0x42,
	0x26, 0x5c, 0xcc, 0xe1, 0x19, 0xf9, 0x42, 0x7c, 0x7a, 0x50, 0xb3, 0xf4, 0x5c, 0x73, 0x0b, 0x4a,
	0x2a, 0xa5, 0xa4, 0xe1, 0x7c, 0x8f, 0xd4, 0xc4, 0xa2, 0x92, 0xa4, 0xd4, 0xc4, 0x92, 0xe2, 0xa0,
	0xd4, 0xe2, 0x82, 0xfc, 0xbc, 0xe2, 0x54, 0x25, 0x06, 0x21, 0x2d, 0x2e, 0x8e, 0x80, 0xcc, 0xbc,
	0xf4, 0x90, 0xd4, 0xe2, 0x12, 0x21, 0x5e, 0xb8, 0x52, 0x90, 0x90, 0x14, 0x2a, 0x57, 0x89, 0x41,
	0x48, 0x8f, 0x8b, 0x25, 0xb0, 0x34, 0xb3, 0x44, 0x48, 0x00, 0x2e, 0x01, 0xe2, 0x06, 0xa5, 0x16,
	0x4a, 0x09, 0xa1, 0x89, 0x14, 0xe4, 0x54, 0x2a, 0x31, 0x24, 0xb1, 0x81, 0x5d, 0x68, 0x0c, 0x08,
	0x00, 0x00, 0xff, 0xff, 0xa8, 0xec, 0xcd, 0x15, 0xee, 0x00, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// YurtClient is the client API for Yurt service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type YurtClient interface {
	Who(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*HeartbeatsResponse, error)
	PingTest(ctx context.Context, in *Ping, opts ...grpc.CallOption) (*Ping, error)
	Quit(ctx context.Context, in *QuitReq, opts ...grpc.CallOption) (*QuitReply, error)
}

type yurtClient struct {
	cc *grpc.ClientConn
}

func NewYurtClient(cc *grpc.ClientConn) YurtClient {
	return &yurtClient{cc}
}

func (c *yurtClient) Who(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*HeartbeatsResponse, error) {
	out := new(HeartbeatsResponse)
	err := c.cc.Invoke(ctx, "/cruxgen.Yurt/Who", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yurtClient) PingTest(ctx context.Context, in *Ping, opts ...grpc.CallOption) (*Ping, error) {
	out := new(Ping)
	err := c.cc.Invoke(ctx, "/cruxgen.Yurt/PingTest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *yurtClient) Quit(ctx context.Context, in *QuitReq, opts ...grpc.CallOption) (*QuitReply, error) {
	out := new(QuitReply)
	err := c.cc.Invoke(ctx, "/cruxgen.Yurt/Quit", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// YurtServer is the server API for Yurt service.
type YurtServer interface {
	Who(context.Context, *Empty) (*HeartbeatsResponse, error)
	PingTest(context.Context, *Ping) (*Ping, error)
	Quit(context.Context, *QuitReq) (*QuitReply, error)
}

func RegisterYurtServer(s *grpc.Server, srv YurtServer) {
	s.RegisterService(&_Yurt_serviceDesc, srv)
}

func _Yurt_Who_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YurtServer).Who(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Yurt/Who",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YurtServer).Who(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Yurt_PingTest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Ping)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YurtServer).PingTest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Yurt/PingTest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YurtServer).PingTest(ctx, req.(*Ping))
	}
	return interceptor(ctx, in, info, handler)
}

func _Yurt_Quit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuitReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(YurtServer).Quit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Yurt/Quit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(YurtServer).Quit(ctx, req.(*QuitReq))
	}
	return interceptor(ctx, in, info, handler)
}

var _Yurt_serviceDesc = grpc.ServiceDesc{
	ServiceName: "cruxgen.Yurt",
	HandlerType: (*YurtServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Who",
			Handler:    _Yurt_Who_Handler,
		},
		{
			MethodName: "PingTest",
			Handler:    _Yurt_PingTest_Handler,
		},
		{
			MethodName: "Quit",
			Handler:    _Yurt_Quit_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "srv_yurt.proto",
}
