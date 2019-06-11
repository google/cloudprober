// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/probes/proto/config.proto

package proto

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import proto2 "github.com/google/cloudprober/metrics/proto"
import proto6 "github.com/google/cloudprober/probes/dns/proto"
import proto7 "github.com/google/cloudprober/probes/external/proto"
import proto5 "github.com/google/cloudprober/probes/http/proto"
import proto4 "github.com/google/cloudprober/probes/ping/proto"
import proto8 "github.com/google/cloudprober/probes/udp/proto"
import proto9 "github.com/google/cloudprober/probes/udplistener/proto"
import proto1 "github.com/google/cloudprober/targets/proto"
import proto3 "github.com/google/cloudprober/validators/proto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

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
	return proto.EnumName(ProbeDef_Type_name, int32(x))
}
func (x *ProbeDef_Type) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(ProbeDef_Type_value, data, "ProbeDef_Type")
	if err != nil {
		return err
	}
	*x = ProbeDef_Type(value)
	return nil
}
func (ProbeDef_Type) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_config_be8fa0af4204c62d, []int{0, 0}
}

// IP version to use for networking probes. If specified, this is used at the
// time of resolving a target, picking the correct IP for the source IP if
// source_interface option is provided, and to craft the packet correctly
// for PING probes.
//
// If ip_version is not configured but source_ip is provided, we get
// ip_version from it. If both are  confgiured, an error is returned if there
// is a conflict between the two.
//
// If left unspecified and both addresses are available in resolve call or on
// source interface, IPv4 is preferred.
// Future work: provide an option to prefer IPv4 and IPv6 explicitly.
type ProbeDef_IPVersion int32

const (
	ProbeDef_IP_VERSION_UNSPECIFIED ProbeDef_IPVersion = 0
	ProbeDef_IPV4                   ProbeDef_IPVersion = 1
	ProbeDef_IPV6                   ProbeDef_IPVersion = 2
)

var ProbeDef_IPVersion_name = map[int32]string{
	0: "IP_VERSION_UNSPECIFIED",
	1: "IPV4",
	2: "IPV6",
}
var ProbeDef_IPVersion_value = map[string]int32{
	"IP_VERSION_UNSPECIFIED": 0,
	"IPV4":                   1,
	"IPV6":                   2,
}

