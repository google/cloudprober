// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/config/proto/config.proto

package proto

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import proto1 "github.com/google/cloudprober/probes/proto"
import proto3 "github.com/google/cloudprober/servers/proto"
import proto2 "github.com/google/cloudprober/surfacers/proto"
import proto6 "github.com/google/cloudprober/targets/proto"
import proto4 "github.com/google/cloudprober/targets/rds/server/proto"
import proto5 "github.com/google/cloudprober/targets/rtc/rtcreporter/proto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type ProberConfig struct {
	// Probes to run.
	Probe []*proto1.ProbeDef `protobuf:"bytes,1,rep,name=probe" json:"probe,omitempty"`
	// Surfacers are used to export probe results for further processing.
	// If no surfacer is configured, a prometheus and a file surfacer are
	// initialized:
	//  - Prometheus makes probe results available at http://<host>:9313/metrics.
	//  - File surfacer writes results to stdout.
	//
	// You can disable default surfacers (in case you want no surfacer at all), by
	// adding the following to your config:
	//   surfacer {}
	Surfacer []*proto2.SurfacerDef `protobuf:"bytes,2,rep,name=surfacer" json:"surfacer,omitempty"`
	// Servers to run inside cloudprober. These servers can serve as targets for
	// other probes.
	Server []*proto3.ServerDef `protobuf:"bytes,3,rep,name=server" json:"server,omitempty"`
	// Resource discovery server
	RdsServer *proto4.ServerConf `protobuf:"bytes,95,opt,name=rds_server,json=rdsServer" json:"rds_server,omitempty"`
	// Port for the default HTTP server. This port is also used for prometheus
	// exporter (URL /metrics). Default port is 9313. If not specified in the
	// config, default port can be overridden by the environment variable
	// CLOUDPROBER_PORT.
	Port *int32 `protobuf:"varint,96,opt,name=port" json:"port,omitempty"`
	// How often to export system variables. To learn more about system variables:
	// http://godoc.org/github.com/google/cloudprober/sysvars.
	SysvarsIntervalMsec *int32 `protobuf:"varint,97,opt,name=sysvars_interval_msec,json=sysvarsIntervalMsec,def=10000" json:"sysvars_interval_msec,omitempty"`
	// Variables specified in this environment variable are exported as it is.
	// This is specifically useful to export information about system environment,
	// for example, docker image tag/digest-id, OS version etc. See
	// tools/cloudprober_startup.sh in the cloudprober directory for an example on
	// how to use these variables.
	SysvarsEnvVar *string `protobuf:"bytes,98,opt,name=sysvars_env_var,json=sysvarsEnvVar,def=SYSVARS" json:"sysvars_env_var,omitempty"`
	// Options for RTC reporter. RTC reporter reports information about the
	// current instance to a Runtime Config (RTC). This is useful if you want your
	// instance to be dynamically discoverable through RTC targets. This is
	// disabled by default.
	RtcReportOptions *proto5.RtcReportOptions `protobuf:"bytes,99,opt,name=rtc_report_options,json=rtcReportOptions" json:"rtc_report_options,omitempty"`
	// Global targets options. Per-probe options are specified within the probe
	// stanza.
	GlobalTargetsOptions *proto6.GlobalTargetsOptions `protobuf:"bytes,100,opt,name=global_targets_options,json=globalTargetsOptions" json:"global_targets_options,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                     `json:"-"`
	XXX_unrecognized     []byte                       `json:"-"`
	XXX_sizecache        int32                        `json:"-"`
}

func (m *ProberConfig) Reset()         { *m = ProberConfig{} }
func (m *ProberConfig) String() string { return proto.CompactTextString(m) }
func (*ProberConfig) ProtoMessage()    {}
func (*ProberConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_1514555b909b1ad9, []int{0}
}
func (m *ProberConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProberConfig.Unmarshal(m, b)
}
func (m *ProberConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProberConfig.Marshal(b, m, deterministic)
}
func (dst *ProberConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProberConfig.Merge(dst, src)
}
func (m *ProberConfig) XXX_Size() int {
	return xxx_messageInfo_ProberConfig.Size(m)
}
func (m *ProberConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_ProberConfig.DiscardUnknown(m)
}

var xxx_messageInfo_ProberConfig proto.InternalMessageInfo

const Default_ProberConfig_SysvarsIntervalMsec int32 = 10000
const Default_ProberConfig_SysvarsEnvVar string = "SYSVARS"

func (m *ProberConfig) GetProbe() []*proto1.ProbeDef {
	if m != nil {
		return m.Probe
	}
	return nil
}

func (m *ProberConfig) GetSurfacer() []*proto2.SurfacerDef {
	if m != nil {
		return m.Surfacer
	}
	return nil
}

func (m *ProberConfig) GetServer() []*proto3.ServerDef {
	if m != nil {
		return m.Server
	}
	return nil
}

func (m *ProberConfig) GetRdsServer() *proto4.ServerConf {
	if m != nil {
		return m.RdsServer
	}
	return nil
}

func (m *ProberConfig) GetPort() int32 {
	if m != nil && m.Port != nil {
		return *m.Port
	}
	return 0
}

func (m *ProberConfig) GetSysvarsIntervalMsec() int32 {
	if m != nil && m.SysvarsIntervalMsec != nil {
		return *m.SysvarsIntervalMsec
	}
	return Default_ProberConfig_SysvarsIntervalMsec
}

func (m *ProberConfig) GetSysvarsEnvVar() string {
	if m != nil && m.SysvarsEnvVar != nil {
		return *m.SysvarsEnvVar
	}
	return Default_ProberConfig_SysvarsEnvVar
}

func (m *ProberConfig) GetRtcReportOptions() *proto5.RtcReportOptions {
	if m != nil {
		return m.RtcReportOptions
	}
	return nil
}

func (m *ProberConfig) GetGlobalTargetsOptions() *proto6.GlobalTargetsOptions {
	if m != nil {
		return m.GlobalTargetsOptions
	}
	return nil
}

func init() {
	proto.RegisterType((*ProberConfig)(nil), "cloudprober.ProberConfig")
}

func init() {
	proto.RegisterFile("github.com/google/cloudprober/config/proto/config.proto", fileDescriptor_config_1514555b909b1ad9)
}

var fileDescriptor_config_1514555b909b1ad9 = []byte{
	// 424 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x91, 0x4f, 0x8f, 0xd3, 0x30,
	0x10, 0xc5, 0x15, 0x76, 0x0b, 0xac, 0x0b, 0x02, 0x99, 0x3f, 0x8a, 0x56, 0x08, 0x05, 0xb8, 0x84,
	0x4b, 0x92, 0xed, 0x01, 0xd8, 0x4a, 0x1c, 0xa0, 0x20, 0xc4, 0xa1, 0x02, 0x39, 0xa8, 0x12, 0x27,
	0xe3, 0x38, 0x6e, 0x88, 0x94, 0xc6, 0xd1, 0xd8, 0x8d, 0xc4, 0xa7, 0xe3, 0xab, 0xa1, 0xd8, 0x4e,
	0x95, 0xa2, 0x6c, 0xdb, 0x43, 0x65, 0xcf, 0xf8, 0xfd, 0xde, 0xeb, 0x64, 0xd0, 0xdb, 0xa2, 0xd4,
	0xbf, 0xb7, 0x59, 0xc4, 0xe5, 0x26, 0x2e, 0xa4, 0x2c, 0x2a, 0x11, 0xf3, 0x4a, 0x6e, 0xf3, 0x06,
	0x64, 0x26, 0x20, 0xe6, 0xb2, 0x5e, 0x97, 0x45, 0xdc, 0x80, 0xd4, 0xd2, 0x15, 0x91, 0x29, 0xf0,
	0x74, 0x20, 0xbb, 0x3c, 0xe2, 0x62, 0x0e, 0x35, 0xe2, 0x72, 0xf9, 0xee, 0x30, 0xa8, 0x04, 0xb4,
	0x02, 0x46, 0xc9, 0xeb, 0xc3, 0xa4, 0x66, 0x50, 0x08, 0xdd, 0x93, 0xae, 0x72, 0xe8, 0xfc, 0x48,
	0xe8, 0x16, 0xd6, 0x8c, 0xdf, 0x10, 0xbb, 0x38, 0x2d, 0x16, 0x72, 0xe5, 0xfe, 0xfc, 0x98, 0xc9,
	0xf2, 0x44, 0x13, 0xcd, 0xbb, 0x1f, 0x88, 0x46, 0x82, 0xde, 0x39, 0x0d, 0x3a, 0xd6, 0xee, 0xe5,
	0xdf, 0x73, 0x74, 0xef, 0xbb, 0x41, 0x17, 0x26, 0x05, 0xcf, 0xd0, 0xc4, 0x58, 0xf9, 0x5e, 0x70,
	0x16, 0x4e, 0x67, 0xcf, 0xa2, 0x81, 0x7b, 0x64, 0x97, 0x11, 0x19, 0xe0, 0x93, 0x58, 0x13, 0x2b,
	0xc5, 0xef, 0xd1, 0xdd, 0x7e, 0x70, 0xff, 0x96, 0xc1, 0x5e, 0xec, 0x61, 0xfd, 0x63, 0x94, 0xba,
	0x4b, 0xc7, 0xee, 0x10, 0xfc, 0x06, 0xdd, 0xb6, 0xf3, 0xfa, 0x67, 0x06, 0x7e, 0xbe, 0x0f, 0xdb,
	0x3d, 0x46, 0xa9, 0x39, 0x3b, 0xd2, 0xa9, 0xf1, 0x47, 0x84, 0x20, 0x57, 0xd4, 0xb1, 0x34, 0xf0,
	0xc2, 0xe9, 0xec, 0xd5, 0x1e, 0xdb, 0xef, 0x0e, 0xf2, 0x9e, 0xef, 0xa6, 0x24, 0x17, 0x90, 0x2b,
	0x5b, 0x62, 0x8c, 0xce, 0xbb, 0xef, 0xe1, 0xff, 0x0a, 0xbc, 0x70, 0x42, 0xcc, 0x1d, 0x5f, 0xa3,
	0x27, 0xea, 0x8f, 0x6a, 0x19, 0x28, 0x5a, 0xd6, 0x5a, 0x40, 0xcb, 0x2a, 0xba, 0x51, 0x82, 0xfb,
	0xac, 0x13, 0xcd, 0x27, 0x57, 0x49, 0x92, 0x24, 0xe4, 0x91, 0xd3, 0x7c, 0x75, 0x92, 0xa5, 0x12,
	0x1c, 0xc7, 0xe8, 0x41, 0x8f, 0x8a, 0xba, 0xa5, 0x2d, 0x03, 0x3f, 0x0b, 0xbc, 0xf0, 0x62, 0x7e,
	0x27, 0xfd, 0x99, 0xae, 0x3e, 0x90, 0x94, 0xdc, 0x77, 0xef, 0x9f, 0xeb, 0x76, 0xc5, 0x00, 0x53,
	0x84, 0x41, 0x73, 0x6a, 0xb7, 0x42, 0x65, 0xa3, 0x4b, 0x59, 0x2b, 0x9f, 0x9b, 0x59, 0xae, 0xc6,
	0x67, 0x19, 0xec, 0x90, 0x68, 0x4e, 0xcc, 0xfd, 0x9b, 0x05, 0xc9, 0x43, 0xf8, 0xaf, 0x83, 0x29,
	0x7a, 0x5a, 0x54, 0x32, 0x63, 0x15, 0x75, 0x06, 0xbb, 0x90, 0xdc, 0x84, 0xbc, 0x1e, 0x0d, 0xf9,
	0x62, 0x90, 0x1f, 0xb6, 0xea, 0xcd, 0x1f, 0x17, 0x23, 0xdd, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff,
	0xf4, 0xb6, 0x20, 0x10, 0x06, 0x04, 0x00, 0x00,
}
