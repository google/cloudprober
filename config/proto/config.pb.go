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
	// Host for the default HTTP server. Default listens on all addresses. If not
	// specified in the config, default port can be overridden by the environment
	// variable CLOUDPROBER_HOST.
	Host *string `protobuf:"bytes,101,opt,name=host" json:"host,omitempty"`
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
	return fileDescriptor_config_3262c8776d939396, []int{0}
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

func (m *ProberConfig) GetHost() string {
	if m != nil && m.Host != nil {
		return *m.Host
	}
	return ""
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
	proto.RegisterFile("github.com/google/cloudprober/config/proto/config.proto", fileDescriptor_config_3262c8776d939396)
}

var fileDescriptor_config_3262c8776d939396 = []byte{
	// 433 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x91, 0x4d, 0x6f, 0xd4, 0x30,
	0x10, 0x86, 0xb5, 0xb4, 0x0b, 0xd4, 0x0b, 0x02, 0x99, 0x0f, 0x59, 0x15, 0x42, 0x01, 0x2e, 0xe1,
	0x92, 0xa4, 0x7b, 0x00, 0xba, 0x12, 0x07, 0x28, 0x08, 0x71, 0xa8, 0x40, 0x0e, 0xaa, 0xc4, 0xc9,
	0x38, 0x8e, 0x37, 0x8d, 0x94, 0xc6, 0xd1, 0xd8, 0x1b, 0x89, 0xdf, 0xc9, 0x1f, 0x42, 0xfe, 0xc8,
	0x2a, 0x8b, 0x42, 0xbb, 0x87, 0xc8, 0x33, 0xe3, 0x79, 0xde, 0x37, 0xe3, 0x41, 0x6f, 0xab, 0xda,
	0x5c, 0x6e, 0x8a, 0x44, 0xa8, 0xab, 0xb4, 0x52, 0xaa, 0x6a, 0x64, 0x2a, 0x1a, 0xb5, 0x29, 0x3b,
	0x50, 0x85, 0x84, 0x54, 0xa8, 0x76, 0x5d, 0x57, 0x69, 0x07, 0xca, 0xa8, 0x90, 0x24, 0x2e, 0xc1,
	0x8b, 0x51, 0xdb, 0xf1, 0x0d, 0x2a, 0xee, 0xd0, 0x13, 0x2a, 0xc7, 0xef, 0xae, 0x07, 0xb5, 0x84,
	0x5e, 0xc2, 0x24, 0xb9, 0xba, 0x81, 0xdc, 0xc0, 0x9a, 0x8b, 0xff, 0xb0, 0xa7, 0xd7, 0xb3, 0x86,
	0x43, 0x25, 0xcd, 0x40, 0x86, 0x2c, 0xa0, 0x67, 0xfb, 0xa1, 0x50, 0xea, 0xf0, 0xf3, 0x53, 0xfe,
	0xe7, 0x7b, 0x8a, 0x18, 0x61, 0x3f, 0x90, 0x9d, 0x02, 0xb3, 0x55, 0x1a, 0x55, 0xbc, 0xdc, 0xcb,
	0x3f, 0x87, 0xe8, 0xde, 0x77, 0x87, 0x9e, 0x39, 0x17, 0xbc, 0x44, 0x73, 0x27, 0x45, 0x66, 0xd1,
	0x41, 0xbc, 0x58, 0x3e, 0x4b, 0x46, 0xea, 0x89, 0x5f, 0x46, 0xe2, 0x80, 0x4f, 0x72, 0x4d, 0x7d,
	0x2b, 0x7e, 0x8f, 0xee, 0x0e, 0x6f, 0x46, 0x6e, 0x39, 0xec, 0xc5, 0x0e, 0x36, 0x5c, 0x26, 0x79,
	0x08, 0x2c, 0xbb, 0x45, 0xf0, 0x1b, 0x74, 0xdb, 0xcf, 0x4b, 0x0e, 0x1c, 0xfc, 0x7c, 0x17, 0xf6,
	0x7b, 0x4c, 0x72, 0x77, 0x5a, 0x32, 0x74, 0xe3, 0x8f, 0x08, 0x41, 0xa9, 0x59, 0x60, 0x59, 0x34,
	0x8b, 0x17, 0xcb, 0x57, 0x3b, 0xec, 0xf0, 0xfe, 0x50, 0x0e, 0xbc, 0x9d, 0x92, 0x1e, 0x41, 0xa9,
	0x7d, 0x8a, 0x31, 0x3a, 0xb4, 0xef, 0x41, 0x7e, 0x45, 0xb3, 0x78, 0x4e, 0x5d, 0x6c, 0x6b, 0x97,
	0x4a, 0x1b, 0x22, 0xa3, 0x59, 0x7c, 0x44, 0x5d, 0x8c, 0x4f, 0xd1, 0x13, 0xfd, 0x5b, 0xf7, 0x1c,
	0x34, 0xab, 0x5b, 0x23, 0xa1, 0xe7, 0x0d, 0xbb, 0xd2, 0x52, 0x10, 0x6e, 0xc1, 0xd5, 0xfc, 0x24,
	0xcb, 0xb2, 0x8c, 0x3e, 0x0a, 0x3d, 0x5f, 0x43, 0xcb, 0xb9, 0x96, 0x02, 0xa7, 0xe8, 0xc1, 0x80,
	0xca, 0xb6, 0x67, 0x3d, 0x07, 0x52, 0x58, 0xe5, 0xd5, 0x9d, 0xfc, 0x67, 0x7e, 0xf1, 0x81, 0xe6,
	0xf4, 0x7e, 0xb8, 0xff, 0xdc, 0xf6, 0x17, 0x1c, 0x30, 0x43, 0x18, 0x8c, 0x60, 0x7e, 0x53, 0x4c,
	0x75, 0xa6, 0x56, 0xad, 0x26, 0xc2, 0xcd, 0x77, 0x32, 0x3d, 0xdf, 0x68, 0xaf, 0xd4, 0x08, 0xea,
	0xe2, 0x6f, 0x1e, 0xa4, 0x0f, 0xe1, 0x9f, 0x0a, 0x66, 0xe8, 0x69, 0xd5, 0xa8, 0x82, 0x37, 0x2c,
	0x08, 0x6c, 0x4d, 0x4a, 0x67, 0xf2, 0x7a, 0xd2, 0xe4, 0x8b, 0x43, 0x7e, 0xf8, 0x6c, 0x10, 0x7f,
	0x5c, 0x4d, 0x54, 0xff, 0x06, 0x00, 0x00, 0xff, 0xff, 0x46, 0xf5, 0x92, 0x53, 0x1a, 0x04, 0x00,
	0x00,
}
