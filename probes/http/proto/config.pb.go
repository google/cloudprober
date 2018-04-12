// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/probes/http/proto/config.proto

/*
Package proto is a generated protocol buffer package.

It is generated from these files:
	github.com/google/cloudprober/probes/http/proto/config.proto

It has these top-level messages:
	ProbeConf
*/
package proto

import proto1 "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto1.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto1.ProtoPackageIsVersion2 // please upgrade the proto package

type ProbeConf_ProtocolType int32

const (
	ProbeConf_HTTP  ProbeConf_ProtocolType = 0
	ProbeConf_HTTPS ProbeConf_ProtocolType = 1
)

var ProbeConf_ProtocolType_name = map[int32]string{
	0: "HTTP",
	1: "HTTPS",
}
var ProbeConf_ProtocolType_value = map[string]int32{
	"HTTP":  0,
	"HTTPS": 1,
}

func (x ProbeConf_ProtocolType) Enum() *ProbeConf_ProtocolType {
	p := new(ProbeConf_ProtocolType)
	*p = x
	return p
}
func (x ProbeConf_ProtocolType) String() string {
	return proto1.EnumName(ProbeConf_ProtocolType_name, int32(x))
}
func (x *ProbeConf_ProtocolType) UnmarshalJSON(data []byte) error {
	value, err := proto1.UnmarshalJSONEnum(ProbeConf_ProtocolType_value, data, "ProbeConf_ProtocolType")
	if err != nil {
		return err
	}
	*x = ProbeConf_ProtocolType(value)
	return nil
}
func (ProbeConf_ProtocolType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

type ProbeConf struct {
	// Which HTTP protocol to use
	Protocol *ProbeConf_ProtocolType `protobuf:"varint,1,opt,name=protocol,enum=cloudprober.probes.http.ProbeConf_ProtocolType,def=0" json:"protocol,omitempty"`
	// Relative URL (to append to all targets). Must begin with '/'
	RelativeUrl *string `protobuf:"bytes,2,opt,name=relative_url,json=relativeUrl" json:"relative_url,omitempty"`
	// Port, default is 80 for HTTP and 443 for HTTPS
	Port *int32 `protobuf:"varint,3,opt,name=port" json:"port,omitempty"`
	// Whether to resolve the target before making the request. If set to false,
	// we hand over the target and relative_url directly to the golang's HTTP
	// module, Otherwise, we resolve the target first to an IP address and
	// make a request using that while passing target name as Host header.
	ResolveFirst *bool `protobuf:"varint,4,opt,name=resolve_first,json=resolveFirst,def=0" json:"resolve_first,omitempty"`
	// Export response (body) count as a metric
	ExportResponseAsMetrics *bool `protobuf:"varint,5,opt,name=export_response_as_metrics,json=exportResponseAsMetrics,def=0" json:"export_response_as_metrics,omitempty"`
	// If specified, this string is used for payload's integrity check.
	// Note: This feature is experimental and will most likely be replaced by a
	// generic validator framework.
	IntegrityCheckPattern *string `protobuf:"bytes,6,opt,name=integrity_check_pattern,json=integrityCheckPattern" json:"integrity_check_pattern,omitempty"`
	// Requests per probe
	RequestsPerProbe *int32 `protobuf:"varint,98,opt,name=requests_per_probe,json=requestsPerProbe,def=1" json:"requests_per_probe,omitempty"`
	// How long to wait between two requests to the same target
	RequestsIntervalMsec *int32 `protobuf:"varint,99,opt,name=requests_interval_msec,json=requestsIntervalMsec,def=25" json:"requests_interval_msec,omitempty"`
	// Export stats after these many milliseconds
	StatsExportIntervalMsec *int32 `protobuf:"varint,100,opt,name=stats_export_interval_msec,json=statsExportIntervalMsec,def=10000" json:"stats_export_interval_msec,omitempty"`
	XXX_unrecognized        []byte `json:"-"`
}

func (m *ProbeConf) Reset()                    { *m = ProbeConf{} }
func (m *ProbeConf) String() string            { return proto1.CompactTextString(m) }
func (*ProbeConf) ProtoMessage()               {}
func (*ProbeConf) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

const Default_ProbeConf_Protocol ProbeConf_ProtocolType = ProbeConf_HTTP
const Default_ProbeConf_ResolveFirst bool = false
const Default_ProbeConf_ExportResponseAsMetrics bool = false
const Default_ProbeConf_RequestsPerProbe int32 = 1
const Default_ProbeConf_RequestsIntervalMsec int32 = 25
const Default_ProbeConf_StatsExportIntervalMsec int32 = 10000

func (m *ProbeConf) GetProtocol() ProbeConf_ProtocolType {
	if m != nil && m.Protocol != nil {
		return *m.Protocol
	}
	return Default_ProbeConf_Protocol
}

func (m *ProbeConf) GetRelativeUrl() string {
	if m != nil && m.RelativeUrl != nil {
		return *m.RelativeUrl
	}
	return ""
}

func (m *ProbeConf) GetPort() int32 {
	if m != nil && m.Port != nil {
		return *m.Port
	}
	return 0
}

func (m *ProbeConf) GetResolveFirst() bool {
	if m != nil && m.ResolveFirst != nil {
		return *m.ResolveFirst
	}
	return Default_ProbeConf_ResolveFirst
}

func (m *ProbeConf) GetExportResponseAsMetrics() bool {
	if m != nil && m.ExportResponseAsMetrics != nil {
		return *m.ExportResponseAsMetrics
	}
	return Default_ProbeConf_ExportResponseAsMetrics
}

func (m *ProbeConf) GetIntegrityCheckPattern() string {
	if m != nil && m.IntegrityCheckPattern != nil {
		return *m.IntegrityCheckPattern
	}
	return ""
}

func (m *ProbeConf) GetRequestsPerProbe() int32 {
	if m != nil && m.RequestsPerProbe != nil {
		return *m.RequestsPerProbe
	}
	return Default_ProbeConf_RequestsPerProbe
}

func (m *ProbeConf) GetRequestsIntervalMsec() int32 {
	if m != nil && m.RequestsIntervalMsec != nil {
		return *m.RequestsIntervalMsec
	}
	return Default_ProbeConf_RequestsIntervalMsec
}

func (m *ProbeConf) GetStatsExportIntervalMsec() int32 {
	if m != nil && m.StatsExportIntervalMsec != nil {
		return *m.StatsExportIntervalMsec
	}
	return Default_ProbeConf_StatsExportIntervalMsec
}

func init() {
	proto1.RegisterType((*ProbeConf)(nil), "cloudprober.probes.http.ProbeConf")
	proto1.RegisterEnum("cloudprober.probes.http.ProbeConf_ProtocolType", ProbeConf_ProtocolType_name, ProbeConf_ProtocolType_value)
}

func init() {
	proto1.RegisterFile("github.com/google/cloudprober/probes/http/proto/config.proto", fileDescriptor0)
}

var fileDescriptor0 = []byte{
	// 391 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x91, 0xc1, 0x8e, 0xd3, 0x30,
	0x10, 0x86, 0xc9, 0xd2, 0xa0, 0xd6, 0x14, 0x54, 0x59, 0x40, 0x2d, 0x4e, 0x61, 0xb9, 0x44, 0x1c,
	0x92, 0xec, 0x4a, 0x20, 0x54, 0x71, 0x81, 0x15, 0x08, 0x0e, 0x2b, 0x05, 0x53, 0xce, 0x56, 0xea,
	0x9d, 0xa4, 0x11, 0x6e, 0x1c, 0xc6, 0x4e, 0x45, 0x1f, 0x83, 0x37, 0x46, 0x76, 0xd2, 0x28, 0x1c,
	0xf6, 0x94, 0x71, 0xfe, 0xef, 0x1b, 0x79, 0x3c, 0xe4, 0x43, 0x55, 0xdb, 0x7d, 0xb7, 0x4b, 0xa4,
	0x3e, 0xa4, 0x95, 0xd6, 0x95, 0x82, 0x54, 0x2a, 0xdd, 0xdd, 0xb5, 0xa8, 0x77, 0x80, 0xa9, 0xff,
	0x98, 0x74, 0x6f, 0x6d, 0xeb, 0x6a, 0xab, 0x53, 0xa9, 0x9b, 0xb2, 0xae, 0x12, 0x7f, 0xa0, 0xeb,
	0x09, 0x9b, 0xf4, 0x6c, 0xe2, 0xd8, 0xcb, 0xbf, 0x33, 0xb2, 0xc8, 0xdd, 0xf9, 0x46, 0x37, 0x25,
	0xfd, 0x4e, 0xe6, 0x9e, 0x97, 0x5a, 0xb1, 0x20, 0x0a, 0xe2, 0xa7, 0xd7, 0x69, 0x72, 0x8f, 0x99,
	0x8c, 0x96, 0xab, 0xbc, 0xb2, 0x3d, 0xb5, 0xb0, 0x99, 0x7d, 0xdd, 0x6e, 0x73, 0x3e, 0xb6, 0xa1,
	0xaf, 0xc8, 0x12, 0x41, 0x15, 0xb6, 0x3e, 0x82, 0xe8, 0x50, 0xb1, 0x8b, 0x28, 0x88, 0x17, 0xfc,
	0xf1, 0xf9, 0xdf, 0x4f, 0x54, 0x94, 0x92, 0x59, 0xab, 0xd1, 0xb2, 0x87, 0x51, 0x10, 0x87, 0xdc,
	0xd7, 0xf4, 0x0d, 0x79, 0x82, 0x60, 0xb4, 0x3a, 0x82, 0x28, 0x6b, 0x34, 0x96, 0xcd, 0xa2, 0x20,
	0x9e, 0x6f, 0xc2, 0xb2, 0x50, 0x06, 0xf8, 0x72, 0xc8, 0xbe, 0xb8, 0x88, 0x7e, 0x22, 0x2f, 0xe1,
	0x8f, 0xb3, 0x04, 0x82, 0x69, 0x75, 0x63, 0x40, 0x14, 0x46, 0x1c, 0xc0, 0x62, 0x2d, 0x0d, 0x0b,
	0xa7, 0xe2, 0xba, 0x07, 0xf9, 0xc0, 0x7d, 0x34, 0xb7, 0x3d, 0x45, 0xdf, 0x91, 0x75, 0xdd, 0x58,
	0xa8, 0xb0, 0xb6, 0x27, 0x21, 0xf7, 0x20, 0x7f, 0x89, 0xb6, 0xb0, 0x16, 0xb0, 0x61, 0x8f, 0xfc,
	0x8d, 0x9f, 0x8f, 0xf1, 0x8d, 0x4b, 0xf3, 0x3e, 0xa4, 0x29, 0xa1, 0x08, 0xbf, 0x3b, 0x30, 0xd6,
	0x88, 0x16, 0x50, 0xf8, 0x17, 0x62, 0x3b, 0x37, 0xc9, 0x26, 0xb8, 0xe2, 0xab, 0x73, 0x98, 0x03,
	0xfa, 0x07, 0xa3, 0xef, 0xc9, 0x8b, 0x51, 0x70, 0x2d, 0xf1, 0x58, 0x28, 0x71, 0x30, 0x20, 0x99,
	0xf4, 0xd2, 0xc5, 0xf5, 0x5b, 0xfe, 0xec, 0x4c, 0x7c, 0x1b, 0x80, 0x5b, 0x03, 0xd2, 0x8d, 0x69,
	0x6c, 0x61, 0x8d, 0x18, 0x86, 0xfd, 0xdf, 0xbe, 0xf3, 0x76, 0x78, 0x95, 0x65, 0x59, 0xc6, 0xd7,
	0x1e, 0xfc, 0xec, 0xb9, 0x69, 0x8f, 0xcb, 0xd7, 0x64, 0x39, 0xdd, 0x16, 0x9d, 0x13, 0xbf, 0xaf,
	0xd5, 0x03, 0xba, 0x20, 0xa1, 0xab, 0x7e, 0xac, 0x82, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x83,
	0xca, 0xd4, 0xa0, 0x6b, 0x02, 0x00, 0x00,
}