func (x ProbeDef_IPVersion) Enum() *ProbeDef_IPVersion {
	p := new(ProbeDef_IPVersion)
	*p = x
	return p
}
func (x ProbeDef_IPVersion) String() string {
	return proto.EnumName(ProbeDef_IPVersion_name, int32(x))
}
func (x *ProbeDef_IPVersion) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(ProbeDef_IPVersion_value, data, "ProbeDef_IPVersion")
	if err != nil {
		return err
	}
	*x = ProbeDef_IPVersion(value)
	return nil
}
func (ProbeDef_IPVersion) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_config_be8fa0af4204c62d, []int{0, 1}
}

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
	Targets *proto1.TargetsDef `protobuf:"bytes,6,req,name=targets" json:"targets,omitempty"`
	// Latency distribution. If specified, latency is stored as a distribution.
	LatencyDistribution *proto2.Dist `protobuf:"bytes,7,opt,name=latency_distribution,json=latencyDistribution" json:"latency_distribution,omitempty"`
	// Latency unit. Any string that's parseable by time.ParseDuration.
	// Valid values: "ns", "us" (or "µs"), "ms", "s", "m", "h".
	LatencyUnit *string `protobuf:"bytes,8,opt,name=latency_unit,json=latencyUnit,def=us" json:"latency_unit,omitempty"`
	// Validators are in experimental phase right now and can change at any time.
	// NOTE: Only PING, HTTP and DNS probes support validators.
	Validator []*proto3.Validator `protobuf:"bytes,9,rep,name=validator" json:"validator,omitempty"`
	// Set the source IP to send packets from, either by providing an IP address
	// directly, or a network interface.
	//
	// Types that are valid to be assigned to SourceIpConfig:
	//	*ProbeDef_SourceIp
	//	*ProbeDef_SourceInterface
	SourceIpConfig isProbeDef_SourceIpConfig `protobuf_oneof:"source_ip_config"`
	IpVersion      *ProbeDef_IPVersion       `protobuf:"varint,12,opt,name=ip_version,json=ipVersion,enum=cloudprober.probes.ProbeDef_IPVersion" json:"ip_version,omitempty"`
	// How often to export stats. Probes usually run at a higher frequency (e.g.
	// every second); stats from individual probes are aggregated within
	// cloudprober until exported. In most cases, users don't need to change the
	// default.
	//
	// By default this field is set in the following way:
	// For all probes except UDP:
	//   stats_export_interval=max(interval, 10s)
	// For UDP:
	//   stats_export_interval=max(2*max(interval, timeout), 10s)
	StatsExportIntervalMsec *int32 `protobuf:"varint,13,opt,name=stats_export_interval_msec,json=statsExportIntervalMsec" json:"stats_export_interval_msec,omitempty"`
	// Types that are valid to be assigned to Probe:
	//	*ProbeDef_PingProbe
	//	*ProbeDef_HttpProbe
	//	*ProbeDef_DnsProbe
	//	*ProbeDef_ExternalProbe
	//	*ProbeDef_UdpProbe
	//	*ProbeDef_UdpListenerProbe
	//	*ProbeDef_UserDefinedProbe
	Probe                        isProbeDef_Probe `protobuf_oneof:"probe"`
	DebugOptions                 *DebugOptions    `protobuf:"bytes,100,opt,name=debug_options,json=debugOptions" json:"debug_options,omitempty"`
	XXX_NoUnkeyedLiteral         struct{}         `json:"-"`
	proto.XXX_InternalExtensions `json:"-"`
	XXX_unrecognized             []byte `json:"-"`
	XXX_sizecache                int32  `json:"-"`
}

func (m *ProbeDef) Reset()         { *m = ProbeDef{} }
func (m *ProbeDef) String() string { return proto.CompactTextString(m) }
func (*ProbeDef) ProtoMessage()    {}
func (*ProbeDef) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_be8fa0af4204c62d, []int{0}
}

var extRange_ProbeDef = []proto.ExtensionRange{
	{Start: 200, End: 536870911},
}

func (*ProbeDef) ExtensionRangeArray() []proto.ExtensionRange {
	return extRange_ProbeDef
}
func (m *ProbeDef) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProbeDef.Unmarshal(m, b)
}
func (m *ProbeDef) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProbeDef.Marshal(b, m, deterministic)
}
func (dst *ProbeDef) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProbeDef.Merge(dst, src)
}
func (m *ProbeDef) XXX_Size() int {
	return xxx_messageInfo_ProbeDef.Size(m)
}
func (m *ProbeDef) XXX_DiscardUnknown() {
	xxx_messageInfo_ProbeDef.DiscardUnknown(m)
}

var xxx_messageInfo_ProbeDef proto.InternalMessageInfo

const Default_ProbeDef_IntervalMsec int32 = 2000
const Default_ProbeDef_TimeoutMsec int32 = 1000
const Default_ProbeDef_LatencyUnit string = "us"

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

func (m *ProbeDef) GetTargets() *proto1.TargetsDef {
	if m != nil {
		return m.Targets
	}
	return nil
}

