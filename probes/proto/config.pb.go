// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/probes/proto/config.proto

/*
Package proto is a generated protocol buffer package.

It is generated from these files:
	github.com/google/cloudprober/probes/proto/config.proto

It has these top-level messages:
	ProbeDef
*/
package proto

import proto1 "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import cloudprober_metrics "github.com/google/cloudprober/metrics"
import cloudprober_probes_http "github.com/google/cloudprober/probes/http/proto"
import cloudprober_probes_dns "github.com/google/cloudprober/probes/dns"
import cloudprober_probes_external "github.com/google/cloudprober/probes/external/proto"
import cloudprober_probes_ping "github.com/google/cloudprober/probes/ping/proto"
import cloudprober_probes_udp "github.com/google/cloudprober/probes/udp"
import cloudprober_probes_udplistener "github.com/google/cloudprober/probes/udplistener"
import cloudprober_targets "github.com/google/cloudprober/targets/proto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto1.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto1.ProtoPackageIsVersion2 // please upgrade the proto package

type ProbeDef_Type int32

const (
	ProbeDef_PING         ProbeDef_Type = 0
	ProbeDef_HTTP         ProbeDef_Type = 1
	ProbeDef_DNS          ProbeDef_Type = 2
	ProbeDef_EXTERNAL     ProbeDef_Type = 3
	ProbeDef_UDP          ProbeDef_Type = 4
	ProbeDef_UDP_LISTENER ProbeDef_Type = 5
	// One of the extension probe types. See "extensions" below for more
	// details.
	ProbeDef_EXTENSION ProbeDef_Type = 98
	// USER_DEFINED probe type is for a one off probe that you want to compile
	// into cloudprober, but you don't expect it to be reused. If you expect
	// it to be reused, you should consider adding it using the extensions
	// mechanism.
	ProbeDef_USER_DEFINED ProbeDef_Type = 99
)

var ProbeDef_Type_name = map[int32]string{
	0:  "PING",
	1:  "HTTP",
	2:  "DNS",
	3:  "EXTERNAL",
	4:  "UDP",
	5:  "UDP_LISTENER",
	98: "EXTENSION",
	99: "USER_DEFINED",
}
var ProbeDef_Type_value = map[string]int32{
	"PING":         0,
	"HTTP":         1,
	"DNS":          2,
	"EXTERNAL":     3,
	"UDP":          4,
	"UDP_LISTENER": 5,
	"EXTENSION":    98,
	"USER_DEFINED": 99,
}

func (x ProbeDef_Type) Enum() *ProbeDef_Type {
	p := new(ProbeDef_Type)
	*p = x
	return p
}
func (x ProbeDef_Type) String() string {
	return proto1.EnumName(ProbeDef_Type_name, int32(x))
}
func (x *ProbeDef_Type) UnmarshalJSON(data []byte) error {
	value, err := proto1.UnmarshalJSONEnum(ProbeDef_Type_value, data, "ProbeDef_Type")
	if err != nil {
		return err
	}
	*x = ProbeDef_Type(value)
	return nil
}
func (ProbeDef_Type) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

