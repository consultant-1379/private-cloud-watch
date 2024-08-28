// Code generated by protoc-gen-go. DO NOT EDIT.
// source: service.proto

package cruxgen

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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

type Empty struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Empty) Reset()         { *m = Empty{} }
func (m *Empty) String() string { return proto.CompactTextString(m) }
func (*Empty) ProtoMessage()    {}
func (*Empty) Descriptor() ([]byte, []int) {
	return fileDescriptor_a0b84a42fa06f626, []int{0}
}

func (m *Empty) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Empty.Unmarshal(m, b)
}
func (m *Empty) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Empty.Marshal(b, m, deterministic)
}
func (m *Empty) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Empty.Merge(m, src)
}
func (m *Empty) XXX_Size() int {
	return xxx_messageInfo_Empty.Size(m)
}
func (m *Empty) XXX_DiscardUnknown() {
	xxx_messageInfo_Empty.DiscardUnknown(m)
}

var xxx_messageInfo_Empty proto.InternalMessageInfo

type QuitReq struct {
	Delay                *Timestamp `protobuf:"bytes,1,opt,name=delay,proto3" json:"delay,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *QuitReq) Reset()         { *m = QuitReq{} }
func (m *QuitReq) String() string { return proto.CompactTextString(m) }
func (*QuitReq) ProtoMessage()    {}
func (*QuitReq) Descriptor() ([]byte, []int) {
	return fileDescriptor_a0b84a42fa06f626, []int{1}
}

func (m *QuitReq) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_QuitReq.Unmarshal(m, b)
}
func (m *QuitReq) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_QuitReq.Marshal(b, m, deterministic)
}
func (m *QuitReq) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QuitReq.Merge(m, src)
}
func (m *QuitReq) XXX_Size() int {
	return xxx_messageInfo_QuitReq.Size(m)
}
func (m *QuitReq) XXX_DiscardUnknown() {
	xxx_messageInfo_QuitReq.DiscardUnknown(m)
}

var xxx_messageInfo_QuitReq proto.InternalMessageInfo

func (m *QuitReq) GetDelay() *Timestamp {
	if m != nil {
		return m.Delay
	}
	return nil
}

type QuitReply struct {
	Message              string   `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *QuitReply) Reset()         { *m = QuitReply{} }
func (m *QuitReply) String() string { return proto.CompactTextString(m) }
func (*QuitReply) ProtoMessage()    {}
func (*QuitReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_a0b84a42fa06f626, []int{2}
}

func (m *QuitReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_QuitReply.Unmarshal(m, b)
}
func (m *QuitReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_QuitReply.Marshal(b, m, deterministic)
}
func (m *QuitReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QuitReply.Merge(m, src)
}
func (m *QuitReply) XXX_Size() int {
	return xxx_messageInfo_QuitReply.Size(m)
}
func (m *QuitReply) XXX_DiscardUnknown() {
	xxx_messageInfo_QuitReply.DiscardUnknown(m)
}

var xxx_messageInfo_QuitReply proto.InternalMessageInfo

func (m *QuitReply) GetMessage() string {
	if m != nil {
		return m.Message
	}
	return ""
}

func init() {
	proto.RegisterType((*Empty)(nil), "cruxgen.Empty")
	proto.RegisterType((*QuitReq)(nil), "cruxgen.QuitReq")
	proto.RegisterType((*QuitReply)(nil), "cruxgen.QuitReply")
}

func init() { proto.RegisterFile("service.proto", fileDescriptor_a0b84a42fa06f626) }

var fileDescriptor_a0b84a42fa06f626 = []byte{
	// 145 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2d, 0x4e, 0x2d, 0x2a,
	0xcb, 0x4c, 0x4e, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x4f, 0x2e, 0x2a, 0xad, 0x48,
	0x4f, 0xcd, 0x93, 0xe2, 0xcf, 0x48, 0x4d, 0x2c, 0x2a, 0x49, 0x4a, 0x4d, 0x2c, 0x81, 0xc8, 0x28,
	0xb1, 0x73, 0xb1, 0xba, 0xe6, 0x16, 0x94, 0x54, 0x2a, 0x19, 0x73, 0xb1, 0x07, 0x96, 0x66, 0x96,
	0x04, 0xa5, 0x16, 0x0a, 0x69, 0x70, 0xb1, 0xa6, 0xa4, 0xe6, 0x24, 0x56, 0x4a, 0x30, 0x2a, 0x30,
	0x6a, 0x70, 0x1b, 0x09, 0xe9, 0x41, 0x75, 0xeb, 0x85, 0x64, 0xe6, 0xa6, 0x16, 0x97, 0x24, 0xe6,
	0x16, 0x04, 0x41, 0x14, 0x28, 0xa9, 0x72, 0x71, 0x42, 0x34, 0x15, 0xe4, 0x54, 0x0a, 0x49, 0x70,
	0xb1, 0xe7, 0xa6, 0x16, 0x17, 0x27, 0xa6, 0xa7, 0x82, 0x35, 0x72, 0x06, 0xc1, 0xb8, 0x49, 0x6c,
	0x60, 0xbb, 0x8c, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0xe5, 0xcc, 0x93, 0x43, 0x96, 0x00, 0x00,
	0x00,
}