func (m *ProbeDef) GetLatencyDistribution() *proto2.Dist {
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

func (m *ProbeDef) GetValidator() []*proto3.Validator {
	if m != nil {
		return m.Validator
	}
	return nil
}

type isProbeDef_SourceIpConfig interface {
	isProbeDef_SourceIpConfig()
}

type ProbeDef_SourceIp struct {
	SourceIp string `protobuf:"bytes,10,opt,name=source_ip,json=sourceIp,oneof"`
}

type ProbeDef_SourceInterface struct {
	SourceInterface string `protobuf:"bytes,11,opt,name=source_interface,json=sourceInterface,oneof"`
}

func (*ProbeDef_SourceIp) isProbeDef_SourceIpConfig() {}

func (*ProbeDef_SourceInterface) isProbeDef_SourceIpConfig() {}

func (m *ProbeDef) GetSourceIpConfig() isProbeDef_SourceIpConfig {
	if m != nil {
		return m.SourceIpConfig
	}
	return nil
}

func (m *ProbeDef) GetSourceIp() string {
	if x, ok := m.GetSourceIpConfig().(*ProbeDef_SourceIp); ok {
		return x.SourceIp
	}
	return ""
}

func (m *ProbeDef) GetSourceInterface() string {
	if x, ok := m.GetSourceIpConfig().(*ProbeDef_SourceInterface); ok {
		return x.SourceInterface
	}
	return ""
}

func (m *ProbeDef) GetIpVersion() ProbeDef_IPVersion {
	if m != nil && m.IpVersion != nil {
		return *m.IpVersion
	}
	return ProbeDef_IP_VERSION_UNSPECIFIED
}

func (m *ProbeDef) GetStatsExportIntervalMsec() int32 {
	if m != nil && m.StatsExportIntervalMsec != nil {
		return *m.StatsExportIntervalMsec
	}
	return 0
}

type isProbeDef_Probe interface {
	isProbeDef_Probe()
}

type ProbeDef_PingProbe struct {
	PingProbe *proto4.ProbeConf `protobuf:"bytes,20,opt,name=ping_probe,json=pingProbe,oneof"`
}

type ProbeDef_HttpProbe struct {
	HttpProbe *proto5.ProbeConf `protobuf:"bytes,21,opt,name=http_probe,json=httpProbe,oneof"`
}

type ProbeDef_DnsProbe struct {
	DnsProbe *proto6.ProbeConf `protobuf:"bytes,22,opt,name=dns_probe,json=dnsProbe,oneof"`
}

type ProbeDef_ExternalProbe struct {
	ExternalProbe *proto7.ProbeConf `protobuf:"bytes,23,opt,name=external_probe,json=externalProbe,oneof"`
}

type ProbeDef_UdpProbe struct {
	UdpProbe *proto8.ProbeConf `protobuf:"bytes,24,opt,name=udp_probe,json=udpProbe,oneof"`
}

type ProbeDef_UdpListenerProbe struct {
	UdpListenerProbe *proto9.ProbeConf `protobuf:"bytes,25,opt,name=udp_listener_probe,json=udpListenerProbe,oneof"`
}

type ProbeDef_UserDefinedProbe struct {
	UserDefinedProbe string `protobuf:"bytes,99,opt,name=user_defined_probe,json=userDefinedProbe,oneof"`
}

func (*ProbeDef_PingProbe) isProbeDef_Probe() {}

func (*ProbeDef_HttpProbe) isProbeDef_Probe() {}

func (*ProbeDef_DnsProbe) isProbeDef_Probe() {}

func (*ProbeDef_ExternalProbe) isProbeDef_Probe() {}

func (*ProbeDef_UdpProbe) isProbeDef_Probe() {}

func (*ProbeDef_UdpListenerProbe) isProbeDef_Probe() {}

func (*ProbeDef_UserDefinedProbe) isProbeDef_Probe() {}

func (m *ProbeDef) GetProbe() isProbeDef_Probe {
	if m != nil {
		return m.Probe
	}
	return nil
}

func (m *ProbeDef) GetPingProbe() *proto4.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_PingProbe); ok {
		return x.PingProbe
	}
	return nil
}

func (m *ProbeDef) GetHttpProbe() *proto5.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_HttpProbe); ok {
		return x.HttpProbe
	}
	return nil
}