type ProbeDef struct {
	Name *string        `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	Type *ProbeDef_Type `protobuf:"varint,2,req,name=type,enum=cloudprober.probes.ProbeDef_Type" json:"type,omitempty"`
	// Which machines this probe should run on. If defined, cloudprober will run
	// this probe only if machine's hostname matches this value.
	RunOn *string `protobuf:"bytes,3,opt,name=run_on,json=runOn" json:"run_on,omitempty"`
	// Interval between two probes
	IntervalMsec *int32 `protobuf:"varint,4,opt,name=interval_msec,json=intervalMsec,def=2000" json:"interval_msec,omitempty"`
	// Timeout for each probe
	TimeoutMsec *int32 `protobuf:"varint,5,opt,name=timeout_msec,json=timeoutMsec,def=1000" json:"timeout_msec,omitempty"`
	// Targets for the probe
	Targets *cloudprober_targets.TargetsDef `protobuf:"bytes,6,req,name=targets" json:"targets,omitempty"`
	// Latency distribution. If specified, latency is stored as a distribution.
	LatencyDistribution *cloudprober_metrics.Dist `protobuf:"bytes,7,opt,name=latency_distribution,json=latencyDistribution" json:"latency_distribution,omitempty"`
	// Latency unit. Any string that's parseable by time.ParseDuration.
	// Valid values: "ns", "us" (or "µs"), "ms", "s", "m", "h".
	LatencyUnit *string `protobuf:"bytes,8,opt,name=latency_unit,json=latencyUnit,def=us" json:"latency_unit,omitempty"`
	// Types that are valid to be assigned to Probe:
	//	*ProbeDef_PingProbe
	//	*ProbeDef_HttpProbe
	//	*ProbeDef_DnsProbe
	//	*ProbeDef_ExternalProbe
	//	*ProbeDef_UdpProbe
	//	*ProbeDef_UdpListenerProbe
	//	*ProbeDef_UserDefinedProbe
	Probe                         isProbeDef_Probe `protobuf_oneof:"probe"`
	proto1.XXX_InternalExtensions `json:"-"`
	XXX_unrecognized              []byte `json:"-"`
}

func (m *ProbeDef) Reset()                    { *m = ProbeDef{} }
func (m *ProbeDef) String() string            { return proto1.CompactTextString(m) }
func (*ProbeDef) ProtoMessage()               {}
func (*ProbeDef) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

var extRange_ProbeDef = []proto1.ExtensionRange{
	{200, 536870911},
}

func (*ProbeDef) ExtensionRangeArray() []proto1.ExtensionRange {
	return extRange_ProbeDef
}

const Default_ProbeDef_IntervalMsec int32 = 2000
const Default_ProbeDef_TimeoutMsec int32 = 1000
const Default_ProbeDef_LatencyUnit string = "us"

type isProbeDef_Probe interface {
	isProbeDef_Probe()
}

type ProbeDef_PingProbe struct {
	PingProbe *cloudprober_probes_ping.ProbeConf `protobuf:"bytes,20,opt,name=ping_probe,json=pingProbe,oneof"`
}
type ProbeDef_HttpProbe struct {
	HttpProbe *cloudprober_probes_http.ProbeConf `protobuf:"bytes,21,opt,name=http_probe,json=httpProbe,oneof"`
}
type ProbeDef_DnsProbe struct {
	DnsProbe *cloudprober_probes_dns.ProbeConf `protobuf:"bytes,22,opt,name=dns_probe,json=dnsProbe,oneof"`
}
type ProbeDef_ExternalProbe struct {
	ExternalProbe *cloudprober_probes_external.ProbeConf `protobuf:"bytes,23,opt,name=external_probe,json=externalProbe,oneof"`
}
type ProbeDef_UdpProbe struct {
	UdpProbe *cloudprober_probes_udp.ProbeConf `protobuf:"bytes,24,opt,name=udp_probe,json=udpProbe,oneof"`
}
type ProbeDef_UdpListenerProbe struct {
	UdpListenerProbe *cloudprober_probes_udplistener.ProbeConf `protobuf:"bytes,25,opt,name=udp_listener_probe,json=udpListenerProbe,oneof"`
}
type ProbeDef_UserDefinedProbe struct {
	UserDefinedProbe string `protobuf:"bytes,99,opt,name=user_defined_probe,json=userDefinedProbe,oneof"`
}

func (*ProbeDef_PingProbe) isProbeDef_Probe()        {}
func (*ProbeDef_HttpProbe) isProbeDef_Probe()        {}
func (*ProbeDef_DnsProbe) isProbeDef_Probe()         {}
func (*ProbeDef_ExternalProbe) isProbeDef_Probe()    {}
func (*ProbeDef_UdpProbe) isProbeDef_Probe()         {}
func (*ProbeDef_UdpListenerProbe) isProbeDef_Probe() {}
func (*ProbeDef_UserDefinedProbe) isProbeDef_Probe() {}

func (m *ProbeDef) GetProbe() isProbeDef_Probe {
	if m != nil {
		return m.Probe
	}
	return nil
}

func (m *ProbeDef) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *ProbeDef) GetType() ProbeDef_Type {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return ProbeDef_PING
}

func (m *ProbeDef) GetRunOn() string {
	if m != nil && m.RunOn != nil {
		return *m.RunOn
	}
	return ""
}

func (m *ProbeDef) GetIntervalMsec() int32 {
	if m != nil && m.IntervalMsec != nil {
		return *m.IntervalMsec
	}
	return Default_ProbeDef_IntervalMsec
}

func (m *ProbeDef) GetTimeoutMsec() int32 {
	if m != nil && m.TimeoutMsec != nil {
		return *m.TimeoutMsec
	}
	return Default_ProbeDef_TimeoutMsec
}

func (m *ProbeDef) GetTargets() *cloudprober_targets.TargetsDef {
	if m != nil {
		return m.Targets
	}
	return nil
}

func (m *ProbeDef) GetLatencyDistribution() *cloudprober_metrics.Dist {
	if m != nil {
		return m.LatencyDistribution
	}
	return nil
}

func (m *ProbeDef) GetLatencyUnit() string {
	if m != nil && m.LatencyUnit != nil {
		return *m.LatencyUnit
	}
	return Default_ProbeDef_LatencyUnit
}

func (m *ProbeDef) GetPingProbe() *cloudprober_probes_ping.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_PingProbe); ok {
		return x.PingProbe
	}
	return nil
}

func (m *ProbeDef) GetHttpProbe() *cloudprober_probes_http.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_HttpProbe); ok {
		return x.HttpProbe
	}
	return nil
}

func (m *ProbeDef) GetDnsProbe() *cloudprober_probes_dns.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_DnsProbe); ok {
		return x.DnsProbe
	}
	return nil
}

func (m *ProbeDef) GetExternalProbe() *cloudprober_probes_external.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_ExternalProbe); ok {
		return x.ExternalProbe
	}
	return nil
}

func (m *ProbeDef) GetUdpProbe() *cloudprober_probes_udp.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_UdpProbe); ok {
		return x.UdpProbe
	}
	return nil
}

func (m *ProbeDef) GetUdpListenerProbe() *cloudprober_probes_udplistener.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_UdpListenerProbe); ok {
		return x.UdpListenerProbe
	}
	return nil
}

func (m *ProbeDef) GetUserDefinedProbe() string {
	if x, ok := m.GetProbe().(*ProbeDef_UserDefinedProbe); ok {
		return x.UserDefinedProbe
	}
	return ""
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*ProbeDef) XXX_OneofFuncs() (func(msg proto1.Message, b *proto1.Buffer) error, func(msg proto1.Message, tag, wire int, b *proto1.Buffer) (bool, error), func(msg proto1.Message) (n int), []interface{}) {
	return _ProbeDef_OneofMarshaler, _ProbeDef_OneofUnmarshaler, _ProbeDef_OneofSizer, []interface{}{
		(*ProbeDef_PingProbe)(nil),
		(*ProbeDef_HttpProbe)(nil),
		(*ProbeDef_DnsProbe)(nil),
		(*ProbeDef_ExternalProbe)(nil),
		(*ProbeDef_UdpProbe)(nil),
		(*ProbeDef_UdpListenerProbe)(nil),
		(*ProbeDef_UserDefinedProbe)(nil),
	}
}

func _ProbeDef_OneofMarshaler(msg proto1.Message, b *proto1.Buffer) error {
	m := msg.(*ProbeDef)
	// probe
	switch x := m.Probe.(type) {
	case *ProbeDef_PingProbe:
		b.EncodeVarint(20<<3 | proto1.WireBytes)
		if err := b.EncodeMessage(x.PingProbe); err != nil {
			return err
		}
	case *ProbeDef_HttpProbe:
		b.EncodeVarint(21<<3 | proto1.WireBytes)
		if err := b.EncodeMessage(x.HttpProbe); err != nil {
			return err
		}
	case *ProbeDef_DnsProbe:
		b.EncodeVarint(22<<3 | proto1.WireBytes)
		if err := b.EncodeMessage(x.DnsProbe); err != nil {
			return err
		}
	case *ProbeDef_ExternalProbe:
		b.EncodeVarint(23<<3 | proto1.WireBytes)
		if err := b.EncodeMessage(x.ExternalProbe); err != nil {
			return err
		}
	case *ProbeDef_UdpProbe:
		b.EncodeVarint(24<<3 | proto1.WireBytes)
		if err := b.EncodeMessage(x.UdpProbe); err != nil {
			return err
		}
	case *ProbeDef_UdpListenerProbe:
		b.EncodeVarint(25<<3 | proto1.WireBytes)
		if err := b.EncodeMessage(x.UdpListenerProbe); err != nil {
			return err
		}
	case *ProbeDef_UserDefinedProbe:
		b.EncodeVarint(99<<3 | proto1.WireBytes)
		b.EncodeStringBytes(x.UserDefinedProbe)
	case nil:
	default:
		return fmt.Errorf("ProbeDef.Probe has unexpected type %T", x)
	}
	return nil
}

func _ProbeDef_OneofUnmarshaler(msg proto1.Message, tag, wire int, b *proto1.Buffer) (bool, error) {
	m := msg.(*ProbeDef)
	switch tag {
	case 20: // probe.ping_probe
		if wire != proto1.WireBytes {
			return true, proto1.ErrInternalBadWireType
		}
		msg := new(cloudprober_probes_ping.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_PingProbe{msg}
		return true, err
	case 21: // probe.http_probe
		if wire != proto1.WireBytes {
			return true, proto1.ErrInternalBadWireType
		}
		msg := new(cloudprober_probes_http.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_HttpProbe{msg}
		return true, err
	case 22: // probe.dns_probe
		if wire != proto1.WireBytes {
			return true, proto1.ErrInternalBadWireType
		}
		msg := new(cloudprober_probes_dns.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_DnsProbe{msg}
		return true, err
	case 23: // probe.external_probe
		if wire != proto1.WireBytes {
			return true, proto1.ErrInternalBadWireType
		}
		msg := new(cloudprober_probes_external.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_ExternalProbe{msg}
		return true, err
	case 24: // probe.udp_probe
		if wire != proto1.WireBytes {
			return true, proto1.ErrInternalBadWireType
		}
		msg := new(cloudprober_probes_udp.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_UdpProbe{msg}
		return true, err
	case 25: // probe.udp_listener_probe
		if wire != proto1.WireBytes {
			return true, proto1.ErrInternalBadWireType
		}
		msg := new(cloudprober_probes_udplistener.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_UdpListenerProbe{msg}
		return true, err
	case 99: // probe.user_defined_probe
		if wire != proto1.WireBytes {
			return true, proto1.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Probe = &ProbeDef_UserDefinedProbe{x}
		return true, err
	default:
		return false, nil
	}
}

func _ProbeDef_OneofSizer(msg proto1.Message) (n int) {
	m := msg.(*ProbeDef)
	// probe
	switch x := m.Probe.(type) {
	case *ProbeDef_PingProbe:
		s := proto1.Size(x.PingProbe)
		n += proto1.SizeVarint(20<<3 | proto1.WireBytes)
		n += proto1.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_HttpProbe:
		s := proto1.Size(x.HttpProbe)
		n += proto1.SizeVarint(21<<3 | proto1.WireBytes)
		n += proto1.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_DnsProbe:
		s := proto1.Size(x.DnsProbe)
		n += proto1.SizeVarint(22<<3 | proto1.WireBytes)
		n += proto1.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_ExternalProbe:
		s := proto1.Size(x.ExternalProbe)
		n += proto1.SizeVarint(23<<3 | proto1.WireBytes)
		n += proto1.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_UdpProbe:
		s := proto1.Size(x.UdpProbe)
		n += proto1.SizeVarint(24<<3 | proto1.WireBytes)
		n += proto1.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_UdpListenerProbe:
		s := proto1.Size(x.UdpListenerProbe)
		n += proto1.SizeVarint(25<<3 | proto1.WireBytes)
		n += proto1.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_UserDefinedProbe:
		n += proto1.SizeVarint(99<<3 | proto1.WireBytes)
		n += proto1.SizeVarint(uint64(len(x.UserDefinedProbe)))
		n += len(x.UserDefinedProbe)
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

func init() {
	proto1.RegisterType((*ProbeDef)(nil), "cloudprober.probes.ProbeDef")
	proto1.RegisterEnum("cloudprober.probes.ProbeDef_Type", ProbeDef_Type_name, ProbeDef_Type_value)
}

func init() {
	proto1.RegisterFile("github.com/google/cloudprober/probes/proto/config.proto", fileDescriptor0)
}

var fileDescriptor0 = []byte{
	// 649 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x93, 0xd1, 0x6e, 0xd3, 0x30,
	0x14, 0x86, 0xd7, 0x2c, 0x5d, 0x53, 0xb7, 0x9b, 0x22, 0xb3, 0x41, 0xb6, 0x1b, 0xca, 0x24, 0xa0,
	0xe3, 0x22, 0x2d, 0x95, 0x26, 0xb4, 0x09, 0xa4, 0xc1, 0x12, 0x58, 0xa5, 0x92, 0x55, 0x69, 0x27,
	0xc1, 0x55, 0xd4, 0xc6, 0x6e, 0x67, 0xa9, 0x75, 0xa2, 0xd8, 0x46, 0xec, 0xae, 0x8f, 0xc7, 0x4b,
	0xf0, 0x2e, 0xc8, 0x8e, 0x83, 0x16, 0xd1, 0x4d, 0x15, 0x57, 0x4e, 0x8e, 0xff, 0xff, 0xf3, 0x39,
	0xf6, 0x39, 0xe0, 0xdd, 0x9c, 0xf0, 0x5b, 0x31, 0x75, 0xe3, 0x64, 0xd9, 0x99, 0x27, 0xc9, 0x7c,
	0x81, 0x3b, 0xf1, 0x22, 0x11, 0x28, 0xcd, 0x92, 0x29, 0xce, 0x3a, 0x6a, 0x61, 0x72, 0xe1, 0x49,
	0x27, 0x4e, 0xe8, 0x8c, 0xcc, 0x5d, 0xf5, 0x03, 0xe1, 0x3d, 0x99, 0x9b, 0xcb, 0x8e, 0xba, 0x8f,
	0xc3, 0x96, 0x98, 0x67, 0x24, 0x66, 0x1d, 0x44, 0x18, 0xcf, 0x29, 0x47, 0xef, 0x37, 0x3a, 0xfe,
	0x96, 0xf3, 0x74, 0x4d, 0x0e, 0x47, 0xa7, 0x1b, 0xb9, 0x11, 0x65, 0x65, 0xdb, 0xc5, 0x46, 0x36,
	0xfc, 0x93, 0xe3, 0x8c, 0x4e, 0x16, 0xeb, 0x0e, 0xde, 0x2c, 0xed, 0x94, 0xd0, 0xf9, 0xff, 0xa7,
	0x2d, 0x50, 0x5a, 0xb6, 0x7d, 0xd8, 0xd4, 0xb6, 0x20, 0x8c, 0x63, 0x8a, 0xb3, 0xb2, 0xfd, 0xec,
	0x71, 0x3b, 0x9f, 0x64, 0x73, 0xcc, 0x8b, 0xa7, 0xd6, 0x7f, 0xb9, 0xf5, 0xf8, 0x77, 0x0d, 0x58,
	0x43, 0x29, 0xf3, 0xf0, 0x0c, 0x42, 0x60, 0xd2, 0xc9, 0x12, 0x3b, 0x95, 0x96, 0xd1, 0xae, 0x87,
	0xea, 0x1b, 0x9e, 0x02, 0x93, 0xdf, 0xa5, 0xd8, 0x31, 0x5a, 0x46, 0x7b, 0xaf, 0xf7, 0xc2, 0xfd,
	0xb7, 0x37, 0xdc, 0xc2, 0xef, 0x8e, 0xef, 0x52, 0x1c, 0x2a, 0x39, 0x3c, 0x00, 0x3b, 0x99, 0xa0,
	0x51, 0x42, 0x9d, 0xed, 0x56, 0xa5, 0x5d, 0x0f, 0xab, 0x99, 0xa0, 0xd7, 0x14, 0x9e, 0x80, 0x5d,
	0x42, 0x39, 0xce, 0x7e, 0x4c, 0x16, 0xd1, 0x92, 0xe1, 0xd8, 0x31, 0x5b, 0x95, 0x76, 0xf5, 0xdc,
	0xec, 0x75, 0xbb, 0xdd, 0xb0, 0x59, 0x6c, 0x7d, 0x65, 0x38, 0x86, 0xaf, 0x41, 0x93, 0x93, 0x25,
	0x4e, 0x04, 0xcf, 0x95, 0xd5, 0x5c, 0xf9, 0x56, 0x2a, 0x1b, 0x7a, 0x47, 0x09, 0xcf, 0x40, 0x4d,
	0xd7, 0xe4, 0xec, 0xb4, 0x8c, 0x76, 0xa3, 0xf7, 0xbc, 0x94, 0x64, 0x51, 0xef, 0x38, 0x5f, 0x3d,
	0x3c, 0x0b, 0x0b, 0x3d, 0x1c, 0x80, 0xfd, 0xc5, 0x84, 0x63, 0x1a, 0xdf, 0x45, 0xb2, 0x73, 0x33,
	0x32, 0x15, 0x9c, 0x24, 0xd4, 0xa9, 0xb5, 0x2a, 0xed, 0x46, 0xef, 0xb0, 0xc4, 0xd1, 0x2d, 0xee,
	0x7a, 0x84, 0xf1, 0xf0, 0x89, 0xb6, 0x79, 0xf7, 0x5c, 0xf0, 0x25, 0x68, 0x16, 0x34, 0x41, 0x09,
	0x77, 0x2c, 0x59, 0xf9, 0xb9, 0x21, 0x58, 0xd8, 0xd0, 0xf1, 0x1b, 0x4a, 0x38, 0xbc, 0x04, 0x40,
	0xb6, 0x4f, 0xa4, 0xb8, 0xce, 0xbe, 0x3a, 0xea, 0x78, 0xdd, 0xbd, 0x4a, 0x55, 0x7e, 0xb9, 0x97,
	0x09, 0x9d, 0x5d, 0x6d, 0x85, 0x75, 0x19, 0x51, 0x01, 0x09, 0x91, 0xa3, 0xa3, 0x21, 0x07, 0x0f,
	0x43, 0xa4, 0xaa, 0x0c, 0x91, 0x91, 0x1c, 0x72, 0x01, 0xea, 0x88, 0x32, 0xcd, 0x78, 0xaa, 0x18,
	0x6b, 0x1f, 0x18, 0x51, 0x56, 0x42, 0x58, 0x88, 0xb2, 0x9c, 0x70, 0x0d, 0xf6, 0x8a, 0x61, 0xd2,
	0x98, 0x67, 0x0a, 0xf3, 0x6a, 0x1d, 0xa6, 0x50, 0x96, 0x58, 0xbb, 0x45, 0xf4, 0x6f, 0x4a, 0x02,
	0x15, 0x65, 0x39, 0x0f, 0xa7, 0x24, 0x50, 0xb9, 0x2a, 0x4b, 0x20, 0x5d, 0xd4, 0x77, 0x00, 0x25,
	0xa1, 0x98, 0x14, 0x8d, 0x3a, 0x54, 0xa8, 0x93, 0x07, 0x50, 0x85, 0xb8, 0x84, 0xb4, 0x05, 0x4a,
	0x07, 0x7a, 0x23, 0x47, 0xbb, 0x00, 0x0a, 0x86, 0xb3, 0x08, 0xe1, 0x19, 0xa1, 0x18, 0x69, 0x74,
	0x2c, 0x9f, 0x59, 0xe9, 0x19, 0xce, 0xbc, 0x7c, 0x4b, 0xe9, 0x8f, 0x97, 0xc0, 0x94, 0x23, 0x01,
	0x2d, 0x60, 0x0e, 0xfb, 0xc1, 0x17, 0x7b, 0x4b, 0x7e, 0x5d, 0x8d, 0xc7, 0x43, 0xbb, 0x02, 0x6b,
	0x60, 0xdb, 0x0b, 0x46, 0xb6, 0x01, 0x9b, 0xc0, 0xf2, 0xbf, 0x8d, 0xfd, 0x30, 0xf8, 0x38, 0xb0,
	0xb7, 0x65, 0xf8, 0xc6, 0x1b, 0xda, 0x26, 0xb4, 0x41, 0xf3, 0xc6, 0x1b, 0x46, 0x83, 0xfe, 0x68,
	0xec, 0x07, 0x7e, 0x68, 0x57, 0xe1, 0x2e, 0xa8, 0x4b, 0x61, 0x30, 0xea, 0x5f, 0x07, 0xf6, 0x54,
	0x09, 0x46, 0x7e, 0x18, 0x79, 0xfe, 0xe7, 0x7e, 0xe0, 0x7b, 0x76, 0xfc, 0xa6, 0x6e, 0xfd, 0xaa,
	0xd8, 0xab, 0xd5, 0x6a, 0x65, 0x7c, 0xaa, 0x81, 0xaa, 0x4a, 0xee, 0x4f, 0x00, 0x00, 0x00, 0xff,
	0xff, 0x3d, 0x2c, 0x91, 0xb4, 0x05, 0x06, 0x00, 0x00,
}
