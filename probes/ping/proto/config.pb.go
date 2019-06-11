// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/probes/ping/proto/config.proto

package proto

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

// Next tag: 1
type ProbeConf struct {
	// Packets per probe
	PacketsPerProbe *int32 `protobuf:"varint,6,opt,name=packets_per_probe,json=packetsPerProbe,def=2" json:"packets_per_probe,omitempty"`
	// How long to wait between two packets to the same target
	PacketsIntervalMsec *int32 `protobuf:"varint,7,opt,name=packets_interval_msec,json=packetsIntervalMsec,def=25" json:"packets_interval_msec,omitempty"`
	// Export stats after these many probes.
	// NOTE: Setting stats export interval using this field doesn't work anymore.
	// This field will be removed after the release v0.10.3. To set
	// stats_export_interval, please modify the outer probe configuration.
	// Example: probe {
	//   type: PING
	//   stats_export_interval_msec: 10000
	//   ping_probe {}
	// }
	StatsExportInterval *int32 `protobuf:"varint,8,opt,name=stats_export_interval,json=statsExportInterval,def=5" json:"stats_export_interval,omitempty"`
	// Resolve targets after these many probes
	ResolveTargetsInterval *int32 `protobuf:"varint,9,opt,name=resolve_targets_interval,json=resolveTargetsInterval,def=5" json:"resolve_targets_interval,omitempty"`
	// Ping payload size in bytes. It cannot be smaller than 8, number of bytes
	// required for the nanoseconds timestamp.
	PayloadSize *int32 `protobuf:"varint,10,opt,name=payload_size,json=payloadSize,def=56" json:"payload_size,omitempty"`
	// IP proto: 4|6.
	// This field is now deprecated and will be removed after the release v0.10.3.
	// ip_version can be configured in the outer layer of the config:
	// probe {
	//   type: PING
	//   ip_version: 6
	//   ping_probe {}
	// }
	IpVersion *int32 `protobuf:"varint,11,opt,name=ip_version,json=ipVersion" json:"ip_version,omitempty"`
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
	DisableIntegrityCheck *bool    `protobuf:"varint,13,opt,name=disable_integrity_check,json=disableIntegrityCheck,def=0" json:"disable_integrity_check,omitempty"`
	XXX_NoUnkeyedLiteral  struct{} `json:"-"`
	XXX_unrecognized      []byte   `json:"-"`
	XXX_sizecache         int32    `json:"-"`
}

func (m *ProbeConf) Reset()         { *m = ProbeConf{} }
func (m *ProbeConf) String() string { return proto.CompactTextString(m) }
func (*ProbeConf) ProtoMessage()    {}
func (*ProbeConf) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_4c6cf6fb9221606d, []int{0}
}
func (m *ProbeConf) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProbeConf.Unmarshal(m, b)
}
func (m *ProbeConf) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProbeConf.Marshal(b, m, deterministic)
}
func (dst *ProbeConf) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProbeConf.Merge(dst, src)
}
func (m *ProbeConf) XXX_Size() int {
	return xxx_messageInfo_ProbeConf.Size(m)
}
func (m *ProbeConf) XXX_DiscardUnknown() {
	xxx_messageInfo_ProbeConf.DiscardUnknown(m)
}

var xxx_messageInfo_ProbeConf proto.InternalMessageInfo

const Default_ProbeConf_PacketsPerProbe int32 = 2
const Default_ProbeConf_PacketsIntervalMsec int32 = 25
const Default_ProbeConf_StatsExportInterval int32 = 5
const Default_ProbeConf_ResolveTargetsInterval int32 = 5
const Default_ProbeConf_PayloadSize int32 = 56
const Default_ProbeConf_UseDatagramSocket bool = true
const Default_ProbeConf_DisableIntegrityCheck bool = false

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
	return 0
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

func init() {
	proto.RegisterType((*ProbeConf)(nil), "cloudprober.probes.ping.ProbeConf")
}

func init() {
	proto.RegisterFile("github.com/google/cloudprober/probes/ping/proto/config.proto", fileDescriptor_config_4c6cf6fb9221606d)
}