func (m *ProbeDef) GetDnsProbe() *proto6.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_DnsProbe); ok {
		return x.DnsProbe
	}
	return nil
}

func (m *ProbeDef) GetExternalProbe() *proto7.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_ExternalProbe); ok {
		return x.ExternalProbe
	}
	return nil
}

func (m *ProbeDef) GetUdpProbe() *proto8.ProbeConf {
	if x, ok := m.GetProbe().(*ProbeDef_UdpProbe); ok {
		return x.UdpProbe
	}
	return nil
}

func (m *ProbeDef) GetUdpListenerProbe() *proto9.ProbeConf {
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

func (m *ProbeDef) GetDebugOptions() *DebugOptions {
	if m != nil {
		return m.DebugOptions
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*ProbeDef) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _ProbeDef_OneofMarshaler, _ProbeDef_OneofUnmarshaler, _ProbeDef_OneofSizer, []interface{}{
		(*ProbeDef_SourceIp)(nil),
		(*ProbeDef_SourceInterface)(nil),
		(*ProbeDef_PingProbe)(nil),
		(*ProbeDef_HttpProbe)(nil),
		(*ProbeDef_DnsProbe)(nil),
		(*ProbeDef_ExternalProbe)(nil),
		(*ProbeDef_UdpProbe)(nil),
		(*ProbeDef_UdpListenerProbe)(nil),
		(*ProbeDef_UserDefinedProbe)(nil),
	}
}

func _ProbeDef_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*ProbeDef)
	// source_ip_config
	switch x := m.SourceIpConfig.(type) {
	case *ProbeDef_SourceIp:
		b.EncodeVarint(10<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.SourceIp)
	case *ProbeDef_SourceInterface:
		b.EncodeVarint(11<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.SourceInterface)
	case nil:
	default:
		return fmt.Errorf("ProbeDef.SourceIpConfig has unexpected type %T", x)
	}
	// probe
	switch x := m.Probe.(type) {
	case *ProbeDef_PingProbe:
		b.EncodeVarint(20<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.PingProbe); err != nil {
			return err
		}
	case *ProbeDef_HttpProbe:
		b.EncodeVarint(21<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.HttpProbe); err != nil {
			return err
		}
	case *ProbeDef_DnsProbe:
		b.EncodeVarint(22<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.DnsProbe); err != nil {
			return err
		}
	case *ProbeDef_ExternalProbe:
		b.EncodeVarint(23<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.ExternalProbe); err != nil {
			return err
		}
	case *ProbeDef_UdpProbe:
		b.EncodeVarint(24<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.UdpProbe); err != nil {
			return err
		}
	case *ProbeDef_UdpListenerProbe:
		b.EncodeVarint(25<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.UdpListenerProbe); err != nil {
			return err
		}
	case *ProbeDef_UserDefinedProbe:
		b.EncodeVarint(99<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.UserDefinedProbe)
	case nil:
	default:
		return fmt.Errorf("ProbeDef.Probe has unexpected type %T", x)
	}
	return nil
}

