// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/probes/ping/config.proto

/*
Package ping is a generated protocol buffer package.

It is generated from these files:
	github.com/google/cloudprober/probes/ping/config.proto

It has these top-level messages:
	ProbeConf
*/
package ping

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

type ProbeConf struct {
	// Set the source address to send packets from, either by providing an address
	// or a network interface.
	//
	// Types that are valid to be assigned to Source:
	//	*ProbeConf_SourceIp
	//	*ProbeConf_SourceInterface
	Source isProbeConf_Source `protobuf_oneof:"source"`
	// Packets per probe
	PacketsPerProbe *int32 `protobuf:"varint,6,opt,name=packets_per_probe,json=packetsPerProbe,def=2" json:"packets_per_probe,omitempty"`
	// How long to wait between two packets to the same target
	PacketsIntervalMsec *int32 `protobuf:"varint,7,opt,name=packets_interval_msec,json=packetsIntervalMsec,def=25" json:"packets_interval_msec,omitempty"`
	// Export stats after these many probes
	StatsExportInterval *int32 `protobuf:"varint,8,opt,name=stats_export_interval,json=statsExportInterval,def=5" json:"stats_export_interval,omitempty"`
	// Resolve targets after these many probes
	ResolveTargetsInterval *int32 `protobuf:"varint,9,opt,name=resolve_targets_interval,json=resolveTargetsInterval,def=5" json:"resolve_targets_interval,omitempty"`
	// Ping payload size
	PayloadSize *int32 `protobuf:"varint,10,opt,name=payload_size,json=payloadSize,def=56" json:"payload_size,omitempty"`
	// IP protocol version
	IpVersion *int32 `protobuf:"varint,11,opt,name=ip_version,json=ipVersion,def=4" json:"ip_version,omitempty"`
	// Use datagram socket for ICMP.
	// This option enables unprivileged pings (that is, you don't require root
	// privilege to send ICMP packets). Note that most of the Linux distributions
	// don't allow unprivileged pings by default. To enable unprivileged pings on
	// some Linux distributions, you may need to run the following command:
	//     sudo sysctl -w net.ipv4.ping_group_range="0 <large valid group id>"
	// net.ipv4.ping_group_range system setting takes two integers that specify
	// the group id range that is allowed to execute the unprivileged pings. Note
	// that the same setting (with ipv4 in the path) applies to IPv6 as well.
	UseDatagramSocket *bool  `protobuf:"varint,12,opt,name=use_datagram_socket,json=useDatagramSocket,def=1" json:"use_datagram_socket,omitempty"`
	XXX_unrecognized  []byte `json:"-"`
}

func (m *ProbeConf) Reset()                    { *m = ProbeConf{} }
func (m *ProbeConf) String() string            { return proto.CompactTextString(m) }
func (*ProbeConf) ProtoMessage()               {}
func (*ProbeConf) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

const Default_ProbeConf_PacketsPerProbe int32 = 2
const Default_ProbeConf_PacketsIntervalMsec int32 = 25
const Default_ProbeConf_StatsExportInterval int32 = 5
const Default_ProbeConf_ResolveTargetsInterval int32 = 5
const Default_ProbeConf_PayloadSize int32 = 56
const Default_ProbeConf_IpVersion int32 = 4
const Default_ProbeConf_UseDatagramSocket bool = true

type isProbeConf_Source interface{ isProbeConf_Source() }

type ProbeConf_SourceIp struct {
	SourceIp string `protobuf:"bytes,3,opt,name=source_ip,json=sourceIp,oneof"`
}
type ProbeConf_SourceInterface struct {
	SourceInterface string `protobuf:"bytes,4,opt,name=source_interface,json=sourceInterface,oneof"`
}

func (*ProbeConf_SourceIp) isProbeConf_Source()        {}
func (*ProbeConf_SourceInterface) isProbeConf_Source() {}

func (m *ProbeConf) GetSource() isProbeConf_Source {
	if m != nil {
		return m.Source
	}
	return nil
}

func (m *ProbeConf) GetSourceIp() string {
	if x, ok := m.GetSource().(*ProbeConf_SourceIp); ok {
		return x.SourceIp
	}
	return ""
}

