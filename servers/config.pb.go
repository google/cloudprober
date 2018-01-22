// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/servers/config.proto

/*
Package servers is a generated protocol buffer package.

It is generated from these files:
	github.com/google/cloudprober/servers/config.proto

It has these top-level messages:
	Server
*/
package servers

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import cloudprober_servers_http "github.com/google/cloudprober/servers/http"
import cloudprober_servers_udp "github.com/google/cloudprober/servers/udp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Server_Type int32

const (
	Server_HTTP Server_Type = 0
	Server_UDP  Server_Type = 1
)

var Server_Type_name = map[int32]string{
	0: "HTTP",
	1: "UDP",
}
var Server_Type_value = map[string]int32{
	"HTTP": 0,
	"UDP":  1,
}

func (x Server_Type) Enum() *Server_Type {
	p := new(Server_Type)
	*p = x
	return p
}
func (x Server_Type) String() string {
	return proto.EnumName(Server_Type_name, int32(x))
}
func (x *Server_Type) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Server_Type_value, data, "Server_Type")
	if err != nil {
		return err
	}
	*x = Server_Type(value)
	return nil
}
func (Server_Type) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

type Server struct {
	Type *Server_Type `protobuf:"varint,1,req,name=type,enum=cloudprober.servers.Server_Type" json:"type,omitempty"`
	// Types that are valid to be assigned to Server:
	//	*Server_HttpServer
	//	*Server_UdpServer
	Server           isServer_Server `protobuf_oneof:"server"`
	XXX_unrecognized []byte          `json:"-"`
}

func (m *Server) Reset()                    { *m = Server{} }
func (m *Server) String() string            { return proto.CompactTextString(m) }
func (*Server) ProtoMessage()               {}
func (*Server) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type isServer_Server interface{ isServer_Server() }

type Server_HttpServer struct {
	HttpServer *cloudprober_servers_http.ServerConf `protobuf:"bytes,2,opt,name=http_server,json=httpServer,oneof"`
}
type Server_UdpServer struct {
	UdpServer *cloudprober_servers_udp.ServerConf `protobuf:"bytes,3,opt,name=udp_server,json=udpServer,oneof"`
}

func (*Server_HttpServer) isServer_Server() {}
func (*Server_UdpServer) isServer_Server()  {}

func (m *Server) GetServer() isServer_Server {
	if m != nil {
		return m.Server
	}
	return nil
}

func (m *Server) GetType() Server_Type {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return Server_HTTP
}

func (m *Server) GetHttpServer() *cloudprober_servers_http.ServerConf {
	if x, ok := m.GetServer().(*Server_HttpServer); ok {
		return x.HttpServer
	}
	return nil
}

func (m *Server) GetUdpServer() *cloudprober_servers_udp.ServerConf {
	if x, ok := m.GetServer().(*Server_UdpServer); ok {
		return x.UdpServer
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*Server) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _Server_OneofMarshaler, _Server_OneofUnmarshaler, _Server_OneofSizer, []interface{}{
		(*Server_HttpServer)(nil),
		(*Server_UdpServer)(nil),
	}
}

func _Server_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*Server)
	// server
	switch x := m.Server.(type) {
	case *Server_HttpServer:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.HttpServer); err != nil {
			return err
		}
	case *Server_UdpServer:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.UdpServer); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("Server.Server has unexpected type %T", x)
	}
	return nil
}

func _Server_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*Server)
	switch tag {
	case 2: // server.http_server
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(cloudprober_servers_http.ServerConf)
		err := b.DecodeMessage(msg)
		m.Server = &Server_HttpServer{msg}
		return true, err
	case 3: // server.udp_server
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(cloudprober_servers_udp.ServerConf)
		err := b.DecodeMessage(msg)
		m.Server = &Server_UdpServer{msg}
		return true, err
	default:
		return false, nil
	}
}

func _Server_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*Server)
	// server
	switch x := m.Server.(type) {
	case *Server_HttpServer:
		s := proto.Size(x.HttpServer)
		n += proto.SizeVarint(2<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *Server_UdpServer:
		s := proto.Size(x.UdpServer)
		n += proto.SizeVarint(3<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

func init() {
	proto.RegisterType((*Server)(nil), "cloudprober.servers.Server")
	proto.RegisterEnum("cloudprober.servers.Server_Type", Server_Type_name, Server_Type_value)
}

func init() { proto.RegisterFile("github.com/google/cloudprober/servers/config.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 231 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x32, 0x4a, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0x4f, 0xcf, 0xcf, 0x4f, 0xcf, 0x49, 0xd5, 0x4f, 0xce,
	0xc9, 0x2f, 0x4d, 0x29, 0x28, 0xca, 0x4f, 0x4a, 0x2d, 0xd2, 0x2f, 0x4e, 0x2d, 0x2a, 0x4b, 0x2d,
	0x2a, 0xd6, 0x4f, 0xce, 0xcf, 0x4b, 0xcb, 0x4c, 0xd7, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12,
	0x46, 0x52, 0xa1, 0x07, 0x55, 0x21, 0x65, 0x4e, 0x9c, 0x41, 0x19, 0x25, 0x25, 0x05, 0x28, 0xa6,
	0x49, 0x99, 0x11, 0xa7, 0xb1, 0x34, 0x05, 0x55, 0x9f, 0xd2, 0x27, 0x46, 0x2e, 0xb6, 0x60, 0xb0,
	0xa4, 0x90, 0x09, 0x17, 0x4b, 0x49, 0x65, 0x41, 0xaa, 0x04, 0xa3, 0x02, 0x93, 0x06, 0x9f, 0x91,
	0x82, 0x1e, 0x16, 0xf7, 0xe9, 0x41, 0x94, 0xea, 0x85, 0x54, 0x16, 0xa4, 0x06, 0x81, 0x55, 0x0b,
	0xb9, 0x73, 0x71, 0x83, 0x5c, 0x13, 0x0f, 0x51, 0x21, 0xc1, 0xa4, 0xc0, 0xa8, 0xc1, 0x6d, 0xa4,
	0x82, 0x55, 0x33, 0x48, 0x1d, 0xd4, 0x04, 0xe7, 0xfc, 0xbc, 0x34, 0x0f, 0x86, 0x20, 0x2e, 0x90,
	0x10, 0xd4, 0x7a, 0x17, 0x2e, 0xae, 0xd2, 0x14, 0xb8, 0x39, 0xcc, 0x60, 0x73, 0x94, 0xb1, 0x9a,
	0x53, 0x9a, 0x82, 0x66, 0x0c, 0x67, 0x69, 0x0a, 0xd4, 0x14, 0x25, 0x49, 0x2e, 0x16, 0x90, 0xe3,
	0x84, 0x38, 0xb8, 0x58, 0x3c, 0x42, 0x42, 0x02, 0x04, 0x18, 0x84, 0xd8, 0xb9, 0x98, 0x43, 0x5d,
	0x02, 0x04, 0x18, 0x9d, 0x38, 0xb8, 0xd8, 0x20, 0x26, 0x00, 0x02, 0x00, 0x00, 0xff, 0xff, 0x69,
	0xaf, 0x75, 0x31, 0xaf, 0x01, 0x00, 0x00,
}