func _ProbeDef_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*ProbeDef)
	switch tag {
	case 10: // source_ip_config.source_ip
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.SourceIpConfig = &ProbeDef_SourceIp{x}
		return true, err
	case 11: // source_ip_config.source_interface
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.SourceIpConfig = &ProbeDef_SourceInterface{x}
		return true, err
	case 20: // probe.ping_probe
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(proto4.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_PingProbe{msg}
		return true, err
	case 21: // probe.http_probe
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(proto5.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_HttpProbe{msg}
		return true, err
	case 22: // probe.dns_probe
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(proto6.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_DnsProbe{msg}
		return true, err
	case 23: // probe.external_probe
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(proto7.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_ExternalProbe{msg}
		return true, err
	case 24: // probe.udp_probe
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(proto8.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_UdpProbe{msg}
		return true, err
	case 25: // probe.udp_listener_probe
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(proto9.ProbeConf)
		err := b.DecodeMessage(msg)
		m.Probe = &ProbeDef_UdpListenerProbe{msg}
		return true, err
	case 99: // probe.user_defined_probe
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Probe = &ProbeDef_UserDefinedProbe{x}
		return true, err
	default:
		return false, nil
	}
}

func _ProbeDef_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*ProbeDef)
	// source_ip_config
	switch x := m.SourceIpConfig.(type) {
	case *ProbeDef_SourceIp:
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(len(x.SourceIp)))
		n += len(x.SourceIp)
	case *ProbeDef_SourceInterface:
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(len(x.SourceInterface)))
		n += len(x.SourceInterface)
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	// probe
	switch x := m.Probe.(type) {
	case *ProbeDef_PingProbe:
		s := proto.Size(x.PingProbe)
		n += 2 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_HttpProbe:
		s := proto.Size(x.HttpProbe)
		n += 2 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_DnsProbe:
		s := proto.Size(x.DnsProbe)
		n += 2 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_ExternalProbe:
		s := proto.Size(x.ExternalProbe)
		n += 2 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_UdpProbe:
		s := proto.Size(x.UdpProbe)
		n += 2 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_UdpListenerProbe:
		s := proto.Size(x.UdpListenerProbe)
		n += 2 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *ProbeDef_UserDefinedProbe:
		n += 2 // tag and wire
		n += proto.SizeVarint(uint64(len(x.UserDefinedProbe)))
		n += len(x.UserDefinedProbe)
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type DebugOptions struct {
	// Whether to log metrics or not.
	LogMetrics           *bool    `protobuf:"varint,1,opt,name=log_metrics,json=logMetrics" json:"log_metrics,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DebugOptions) Reset()         { *m = DebugOptions{} }
func (m *DebugOptions) String() string { return proto.CompactTextString(m) }
func (*DebugOptions) ProtoMessage()    {}
func (*DebugOptions) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_be8fa0af4204c62d, []int{1}
}
func (m *DebugOptions) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DebugOptions.Unmarshal(m, b)
}
func (m *DebugOptions) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DebugOptions.Marshal(b, m, deterministic)
}
func (dst *DebugOptions) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DebugOptions.Merge(dst, src)
}
func (m *DebugOptions) XXX_Size() int {
	return xxx_messageInfo_DebugOptions.Size(m)
}
func (m *DebugOptions) XXX_DiscardUnknown() {
	xxx_messageInfo_DebugOptions.DiscardUnknown(m)
}

var xxx_messageInfo_DebugOptions proto.InternalMessageInfo

func (m *DebugOptions) GetLogMetrics() bool {
	if m != nil && m.LogMetrics != nil {
		return *m.LogMetrics
	}
	return false
}

func init() {
	proto.RegisterType((*ProbeDef)(nil), "cloudprober.probes.ProbeDef")
	proto.RegisterType((*DebugOptions)(nil), "cloudprober.probes.DebugOptions")
	proto.RegisterEnum("cloudprober.probes.ProbeDef_Type", ProbeDef_Type_name, ProbeDef_Type_value)
	proto.RegisterEnum("cloudprober.probes.ProbeDef_IPVersion", ProbeDef_IPVersion_name, ProbeDef_IPVersion_value)
}

func init() {
	proto.RegisterFile("github.com/google/cloudprober/probes/proto/config.proto", fileDescriptor_config_be8fa0af4204c62d)
}

var fileDescriptor_config_be8fa0af4204c62d = []byte{
	// 876 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x94, 0xdd, 0x6e, 0xdb, 0x36,
	0x14, 0xc7, 0x2b, 0xc7, 0x6e, 0xac, 0x63, 0x3b, 0x13, 0xb8, 0x7e, 0xa8, 0x06, 0x86, 0x6a, 0x06,
	0xb6, 0xb9, 0x1b, 0x60, 0x67, 0xc6, 0xd6, 0xa1, 0xcd, 0x80, 0x75, 0x8d, 0xd4, 0x45, 0x40, 0xea,
	0x18, 0xb4, 0x13, 0x6c, 0x57, 0x82, 0x2c, 0xd1, 0x2a, 0x01, 0x99, 0x12, 0x24, 0x2a, 0x68, 0xee,
	0x72, 0xb5, 0x67, 0xdb, 0x63, 0x0d, 0x14, 0xa9, 0xc4, 0xce, 0x94, 0xc0, 0xd8, 0x95, 0xa8, 0xc3,
	0xff, 0xf9, 0xf1, 0x7c, 0x90, 0x07, 0x7e, 0x89, 0x28, 0xff, 0x54, 0x2c, 0x47, 0x41, 0xb2, 0x1e,
	0x47, 0x49, 0x12, 0xc5, 0x64, 0x1c, 0xc4, 0x49, 0x11, 0xa6, 0x59, 0xb2, 0x24, 0xd9, 0xb8, 0xfc,
	0xe4, 0xe2, 0xc3, 0x93, 0x71, 0x90, 0xb0, 0x15, 0x8d, 0x46, 0xe5, 0x0f, 0x42, 0x1b, 0xb2, 0x91,
	0x94, 0xf5, 0x5f, 0x3f, 0x0c, 0x5b, 0x13, 0x9e, 0xd1, 0xa0, 0xa2, 0x85, 0x34, 0xe7, 0x92, 0xd5,
	0x3f, 0xda, 0x29, 0x88, 0x90, 0xd5, 0x05, 0xd2, 0x7f, 0xb7, 0x93, 0x33, 0xf9, 0xcc, 0x49, 0xc6,
	0xfc, 0xb8, 0x8e, 0xf0, 0xeb, 0x4e, 0x84, 0x4f, 0x9c, 0xa7, 0xff, 0xdf, 0x3b, 0xa5, 0x2c, 0xaa,
	0xf3, 0xde, 0x2d, 0xf5, 0x22, 0xac, 0x3d, 0xfa, 0x78, 0x57, 0xe7, 0x98, 0xe6, 0x9c, 0x30, 0x69,
	0xba, 0x0b, 0x79, 0xf3, 0x30, 0x84, 0xfb, 0x59, 0x44, 0x78, 0x55, 0x79, 0xf5, 0xb7, 0x5b, 0xf0,
	0x97, 0x7e, 0x4c, 0x43, 0x9f, 0x27, 0x59, 0x5d, 0xdf, 0x06, 0x7f, 0x77, 0xa0, 0x3d, 0x13, 0x42,
	0x9b, 0xac, 0x10, 0x82, 0x26, 0xf3, 0xd7, 0xc4, 0xd4, 0xac, 0xc6, 0x50, 0xc7, 0xe5, 0x1a, 0xfd,
	0x0c, 0x4d, 0x7e, 0x95, 0x12, 0xb3, 0x61, 0x35, 0x86, 0x07, 0x93, 0xaf, 0x47, 0xff, 0xbd, 0x70,
	0xa3, 0xca, 0x7f, 0xb4, 0xb8, 0x4a, 0x09, 0x2e, 0xe5, 0xe8, 0x29, 0x3c, 0xce, 0x0a, 0xe6, 0x25,
	0xcc, 0xdc, 0xb3, 0xb4, 0xa1, 0x8e, 0x5b, 0x59, 0xc1, 0xce, 0x18, 0x7a, 0x05, 0x3d, 0xca, 0x38,
	0xc9, 0x2e, 0xfd, 0xd8, 0x5b, 0xe7, 0x24, 0x30, 0x9b, 0x96, 0x36, 0x6c, 0xbd, 0x6d, 0x4e, 0x0e,
	0x0f, 0x0f, 0x71, 0xb7, 0xda, 0xfa, 0x98, 0x93, 0x00, 0x7d, 0x07, 0x5d, 0x4e, 0xd7, 0x24, 0x29,
	0xb8, 0x54, 0xb6, 0xa4, 0xf2, 0x47, 0xa1, 0xec, 0xa8, 0x9d, 0x52, 0xf8, 0x06, 0xf6, 0x55, 0x41,
	0xcc, 0xc7, 0x56, 0x63, 0xd8, 0x99, 0xbc, 0xdc, 0x0a, 0xb2, 0x2a, 0xd6, 0x42, 0x7e, 0x6d, 0xb2,
	0xc2, 0x95, 0x1e, 0x9d, 0xc2, 0x93, 0xd8, 0xe7, 0x84, 0x05, 0x57, 0x9e, 0x78, 0x08, 0x19, 0x5d,
	0x16, 0x9c, 0x26, 0xcc, 0xdc, 0xb7, 0xb4, 0x61, 0x67, 0xf2, 0x62, 0x8b, 0xa3, 0xde, 0xcd, 0xc8,
	0xa6, 0x39, 0xc7, 0x5f, 0x2a, 0x37, 0x7b, 0xc3, 0x0b, 0x7d, 0x03, 0xdd, 0x8a, 0x56, 0x30, 0xca,
	0xcd, 0xb6, 0xc8, 0xfc, 0x6d, 0xa3, 0xc8, 0x71, 0x47, 0xd9, 0xcf, 0x19, 0xe5, 0xe8, 0x37, 0xd0,
	0x6f, 0x7a, 0x62, 0xea, 0xd6, 0xde, 0xb0, 0x73, 0xa7, 0xac, 0xb7, 0x1d, 0x1b, 0x5d, 0x54, 0x4b,
	0x7c, 0xeb, 0x83, 0xbe, 0x02, 0x3d, 0x4f, 0x8a, 0x2c, 0x20, 0x1e, 0x4d, 0x4d, 0x10, 0x87, 0x9c,
	0x3c, 0xc2, 0x6d, 0x69, 0x72, 0x53, 0xf4, 0x03, 0x18, 0xd5, 0xb6, 0xa8, 0xe7, 0xca, 0x0f, 0x88,
	0xd9, 0x51, 0xaa, 0x2f, 0x94, 0xaa, 0xda, 0x40, 0x0e, 0x00, 0x4d, 0xbd, 0x4b, 0x92, 0xe5, 0x22,
	0xef, 0xae, 0xa5, 0x0d, 0x0f, 0x26, 0xdf, 0x3e, 0xd8, 0x64, 0x77, 0x76, 0x21, 0xd5, 0x58, 0xa7,
	0xa9, 0x5a, 0xa2, 0x23, 0xe8, 0xe7, 0xdc, 0xe7, 0xb9, 0x47, 0x3e, 0xa7, 0x49, 0xc6, 0xbd, 0xed,
	0x26, 0xf7, 0x44, 0xeb, 0xf0, 0xf3, 0x52, 0xe1, 0x94, 0x02, 0x77, 0xb3, 0xd3, 0xc7, 0x00, 0xe2,
	0x61, 0x7a, 0xe5, 0x49, 0xe6, 0x93, 0xb2, 0xf6, 0x83, 0xba, 0x18, 0x84, 0x4a, 0x06, 0x72, 0x9c,
	0xb0, 0xd5, 0x89, 0x86, 0x75, 0x61, 0x29, 0x0d, 0x02, 0x22, 0x66, 0x83, 0x82, 0x3c, 0xbd, 0x1f,
	0x22, 0x54, 0xdb, 0x10, 0x61, 0x91, 0x90, 0x77, 0xa0, 0x87, 0x2c, 0x57, 0x8c, 0x67, 0x25, 0xa3,
	0xf6, 0xc6, 0x87, 0x2c, 0xdf, 0x42, 0xb4, 0x43, 0x96, 0x4b, 0xc2, 0x19, 0x1c, 0x54, 0x43, 0x4e,
	0x61, 0x9e, 0x97, 0x98, 0xda, 0x9a, 0x56, 0xca, 0x2d, 0x56, 0xaf, 0xb2, 0xde, 0x84, 0x54, 0x84,
	0x55, 0x5a, 0xe6, 0xfd, 0x21, 0x15, 0xe1, 0x76, 0x56, 0xed, 0x22, 0x54, 0x49, 0xfd, 0x05, 0x48,
	0x10, 0xaa, 0xe9, 0xa3, 0x50, 0x2f, 0x4a, 0xd4, 0xab, 0x7b, 0x50, 0x95, 0x78, 0x0b, 0x69, 0x14,
	0x61, 0x7a, 0xaa, 0x36, 0x24, 0x7a, 0x04, 0xa8, 0xc8, 0x49, 0xe6, 0x85, 0x64, 0x45, 0x19, 0x09,
	0x15, 0x3a, 0x28, 0x2f, 0x9b, 0xd0, 0xe7, 0x24, 0xb3, 0xe5, 0x96, 0xd4, 0x3b, 0xd0, 0x0b, 0xc9,
	0xb2, 0x88, 0xbc, 0x24, 0x15, 0x2f, 0x26, 0x37, 0xc3, 0x32, 0x0a, 0xab, 0x2e, 0x0a, 0x5b, 0x08,
	0xcf, 0xa4, 0x0e, 0x77, 0xc3, 0x8d, 0xbf, 0xc1, 0x1a, 0x9a, 0x62, 0xd4, 0xa0, 0x36, 0x34, 0x67,
	0xee, 0xf4, 0x0f, 0xe3, 0x91, 0x58, 0x9d, 0x2c, 0x16, 0x33, 0x43, 0x43, 0xfb, 0xb0, 0x67, 0x4f,
	0xe7, 0x46, 0x03, 0x75, 0xa1, 0xed, 0xfc, 0xb9, 0x70, 0xf0, 0xf4, 0xf7, 0x53, 0x63, 0x4f, 0x98,
	0xcf, 0xed, 0x99, 0xd1, 0x44, 0x06, 0x74, 0xcf, 0xed, 0x99, 0x77, 0xea, 0xce, 0x17, 0xce, 0xd4,
	0xc1, 0x46, 0x0b, 0xf5, 0x40, 0x17, 0xc2, 0xe9, 0xdc, 0x3d, 0x9b, 0x1a, 0xcb, 0x52, 0x30, 0x77,
	0xb0, 0x67, 0x3b, 0x1f, 0xdc, 0xa9, 0x63, 0x1b, 0xc1, 0xe0, 0x08, 0xf4, 0x9b, 0x4b, 0x8f, 0xfa,
	0xf0, 0xcc, 0x9d, 0x79, 0x17, 0x0e, 0x16, 0x72, 0xef, 0x7c, 0x3a, 0x9f, 0x39, 0xc7, 0xee, 0x07,
	0xd7, 0xb1, 0x65, 0x14, 0xee, 0xec, 0xe2, 0x27, 0x43, 0x53, 0xab, 0xd7, 0x46, 0xe3, 0x7b, 0xbd,
	0xfd, 0x8f, 0x66, 0x5c, 0x5f, 0x5f, 0x5f, 0x37, 0xde, 0xa3, 0xdb, 0x87, 0x99, 0x7a, 0x72, 0x0a,
	0xbf, 0xdf, 0x87, 0x56, 0x99, 0xef, 0x60, 0x0c, 0xdd, 0xcd, 0x8c, 0xd1, 0x4b, 0xe8, 0xc4, 0x49,
	0xe4, 0xa9, 0xa9, 0x63, 0x6a, 0x96, 0x36, 0x6c, 0x63, 0x88, 0x93, 0xe8, 0xa3, 0xb4, 0xfc, 0x1b,
	0x00, 0x00, 0xff, 0xff, 0x87, 0xac, 0x70, 0xec, 0x34, 0x08, 0x00, 0x00,
}