func (m *ProbeConf) GetSourceInterface() string {
	if x, ok := m.GetSource().(*ProbeConf_SourceInterface); ok {
		return x.SourceInterface
	}
	return ""
}

func (m *ProbeConf) GetPacketsPerProbe() int32 {
	if m != nil && m.PacketsPerProbe != nil {
		return *m.PacketsPerProbe
	}
	return Default_ProbeConf_PacketsPerProbe
}

func (m *ProbeConf) GetPacketsIntervalMsec() int32 {
	if m != nil && m.PacketsIntervalMsec != nil {
		return *m.PacketsIntervalMsec
	}
	return Default_ProbeConf_PacketsIntervalMsec
}

func (m *ProbeConf) GetStatsExportInterval() int32 {
	if m != nil && m.StatsExportInterval != nil {
		return *m.StatsExportInterval
	}
	return Default_ProbeConf_StatsExportInterval
}

func (m *ProbeConf) GetResolveTargetsInterval() int32 {
	if m != nil && m.ResolveTargetsInterval != nil {
		return *m.ResolveTargetsInterval
	}
	return Default_ProbeConf_ResolveTargetsInterval
}

func (m *ProbeConf) GetPayloadSize() int32 {
	if m != nil && m.PayloadSize != nil {
		return *m.PayloadSize
	}
	return Default_ProbeConf_PayloadSize
}

func (m *ProbeConf) GetIpVersion() int32 {
	if m != nil && m.IpVersion != nil {
		return *m.IpVersion
	}
	return Default_ProbeConf_IpVersion
}

func (m *ProbeConf) GetUseDatagramSocket() bool {
	if m != nil && m.UseDatagramSocket != nil {
		return *m.UseDatagramSocket
	}
	return Default_ProbeConf_UseDatagramSocket
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*ProbeConf) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _ProbeConf_OneofMarshaler, _ProbeConf_OneofUnmarshaler, _ProbeConf_OneofSizer, []interface{}{
		(*ProbeConf_SourceIp)(nil),
		(*ProbeConf_SourceInterface)(nil),
	}
}

func _ProbeConf_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*ProbeConf)
	// source
	switch x := m.Source.(type) {
	case *ProbeConf_SourceIp:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.SourceIp)
	case *ProbeConf_SourceInterface:
		b.EncodeVarint(4<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.SourceInterface)
	case nil:
	default:
		return fmt.Errorf("ProbeConf.Source has unexpected type %T", x)
	}
	return nil
}

func _ProbeConf_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*ProbeConf)
	switch tag {
	case 3: // source.source_ip
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Source = &ProbeConf_SourceIp{x}
		return true, err
	case 4: // source.source_interface
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Source = &ProbeConf_SourceInterface{x}
		return true, err
	default:
		return false, nil
	}
}

