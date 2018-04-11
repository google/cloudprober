// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/probes/ping/proto/config.proto

/*
Package proto is a generated protocol buffer package.

It is generated from these files:
	github.com/google/cloudprober/probes/ping/proto/config.proto

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
	// Ping payload size in bytes. It cannot be smaller than 8, number of bytes
	// required for the nanoseconds timestamp.
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
	UseDatagramSocket *bool `protobuf:"varint,12,opt,name=use_datagram_socket,json=useDatagramSocket,def=1" json:"use_datagram_socket,omitempty"`
	// Disable integrity checks. To detect data courruption in the network, we
	// craft the outgoing ICMP packet payload in a certain format and verify that
	// the reply payload matches the same format.
	DisableIntegrityCheck *bool  `protobuf:"varint,13,opt,name=disable_integrity_check,json=disableIntegrityCheck,def=0" json:"disable_integrity_check,omitempty"`
	XXX_unrecognized      []byte `json:"-"`
}

func (m *ProbeConf) Reset()                    { *m = ProbeConf{} }
func (m *ProbeConf) String() string            { return proto1.CompactTextString(m) }
func (*ProbeConf) ProtoMessage()               {}
func (*ProbeConf) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

const Default_ProbeConf_PacketsPerProbe int32 = 2
const Default_ProbeConf_PacketsIntervalMsec int32 = 25
const Default_ProbeConf_StatsExportInterval int32 = 5
const Default_ProbeConf_ResolveTargetsInterval int32 = 5
const Default_ProbeConf_PayloadSize int32 = 56
const Default_ProbeConf_IpVersion int32 = 4
const Default_ProbeConf_UseDatagramSocket bool = true
const Default_ProbeConf_DisableIntegrityCheck bool = false

type isProbeConf_Source interface {
	isProbeConf_Source()
}

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

func (m *ProbeConf) GetDisableIntegrityCheck() bool {
	if m != nil && m.DisableIntegrityCheck != nil {
		return *m.DisableIntegrityCheck
	}
	return Default_ProbeConf_DisableIntegrityCheck
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*ProbeConf) XXX_OneofFuncs() (func(msg proto1.Message, b *proto1.Buffer) error, func(msg proto1.Message, tag, wire int, b *proto1.Buffer) (bool, error), func(msg proto1.Message) (n int), []interface{}) {
	return _ProbeConf_OneofMarshaler, _ProbeConf_OneofUnmarshaler, _ProbeConf_OneofSizer, []interface{}{
		(*ProbeConf_SourceIp)(nil),
		(*ProbeConf_SourceInterface)(nil),
	}
}

func _ProbeConf_OneofMarshaler(msg proto1.Message, b *proto1.Buffer) error {
	m := msg.(*ProbeConf)
	// source
	switch x := m.Source.(type) {
	case *ProbeConf_SourceIp:
		b.EncodeVarint(3<<3 | proto1.WireBytes)
		b.EncodeStringBytes(x.SourceIp)
	case *ProbeConf_SourceInterface:
		b.EncodeVarint(4<<3 | proto1.WireBytes)
		b.EncodeStringBytes(x.SourceInterface)
	case nil:
	default:
		return fmt.Errorf("ProbeConf.Source has unexpected type %T", x)
	}
	return nil
}

func _ProbeConf_OneofUnmarshaler(msg proto1.Message, tag, wire int, b *proto1.Buffer) (bool, error) {
	m := msg.(*ProbeConf)
	switch tag {
	case 3: // source.source_ip
		if wire != proto1.WireBytes {
			return true, proto1.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Source = &ProbeConf_SourceIp{x}
		return true, err
	case 4: // source.source_interface
		if wire != proto1.WireBytes {
			return true, proto1.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Source = &ProbeConf_SourceInterface{x}
		return true, err
	default:
		return false, nil
	}
}

func _ProbeConf_OneofSizer(msg proto1.Message) (n int) {
	m := msg.(*ProbeConf)
	// source
	switch x := m.Source.(type) {
	case *ProbeConf_SourceIp:
		n += proto1.SizeVarint(3<<3 | proto1.WireBytes)
		n += proto1.SizeVarint(uint64(len(x.SourceIp)))
		n += len(x.SourceIp)
	case *ProbeConf_SourceInterface:
		n += proto1.SizeVarint(4<<3 | proto1.WireBytes)
		n += proto1.SizeVarint(uint64(len(x.SourceInterface)))
		n += len(x.SourceInterface)
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

func init() {
	proto1.RegisterType((*ProbeConf)(nil), "cloudprober.probes.ping.ProbeConf")
}

func init() {
	proto1.RegisterFile("github.com/google/cloudprober/probes/ping/proto/config.proto", fileDescriptor0)
}

var fileDescriptor0 = []byte{
	// 389 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0x91, 0xdf, 0x6b, 0xd5, 0x30,
	0x18, 0x86, 0xad, 0xfb, 0x61, 0x9b, 0x4d, 0xe6, 0x72, 0x38, 0x2e, 0x37, 0x42, 0x11, 0x84, 0x03,
	0x62, 0x0b, 0x63, 0xdd, 0xc5, 0xd4, 0x1b, 0xa7, 0xe0, 0xb9, 0x10, 0x46, 0x26, 0xde, 0x86, 0x9c,
	0xf4, 0x6b, 0x16, 0xd6, 0xd3, 0x84, 0x24, 0x3d, 0xb8, 0xfd, 0xe9, 0x5e, 0x49, 0x93, 0x74, 0xec,
	0xaa, 0xcd, 0xf7, 0x3c, 0x6f, 0xde, 0x96, 0x0f, 0x7d, 0x91, 0xca, 0xdf, 0x8d, 0x9b, 0x4a, 0xe8,
	0x6d, 0x2d, 0xb5, 0x96, 0x3d, 0xd4, 0xa2, 0xd7, 0x63, 0x6b, 0xac, 0xde, 0x80, 0xad, 0xc3, 0xc3,
	0xd5, 0x46, 0x0d, 0x72, 0x7a, 0xf7, 0xba, 0x16, 0x7a, 0xe8, 0x94, 0xac, 0xc2, 0x01, 0x9f, 0x3d,
	0x73, 0xab, 0xe8, 0x56, 0x93, 0xfb, 0xfe, 0xdf, 0x1e, 0x2a, 0x6e, 0xa6, 0xf3, 0xb5, 0x1e, 0x3a,
	0xfc, 0x0e, 0x15, 0x4e, 0x8f, 0x56, 0x00, 0x53, 0x86, 0xec, 0x95, 0xd9, 0xaa, 0xf8, 0xf9, 0x82,
	0xe6, 0x71, 0xb4, 0x36, 0xf8, 0x23, 0x7a, 0x33, 0xe3, 0xc1, 0x83, 0xed, 0xb8, 0x00, 0xb2, 0x9f,
	0xac, 0x93, 0x64, 0xcd, 0x00, 0x7f, 0x42, 0xa7, 0x86, 0x8b, 0x7b, 0xf0, 0x8e, 0x19, 0xb0, 0x2c,
	0x94, 0x92, 0xc3, 0x32, 0x5b, 0x1d, 0x5c, 0x65, 0xe7, 0xf4, 0x24, 0xb1, 0x1b, 0xb0, 0xa1, 0x1e,
	0x5f, 0xa2, 0xe5, 0xac, 0x87, 0xcb, 0x77, 0xbc, 0x67, 0x5b, 0x07, 0x82, 0xbc, 0x0a, 0x91, 0x97,
	0xe7, 0x0d, 0x5d, 0x24, 0x61, 0x9d, 0xf8, 0x2f, 0x07, 0x02, 0x37, 0x68, 0xe9, 0x3c, 0xf7, 0x8e,
	0xc1, 0x5f, 0xa3, 0xad, 0x7f, 0x0a, 0x93, 0x3c, 0x56, 0x35, 0x74, 0x11, 0xf8, 0x8f, 0x80, 0xe7,
	0x28, 0xfe, 0x8c, 0x88, 0x05, 0xa7, 0xfb, 0x1d, 0x30, 0xcf, 0xad, 0x7c, 0x5e, 0x4b, 0x8a, 0x39,
	0xf9, 0x36, 0x29, 0xbf, 0xa3, 0xf1, 0x14, 0xfe, 0x80, 0x8e, 0x0d, 0x7f, 0xe8, 0x35, 0x6f, 0x99,
	0x53, 0x8f, 0x40, 0x50, 0xfc, 0xc4, 0xe6, 0x92, 0x1e, 0xa5, 0xf9, 0xad, 0x7a, 0x04, 0x5c, 0x22,
	0xa4, 0x0c, 0xdb, 0x81, 0x75, 0x4a, 0x0f, 0xe4, 0x28, 0xde, 0x7a, 0x41, 0x0b, 0x65, 0xfe, 0xc4,
	0x19, 0xbe, 0x40, 0x8b, 0xd1, 0x01, 0x6b, 0xb9, 0xe7, 0xd2, 0xf2, 0x2d, 0x73, 0x7a, 0xfa, 0x41,
	0x72, 0x5c, 0x66, 0xab, 0xfc, 0x6a, 0xdf, 0xdb, 0x11, 0xe8, 0xe9, 0xe8, 0xe0, 0x7b, 0xe2, 0xb7,
	0x01, 0xe3, 0xaf, 0xe8, 0xac, 0x55, 0x8e, 0x6f, 0xfa, 0xb8, 0x07, 0x69, 0x95, 0x7f, 0x60, 0xe2,
	0x0e, 0xc4, 0x3d, 0x79, 0x1d, 0x92, 0x07, 0x1d, 0xef, 0x1d, 0xd0, 0x65, 0xb2, 0xd6, 0xb3, 0x74,
	0x3d, 0x39, 0xdf, 0x72, 0x74, 0x18, 0x77, 0xf5, 0x3f, 0x00, 0x00, 0xff, 0xff, 0x88, 0xc8, 0x51,
	0xfe, 0x54, 0x02, 0x00, 0x00,
}
