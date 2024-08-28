// Code generated by protoc-gen-go. DO NOT EDIT.
// source: srv_genghis.proto

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

type ClientHordeReq struct {
	Horde                string   `protobuf:"bytes,1,opt,name=horde,proto3" json:"horde,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ClientHordeReq) Reset()         { *m = ClientHordeReq{} }
func (m *ClientHordeReq) String() string { return proto.CompactTextString(m) }
func (*ClientHordeReq) ProtoMessage()    {}
func (*ClientHordeReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2ea11556d16cdbe, []int{0}
}

func (m *ClientHordeReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ClientHordeReq.Unmarshal(m, b)
}
func (m *ClientHordeReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ClientHordeReq.Marshal(b, m, deterministic)
}
func (m *ClientHordeReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ClientHordeReq.Merge(m, src)
}
func (m *ClientHordeReq) XXX_Size() int {
	return xxx_messageInfo_ClientHordeReq.Size(m)
}
func (m *ClientHordeReq) XXX_DiscardUnknown() {
	xxx_messageInfo_ClientHordeReq.DiscardUnknown(m)
}

var xxx_messageInfo_ClientHordeReq proto.InternalMessageInfo

func (m *ClientHordeReq) GetHorde() string {
	if m != nil {
		return m.Horde
	}
	return ""
}

type NavailReply struct {
	Total                int32    `protobuf:"varint,1,opt,name=total,proto3" json:"total,omitempty"`
	Avail                int32    `protobuf:"varint,2,opt,name=avail,proto3" json:"avail,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NavailReply) Reset()         { *m = NavailReply{} }
func (m *NavailReply) String() string { return proto.CompactTextString(m) }
func (*NavailReply) ProtoMessage()    {}
func (*NavailReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2ea11556d16cdbe, []int{1}
}

func (m *NavailReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NavailReply.Unmarshal(m, b)
}
func (m *NavailReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NavailReply.Marshal(b, m, deterministic)
}
func (m *NavailReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NavailReply.Merge(m, src)
}
func (m *NavailReply) XXX_Size() int {
	return xxx_messageInfo_NavailReply.Size(m)
}
func (m *NavailReply) XXX_DiscardUnknown() {
	xxx_messageInfo_NavailReply.DiscardUnknown(m)
}

var xxx_messageInfo_NavailReply proto.InternalMessageInfo

func (m *NavailReply) GetTotal() int32 {
	if m != nil {
		return m.Total
	}
	return 0
}

func (m *NavailReply) GetAvail() int32 {
	if m != nil {
		return m.Avail
	}
	return 0
}

type AllocHordeReq struct {
	Who                  string   `protobuf:"bytes,1,opt,name=who,proto3" json:"who,omitempty"`
	Name                 string   `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	N                    int32    `protobuf:"varint,3,opt,name=n,proto3" json:"n,omitempty"`
	Err                  *Err     `protobuf:"bytes,4,opt,name=err,proto3" json:"err,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AllocHordeReq) Reset()         { *m = AllocHordeReq{} }
func (m *AllocHordeReq) String() string { return proto.CompactTextString(m) }
func (*AllocHordeReq) ProtoMessage()    {}
func (*AllocHordeReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2ea11556d16cdbe, []int{2}
}

func (m *AllocHordeReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AllocHordeReq.Unmarshal(m, b)
}
func (m *AllocHordeReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AllocHordeReq.Marshal(b, m, deterministic)
}
func (m *AllocHordeReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AllocHordeReq.Merge(m, src)
}
func (m *AllocHordeReq) XXX_Size() int {
	return xxx_messageInfo_AllocHordeReq.Size(m)
}
func (m *AllocHordeReq) XXX_DiscardUnknown() {
	xxx_messageInfo_AllocHordeReq.DiscardUnknown(m)
}

var xxx_messageInfo_AllocHordeReq proto.InternalMessageInfo

func (m *AllocHordeReq) GetWho() string {
	if m != nil {
		return m.Who
	}
	return ""
}

func (m *AllocHordeReq) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *AllocHordeReq) GetN() int32 {
	if m != nil {
		return m.N
	}
	return 0
}

func (m *AllocHordeReq) GetErr() *Err {
	if m != nil {
		return m.Err
	}
	return nil
}

type AllocHordeReply struct {
	Err                  *Err     `protobuf:"bytes,1,opt,name=err,proto3" json:"err,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AllocHordeReply) Reset()         { *m = AllocHordeReply{} }
func (m *AllocHordeReply) String() string { return proto.CompactTextString(m) }
func (*AllocHordeReply) ProtoMessage()    {}
func (*AllocHordeReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2ea11556d16cdbe, []int{3}
}

func (m *AllocHordeReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AllocHordeReply.Unmarshal(m, b)
}
func (m *AllocHordeReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AllocHordeReply.Marshal(b, m, deterministic)
}
func (m *AllocHordeReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AllocHordeReply.Merge(m, src)
}
func (m *AllocHordeReply) XXX_Size() int {
	return xxx_messageInfo_AllocHordeReply.Size(m)
}
func (m *AllocHordeReply) XXX_DiscardUnknown() {
	xxx_messageInfo_AllocHordeReply.DiscardUnknown(m)
}

var xxx_messageInfo_AllocHordeReply proto.InternalMessageInfo

func (m *AllocHordeReply) GetErr() *Err {
	if m != nil {
		return m.Err
	}
	return nil
}

type HordeReply struct {
	Name                 string     `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Nodes                []string   `protobuf:"bytes,2,rep,name=nodes,proto3" json:"nodes,omitempty"`
	Want                 int32      `protobuf:"varint,3,opt,name=want,proto3" json:"want,omitempty"`
	Start                *Timestamp `protobuf:"bytes,4,opt,name=start,proto3" json:"start,omitempty"`
	Req                  string     `protobuf:"bytes,5,opt,name=req,proto3" json:"req,omitempty"`
	Err                  *Err       `protobuf:"bytes,6,opt,name=err,proto3" json:"err,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *HordeReply) Reset()         { *m = HordeReply{} }
func (m *HordeReply) String() string { return proto.CompactTextString(m) }
func (*HordeReply) ProtoMessage()    {}
func (*HordeReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2ea11556d16cdbe, []int{4}
}

func (m *HordeReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_HordeReply.Unmarshal(m, b)
}
func (m *HordeReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_HordeReply.Marshal(b, m, deterministic)
}
func (m *HordeReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_HordeReply.Merge(m, src)
}
func (m *HordeReply) XXX_Size() int {
	return xxx_messageInfo_HordeReply.Size(m)
}
func (m *HordeReply) XXX_DiscardUnknown() {
	xxx_messageInfo_HordeReply.DiscardUnknown(m)
}

var xxx_messageInfo_HordeReply proto.InternalMessageInfo

func (m *HordeReply) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *HordeReply) GetNodes() []string {
	if m != nil {
		return m.Nodes
	}
	return nil
}

func (m *HordeReply) GetWant() int32 {
	if m != nil {
		return m.Want
	}
	return 0
}

func (m *HordeReply) GetStart() *Timestamp {
	if m != nil {
		return m.Start
	}
	return nil
}

func (m *HordeReply) GetReq() string {
	if m != nil {
		return m.Req
	}
	return ""
}

func (m *HordeReply) GetErr() *Err {
	if m != nil {
		return m.Err
	}
	return nil
}

type FlockPost struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Nodes                []string `protobuf:"bytes,2,rep,name=nodes,proto3" json:"nodes,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *FlockPost) Reset()         { *m = FlockPost{} }
func (m *FlockPost) String() string { return proto.CompactTextString(m) }
func (*FlockPost) ProtoMessage()    {}
func (*FlockPost) Descriptor() ([]byte, []int) {
	return fileDescriptor_a2ea11556d16cdbe, []int{5}
}

func (m *FlockPost) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FlockPost.Unmarshal(m, b)
}
func (m *FlockPost) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FlockPost.Marshal(b, m, deterministic)
}
func (m *FlockPost) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FlockPost.Merge(m, src)
}
func (m *FlockPost) XXX_Size() int {
	return xxx_messageInfo_FlockPost.Size(m)
}
func (m *FlockPost) XXX_DiscardUnknown() {
	xxx_messageInfo_FlockPost.DiscardUnknown(m)
}