func _ProbeConf_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*ProbeConf)
	// source
	switch x := m.Source.(type) {
	case *ProbeConf_SourceIp:
		n += proto.SizeVarint(3<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(len(x.SourceIp)))
		n += len(x.SourceIp)
	case *ProbeConf_SourceInterface:
		n += proto.SizeVarint(4<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(len(x.SourceInterface)))
		n += len(x.SourceInterface)
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

func init() {
	proto.RegisterType((*ProbeConf)(nil), "cloudprober.probes.ping.ProbeConf")
}

func init() {
	proto.RegisterFile("github.com/google/cloudprober/probes/ping/config.proto", fileDescriptor0)
}

var fileDescriptor0 = []byte{
	// 353 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0x90, 0x5f, 0xab, 0xda, 0x30,
	0x18, 0xc6, 0xd7, 0xe9, 0x5c, 0x1b, 0x05, 0x67, 0xc4, 0x2d, 0x37, 0x83, 0x32, 0x18, 0x08, 0x63,
	0x2d, 0x88, 0xf5, 0xc2, 0xdd, 0xed, 0x0f, 0xcc, 0x8b, 0x81, 0xd4, 0xb1, 0xdb, 0x10, 0xe3, 0x6b,
	0x17, 0x4e, 0x6d, 0x42, 0x92, 0xca, 0x39, 0x7e, 0xb5, 0xf3, 0xe5, 0x0e, 0x26, 0xa9, 0x78, 0x55,
	0xfa, 0xfe, 0x7e, 0xcf, 0xfb, 0x24, 0x41, 0xab, 0x4a, 0xd8, 0xff, 0xed, 0x3e, 0xe3, 0xf2, 0x94,
	0x57, 0x52, 0x56, 0x35, 0xe4, 0xbc, 0x96, 0xed, 0x41, 0x69, 0xb9, 0x07, 0x9d, 0xbb, 0x8f, 0xc9,
	0x95, 0x68, 0xaa, 0x9c, 0xcb, 0xe6, 0x28, 0xaa, 0x4c, 0x69, 0x69, 0x25, 0xfe, 0x70, 0x67, 0x65,
	0xde, 0xca, 0xae, 0xd6, 0xa7, 0xe7, 0x1e, 0x4a, 0xb6, 0xd7, 0xff, 0x1f, 0xb2, 0x39, 0xe2, 0x8f,
	0x28, 0x31, 0xb2, 0xd5, 0x1c, 0xa8, 0x50, 0xa4, 0x97, 0x46, 0xf3, 0xe4, 0xf7, 0xab, 0x32, 0xf6,
	0xa3, 0x8d, 0xc2, 0x5f, 0xd0, 0xbb, 0x0e, 0x37, 0x16, 0xf4, 0x91, 0x71, 0x20, 0xfd, 0x60, 0x8d,
	0x83, 0xd5, 0x01, 0xfc, 0x15, 0x4d, 0x14, 0xe3, 0x0f, 0x60, 0x0d, 0x55, 0xa0, 0xa9, 0x2b, 0x25,
	0x83, 0x34, 0x9a, 0xbf, 0x59, 0x47, 0x8b, 0x72, 0x1c, 0xd8, 0x16, 0xb4, 0xab, 0xc7, 0x2b, 0x34,
	0xeb, 0x74, 0xb7, 0xfc, 0xcc, 0x6a, 0x7a, 0x32, 0xc0, 0xc9, 0x5b, 0x17, 0x79, 0xbd, 0x28, 0xca,
	0x69, 0x10, 0x36, 0x81, 0xff, 0x31, 0xc0, 0x71, 0x81, 0x66, 0xc6, 0x32, 0x6b, 0x28, 0x3c, 0x2a,
	0xa9, 0xed, 0x2d, 0x4c, 0x62, 0x5f, 0x55, 0x94, 0x53, 0xc7, 0x7f, 0x39, 0xdc, 0x45, 0xf1, 0x37,
	0x44, 0x34, 0x18, 0x59, 0x9f, 0x81, 0x5a, 0xa6, 0xab, 0xfb, 0x5a, 0x92, 0x74, 0xc9, 0xf7, 0x41,
	0xf9, 0xeb, 0x8d, 0x5b, 0xf8, 0x33, 0x1a, 0x29, 0xf6, 0x54, 0x4b, 0x76, 0xa0, 0x46, 0x5c, 0x80,
	0x20, 0x7f, 0xc4, 0x62, 0x55, 0x0e, 0xc3, 0x7c, 0x27, 0x2e, 0x80, 0x53, 0x84, 0x84, 0xa2, 0x67,
	0xd0, 0x46, 0xc8, 0x86, 0x0c, 0xfd, 0xd6, 0x65, 0x99, 0x08, 0xf5, 0xcf, 0xcf, 0xf0, 0x12, 0x4d,
	0x5b, 0x03, 0xf4, 0xc0, 0x2c, 0xab, 0x34, 0x3b, 0x51, 0x23, 0xaf, 0x17, 0x24, 0xa3, 0x34, 0x9a,
	0xc7, 0xeb, 0xbe, 0xd5, 0x2d, 0x94, 0x93, 0xd6, 0xc0, 0xcf, 0xc0, 0x77, 0x0e, 0x7f, 0x8f, 0xd1,
	0xc0, 0x3f, 0xf6, 0x4b, 0x00, 0x00, 0x00, 0xff, 0xff, 0xd8, 0x82, 0x7c, 0xa8, 0x0f, 0x02, 0x00,
	0x00,
}