var fileDescriptor_config_4c6cf6fb9221606d = []byte{
	// 342 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0x8f, 0x41, 0x6b, 0x2a, 0x31,
	0x10, 0xc7, 0xf1, 0xbd, 0xe7, 0xab, 0x46, 0x4b, 0x31, 0x62, 0xcd, 0xa5, 0x20, 0x85, 0x82, 0x97,
	0xee, 0x82, 0x74, 0x3d, 0xd8, 0xf6, 0x64, 0x7b, 0xf0, 0x50, 0x90, 0xb5, 0xf4, 0x1a, 0x62, 0x76,
	0x8c, 0xc1, 0x75, 0x13, 0x92, 0xac, 0x54, 0xbf, 0x4d, 0xbf, 0x69, 0xd9, 0xec, 0xae, 0x78, 0x4a,
	0x66, 0xfe, 0xbf, 0x1f, 0x33, 0x83, 0x5e, 0x84, 0x74, 0xdb, 0x7c, 0x1d, 0x70, 0xb5, 0x0f, 0x85,
	0x52, 0x22, 0x85, 0x90, 0xa7, 0x2a, 0x4f, 0xb4, 0x51, 0x6b, 0x30, 0xa1, 0x7f, 0x6c, 0xa8, 0x65,
	0x26, 0x8a, 0xbf, 0x53, 0x21, 0x57, 0xd9, 0x46, 0x8a, 0xc0, 0x17, 0x78, 0x78, 0xc1, 0x06, 0x25,
	0x1b, 0x14, 0xec, 0xfd, 0xcf, 0x5f, 0xd4, 0x5e, 0x16, 0xf5, 0x5c, 0x65, 0x1b, 0xfc, 0x88, 0x7a,
	0x9a, 0xf1, 0x1d, 0x38, 0x4b, 0x35, 0x18, 0xea, 0x41, 0xf2, 0x7f, 0xd4, 0x18, 0x37, 0x67, 0x8d,
	0x49, 0x7c, 0x53, 0x65, 0x4b, 0x30, 0x5e, 0xc1, 0x53, 0x34, 0xa8, 0x71, 0x99, 0x39, 0x30, 0x07,
	0x96, 0xd2, 0xbd, 0x05, 0x4e, 0xae, 0xbc, 0xf2, 0x67, 0x12, 0xc5, 0xfd, 0x0a, 0x58, 0x54, 0xf9,
	0x87, 0x05, 0x8e, 0x23, 0x34, 0xb0, 0x8e, 0x39, 0x4b, 0xe1, 0x5b, 0x2b, 0xe3, 0xce, 0x32, 0x69,
	0x95, 0xa3, 0xa2, 0xb8, 0xef, 0xf3, 0x77, 0x1f, 0xd7, 0x2a, 0x7e, 0x46, 0xc4, 0x80, 0x55, 0xe9,
	0x01, 0xa8, 0x63, 0x46, 0x5c, 0x8e, 0x25, 0xed, 0xda, 0xbc, 0xad, 0x90, 0xcf, 0x92, 0x38, 0xcb,
	0x0f, 0xa8, 0xab, 0xd9, 0x31, 0x55, 0x2c, 0xa1, 0x56, 0x9e, 0x80, 0xa0, 0x72, 0xc5, 0x68, 0x1a,
	0x77, 0xaa, 0xfe, 0x4a, 0x9e, 0x00, 0xdf, 0x21, 0x24, 0x35, 0x3d, 0x80, 0xb1, 0x52, 0x65, 0xa4,
	0x53, 0x40, 0x71, 0x5b, 0xea, 0xaf, 0xb2, 0x81, 0x9f, 0x50, 0x3f, 0xb7, 0x40, 0x13, 0xe6, 0x98,
	0x30, 0x6c, 0x4f, 0xad, 0x2a, 0xae, 0x23, 0xdd, 0x51, 0x63, 0xdc, 0x9a, 0xfd, 0x73, 0x26, 0x87,
	0xb8, 0x97, 0x5b, 0x78, 0xab, 0xf2, 0x95, 0x8f, 0xf1, 0x2b, 0x1a, 0x26, 0xd2, 0xb2, 0x75, 0x0a,
	0x7e, 0x61, 0x61, 0xa4, 0x3b, 0x52, 0xbe, 0x05, 0xbe, 0x23, 0xd7, 0xde, 0x6c, 0x6e, 0x58, 0x6a,
	0x21, 0x1e, 0x54, 0xd4, 0xa2, 0x86, 0xe6, 0x05, 0xf3, 0x1b, 0x00, 0x00, 0xff, 0xff, 0x18, 0xc6,
	0xc4, 0x64, 0xfb, 0x01, 0x00, 0x00,
}