var xxx_messageInfo_FlockPost proto.InternalMessageInfo

func (m *FlockPost) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *FlockPost) GetNodes() []string {
	if m != nil {
		return m.Nodes
	}
	return nil
}

func init() {
	proto.RegisterType((*ClientHordeReq)(nil), "cruxgen.ClientHordeReq")
	proto.RegisterType((*NavailReply)(nil), "cruxgen.NavailReply")
	proto.RegisterType((*AllocHordeReq)(nil), "cruxgen.AllocHordeReq")
	proto.RegisterType((*AllocHordeReply)(nil), "cruxgen.AllocHordeReply")
	proto.RegisterType((*HordeReply)(nil), "cruxgen.HordeReply")
	proto.RegisterType((*FlockPost)(nil), "cruxgen.FlockPost")
}

func init() { proto.RegisterFile("srv_genghis.proto", fileDescriptor_a2ea11556d16cdbe) }

var fileDescriptor_a2ea11556d16cdbe = []byte{
	// 456 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x53, 0xcd, 0x6e, 0xd3, 0x40,
	0x10, 0x8e, 0xeb, 0x38, 0x25, 0x93, 0xa4, 0x2d, 0x43, 0x04, 0x96, 0x0f, 0x28, 0xf2, 0x01, 0x45,
	0x48, 0x44, 0x50, 0xc4, 0x81, 0x03, 0x12, 0xa8, 0xe2, 0xe7, 0x84, 0x8a, 0x55, 0xce, 0x68, 0xeb,
	0x8e, 0x1c, 0x0b, 0x67, 0xd7, 0xd9, 0xdd, 0xa6, 0xf4, 0x1d, 0x78, 0x0e, 0x9e, 0x13, 0xed, 0xae,
	0xbb, 0x76, 0x22, 0x82, 0x7a, 0x9b, 0x6f, 0xe6, 0xdb, 0x99, 0x6f, 0x7e, 0x16, 0x1e, 0x2a, 0xb9,
	0xf9, 0x51, 0x10, 0x2f, 0x96, 0xa5, 0x5a, 0xd4, 0x52, 0x68, 0x81, 0x87, 0xb9, 0xbc, 0xfe, 0x55,
	0x10, 0x4f, 0x8e, 0x97, 0xc4, 0xa4, 0xbe, 0x24, 0xa6, 0x5d, 0x24, 0x81, 0xba, 0xe4, 0x45, 0x63,
	0x0f, 0x49, 0xca, 0xc6, 0x9c, 0x28, 0x92, 0x9b, 0x32, 0x27, 0x07, 0xd3, 0x67, 0x70, 0x74, 0x56,
	0x95, 0xc4, 0xf5, 0x17, 0x21, 0xaf, 0x28, 0xa3, 0x35, 0x4e, 0x21, 0x5a, 0x1a, 0x3b, 0x0e, 0x66,
	0xc1, 0x7c, 0x98, 0x39, 0x90, 0xbe, 0x85, 0xd1, 0x57, 0xb6, 0x61, 0x65, 0x95, 0x51, 0x5d, 0xdd,
	0x1a, 0x92, 0x16, 0x9a, 0x55, 0x96, 0x14, 0x65, 0x0e, 0x18, 0xaf, 0xe5, 0xc4, 0x07, 0xce, 0x6b,
	0x41, 0x9a, 0xc3, 0xe4, 0x43, 0x55, 0x89, 0xdc, 0x57, 0x38, 0x81, 0xf0, 0x66, 0x29, 0x9a, 0xfc,
	0xc6, 0x44, 0x84, 0x3e, 0x67, 0x2b, 0xb2, 0xef, 0x86, 0x99, 0xb5, 0x71, 0x0c, 0x01, 0x8f, 0x43,
	0x9b, 0x28, 0xe0, 0xf8, 0x14, 0x42, 0x92, 0x32, 0xee, 0xcf, 0x82, 0xf9, 0xe8, 0x74, 0xbc, 0x68,
	0xba, 0x5e, 0x7c, 0x94, 0x32, 0x33, 0x81, 0xf4, 0x15, 0x1c, 0x77, 0x8b, 0x18, 0x8d, 0xcd, 0x93,
	0x60, 0xdf, 0x93, 0x3f, 0x01, 0x40, 0x87, 0x7e, 0xa7, 0x21, 0xe8, 0x68, 0x98, 0x42, 0xc4, 0xc5,
	0x15, 0xa9, 0xf8, 0x60, 0x16, 0x9a, 0x59, 0x58, 0x60, 0x98, 0x37, 0x8c, 0xeb, 0x46, 0x9c, 0xb5,
	0x71, 0x0e, 0x91, 0xd2, 0x4c, 0xea, 0x46, 0x21, 0xfa, 0x72, 0x17, 0xe5, 0x8a, 0x94, 0x66, 0xab,
	0x3a, 0x73, 0x04, 0xd3, 0xbd, 0xa4, 0x75, 0x1c, 0xb9, 0xee, 0x25, 0xad, 0xef, 0x84, 0x0e, 0xf6,
	0x09, 0x7d, 0x03, 0xc3, 0x4f, 0x95, 0xc8, 0x7f, 0x9e, 0x0b, 0xa5, 0xef, 0x2f, 0xf3, 0xf4, 0x77,
	0x08, 0x87, 0x9f, 0xdd, 0xb1, 0xe0, 0x3b, 0x18, 0x75, 0xd6, 0x8c, 0x4f, 0x7c, 0x91, 0xed, 0xe5,
	0x27, 0x8f, 0x7c, 0xa0, 0x9d, 0x4c, 0xda, 0xc3, 0xf7, 0x00, 0xed, 0x74, 0xf1, 0xb1, 0x27, 0x6d,
	0xed, 0x35, 0x89, 0xff, 0xe9, 0x77, 0x19, 0xce, 0x60, 0xfc, 0x9d, 0x77, 0x72, 0xec, 0x55, 0xf0,
	0xbf, 0x24, 0x2f, 0x61, 0xe0, 0x8e, 0x10, 0x8f, 0xda, 0x29, 0xad, 0x6a, 0x7d, 0x9b, 0x4c, 0x3d,
	0xee, 0x5c, 0x69, 0xda, 0xc3, 0x17, 0x10, 0xd9, 0xd1, 0x61, 0xbb, 0x10, 0x3f, 0xca, 0x64, 0x27,
	0x49, 0xda, 0xc3, 0xe7, 0xf0, 0xe0, 0xbc, 0xe4, 0xc5, 0x05, 0x29, 0x8d, 0x13, 0x1f, 0x35, 0xae,
	0x64, 0x1b, 0xa6, 0x3d, 0x5c, 0x40, 0xff, 0xdb, 0x75, 0xa9, 0xf1, 0xc4, 0x07, 0x0c, 0x34, 0x2d,
	0xe0, 0x8e, 0xc7, 0x4a, 0xb9, 0x1c, 0xd8, 0x0f, 0xf7, 0xfa, 0x6f, 0x00, 0x00, 0x00, 0xff, 0xff,
	0x33, 0x6a, 0x72, 0xc4, 0xc5, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// GenghisClient is the client API for Genghis service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type GenghisClient interface {
	ClientHorde(ctx context.Context, in *ClientHordeReq, opts ...grpc.CallOption) (*HordeReply, error)
	AllocHorde(ctx context.Context, in *AllocHordeReq, opts ...grpc.CallOption) (*AllocHordeReply, error)
	UnAllocHorde(ctx context.Context, in *ClientHordeReq, opts ...grpc.CallOption) (*AllocHordeReply, error)
	Navail(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*NavailReply, error)
	Flock(ctx context.Context, in *FlockPost, opts ...grpc.CallOption) (*Empty, error)
	PingTest(ctx context.Context, in *Ping, opts ...grpc.CallOption) (*Ping, error)
	Quit(ctx context.Context, in *QuitReq, opts ...grpc.CallOption) (*QuitReply, error)
}

