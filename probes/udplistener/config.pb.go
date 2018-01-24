// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/probes/udplistener/config.proto

/*
Package udplistener is a generated protocol buffer package.

It is generated from these files:
	github.com/google/cloudprober/probes/udplistener/config.proto

It has these top-level messages:
	ProbeConf
*/
package udplistener

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Probe response to an incoming packet: echo back or discard.
type ProbeConf_Type int32

const (
	ProbeConf_INVALID ProbeConf_Type = 0
	ProbeConf_ECHO    ProbeConf_Type = 1
	ProbeConf_DISCARD ProbeConf_Type = 2
)

var ProbeConf_Type_name = map[int32]string{
	0: "INVALID",
	1: "ECHO",
	2: "DISCARD",
}
var ProbeConf_Type_value = map[string]int32{
	"INVALID": 0,
	"ECHO":    1,
	"DISCARD": 2,
}

func (x ProbeConf_Type) Enum() *ProbeConf_Type {
	p := new(ProbeConf_Type)
	*p = x
	return p
}
func (x ProbeConf_Type) String() string {
	return proto.EnumName(ProbeConf_Type_name, int32(x))
}
func (x *ProbeConf_Type) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(ProbeConf_Type_value, data, "ProbeConf_Type")
	if err != nil {
		return err
	}
	*x = ProbeConf_Type(value)
	return nil
}
func (ProbeConf_Type) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

type ProbeConf struct {
	// Export stats after these many milliseconds
	StatsExportIntervalMsec *int32 `protobuf:"varint,2,opt,name=stats_export_interval_msec,json=statsExportIntervalMsec,def=10000" json:"stats_export_interval_msec,omitempty"`
	// Port to listen.
	Port             *int32          `protobuf:"varint,3,opt,name=port,def=32212" json:"port,omitempty"`
	Type             *ProbeConf_Type `protobuf:"varint,4,opt,name=type,enum=cloudprober.probes.udplistener.ProbeConf_Type" json:"type,omitempty"`
	XXX_unrecognized []byte          `json:"-"`
}

func (m *ProbeConf) Reset()                    { *m = ProbeConf{} }
func (m *ProbeConf) String() string            { return proto.CompactTextString(m) }
func (*ProbeConf) ProtoMessage()               {}
func (*ProbeConf) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

const Default_ProbeConf_StatsExportIntervalMsec int32 = 10000
const Default_ProbeConf_Port int32 = 32212

func (m *ProbeConf) GetStatsExportIntervalMsec() int32 {
	if m != nil && m.StatsExportIntervalMsec != nil {
		return *m.StatsExportIntervalMsec
	}
	return Default_ProbeConf_StatsExportIntervalMsec
}

func (m *ProbeConf) GetPort() int32 {
	if m != nil && m.Port != nil {
		return *m.Port
	}
	return Default_ProbeConf_Port
}

func (m *ProbeConf) GetType() ProbeConf_Type {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return ProbeConf_INVALID
}

func init() {
	proto.RegisterType((*ProbeConf)(nil), "cloudprober.probes.udplistener.ProbeConf")
	proto.RegisterEnum("cloudprober.probes.udplistener.ProbeConf_Type", ProbeConf_Type_name, ProbeConf_Type_value)
}

func init() {
	proto.RegisterFile("github.com/google/cloudprober/probes/udplistener/config.proto", fileDescriptor0)
}

var fileDescriptor0 = []byte{
	// 249 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0xce, 0x4f, 0x4b, 0xc3, 0x30,
	0x18, 0xc7, 0x71, 0x3b, 0x33, 0xd4, 0x08, 0x52, 0x72, 0xb1, 0x7a, 0x90, 0xb1, 0xd3, 0xf0, 0x90,
	0x74, 0xf5, 0x26, 0x78, 0xd8, 0xda, 0x81, 0x05, 0xff, 0x51, 0xc5, 0x6b, 0xd9, 0xb2, 0x67, 0xb5,
	0xd0, 0xf5, 0x09, 0x49, 0x2a, 0xee, 0xad, 0xfa, 0x6a, 0x24, 0x99, 0x48, 0x4f, 0x9e, 0x42, 0xbe,
	0x7c, 0x1e, 0xf8, 0xd1, 0xbb, 0xaa, 0xb6, 0x1f, 0xdd, 0x8a, 0x4b, 0xdc, 0x8a, 0x0a, 0xb1, 0x6a,
	0x40, 0xc8, 0x06, 0xbb, 0xb5, 0xd2, 0xb8, 0x02, 0x2d, 0xfc, 0x63, 0x44, 0xb7, 0x56, 0x4d, 0x6d,
	0x2c, 0xb4, 0xa0, 0x85, 0xc4, 0x76, 0x53, 0x57, 0x5c, 0x69, 0xb4, 0xc8, 0xae, 0x7a, 0x98, 0xef,
	0x31, 0xef, 0xe1, 0xf1, 0x77, 0x40, 0x4f, 0x5e, 0x5c, 0x4e, 0xb1, 0xdd, 0xb0, 0x39, 0xbd, 0x34,
	0x76, 0x69, 0x4d, 0x09, 0x5f, 0x0a, 0xb5, 0x2d, 0xeb, 0xd6, 0x82, 0xfe, 0x5c, 0x36, 0xe5, 0xd6,
	0x80, 0x8c, 0x06, 0xa3, 0x60, 0x32, 0xbc, 0x1d, 0x4e, 0xe3, 0x38, 0x8e, 0x8b, 0x73, 0x0f, 0x17,
	0xde, 0xe5, 0xbf, 0xec, 0xd1, 0x80, 0x64, 0x17, 0x94, 0xb8, 0x16, 0x1d, 0xee, 0xf5, 0x4d, 0x92,
	0x4c, 0x93, 0xc2, 0x27, 0x36, 0xa7, 0xc4, 0xee, 0x14, 0x44, 0x64, 0x14, 0x4c, 0xce, 0x12, 0xce,
	0xff, 0xdf, 0xc6, 0xff, 0x76, 0xf1, 0xb7, 0x9d, 0x82, 0xc2, 0xdf, 0x8e, 0xaf, 0x29, 0x71, 0x3f,
	0x76, 0x4a, 0x8f, 0xf2, 0xa7, 0xf7, 0xd9, 0x43, 0x9e, 0x85, 0x07, 0xec, 0x98, 0x92, 0x45, 0x7a,
	0xff, 0x1c, 0x06, 0x2e, 0x67, 0xf9, 0x6b, 0x3a, 0x2b, 0xb2, 0x70, 0xf0, 0x13, 0x00, 0x00, 0xff,
	0xff, 0xcf, 0x7b, 0x77, 0x59, 0x3c, 0x01, 0x00, 0x00,
}