type genghisClient struct {
	cc *grpc.ClientConn
}

func NewGenghisClient(cc *grpc.ClientConn) GenghisClient {
	return &genghisClient{cc}
}

func (c *genghisClient) ClientHorde(ctx context.Context, in *ClientHordeReq, opts ...grpc.CallOption) (*HordeReply, error) {
	out := new(HordeReply)
	err := c.cc.Invoke(ctx, "/cruxgen.Genghis/ClientHorde", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *genghisClient) AllocHorde(ctx context.Context, in *AllocHordeReq, opts ...grpc.CallOption) (*AllocHordeReply, error) {
	out := new(AllocHordeReply)
	err := c.cc.Invoke(ctx, "/cruxgen.Genghis/AllocHorde", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *genghisClient) UnAllocHorde(ctx context.Context, in *ClientHordeReq, opts ...grpc.CallOption) (*AllocHordeReply, error) {
	out := new(AllocHordeReply)
	err := c.cc.Invoke(ctx, "/cruxgen.Genghis/UnAllocHorde", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *genghisClient) Navail(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*NavailReply, error) {
	out := new(NavailReply)
	err := c.cc.Invoke(ctx, "/cruxgen.Genghis/Navail", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *genghisClient) Flock(ctx context.Context, in *FlockPost, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/cruxgen.Genghis/Flock", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *genghisClient) PingTest(ctx context.Context, in *Ping, opts ...grpc.CallOption) (*Ping, error) {
	out := new(Ping)
	err := c.cc.Invoke(ctx, "/cruxgen.Genghis/PingTest", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *genghisClient) Quit(ctx context.Context, in *QuitReq, opts ...grpc.CallOption) (*QuitReply, error) {
	out := new(QuitReply)
	err := c.cc.Invoke(ctx, "/cruxgen.Genghis/Quit", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GenghisServer is the server API for Genghis service.
type GenghisServer interface {
	ClientHorde(context.Context, *ClientHordeReq) (*HordeReply, error)
	AllocHorde(context.Context, *AllocHordeReq) (*AllocHordeReply, error)
	UnAllocHorde(context.Context, *ClientHordeReq) (*AllocHordeReply, error)
	Navail(context.Context, *Empty) (*NavailReply, error)
	Flock(context.Context, *FlockPost) (*Empty, error)
	PingTest(context.Context, *Ping) (*Ping, error)
	Quit(context.Context, *QuitReq) (*QuitReply, error)
}

func RegisterGenghisServer(s *grpc.Server, srv GenghisServer) {
	s.RegisterService(&_Genghis_serviceDesc, srv)
}

func _Genghis_ClientHorde_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClientHordeReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GenghisServer).ClientHorde(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Genghis/ClientHorde",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GenghisServer).ClientHorde(ctx, req.(*ClientHordeReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Genghis_AllocHorde_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AllocHordeReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GenghisServer).AllocHorde(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Genghis/AllocHorde",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GenghisServer).AllocHorde(ctx, req.(*AllocHordeReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Genghis_UnAllocHorde_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClientHordeReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GenghisServer).UnAllocHorde(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Genghis/UnAllocHorde",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GenghisServer).UnAllocHorde(ctx, req.(*ClientHordeReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Genghis_Navail_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GenghisServer).Navail(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Genghis/Navail",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GenghisServer).Navail(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _Genghis_Flock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FlockPost)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GenghisServer).Flock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Genghis/Flock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GenghisServer).Flock(ctx, req.(*FlockPost))
	}
	return interceptor(ctx, in, info, handler)
}

func _Genghis_PingTest_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Ping)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GenghisServer).PingTest(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Genghis/PingTest",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GenghisServer).PingTest(ctx, req.(*Ping))
	}
	return interceptor(ctx, in, info, handler)
}

func _Genghis_Quit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QuitReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GenghisServer).Quit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/cruxgen.Genghis/Quit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GenghisServer).Quit(ctx, req.(*QuitReq))
	}
	return interceptor(ctx, in, info, handler)
}

var _Genghis_serviceDesc = grpc.ServiceDesc{
	ServiceName: "cruxgen.Genghis",
	HandlerType: (*GenghisServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ClientHorde",
			Handler:    _Genghis_ClientHorde_Handler,
		},
		{
			MethodName: "AllocHorde",
			Handler:    _Genghis_AllocHorde_Handler,
		},
		{
			MethodName: "UnAllocHorde",
			Handler:    _Genghis_UnAllocHorde_Handler,
		},
		{
			MethodName: "Navail",
			Handler:    _Genghis_Navail_Handler,
		},
		{
			MethodName: "Flock",
			Handler:    _Genghis_Flock_Handler,
		},
		{
			MethodName: "PingTest",
			Handler:    _Genghis_PingTest_Handler,
		},
		{
			MethodName: "Quit",
			Handler:    _Genghis_Quit_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "srv_genghis.proto",
}
