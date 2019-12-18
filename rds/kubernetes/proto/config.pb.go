// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/rds/kubernetes/proto/config.proto

package proto

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	proto1 "github.com/google/cloudprober/common/tlsconfig/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Pods struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Pods) Reset()         { *m = Pods{} }
func (m *Pods) String() string { return proto.CompactTextString(m) }
func (*Pods) ProtoMessage()    {}
func (*Pods) Descriptor() ([]byte, []int) {
	return fileDescriptor_b730109f5f77edc1, []int{0}
}

func (m *Pods) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Pods.Unmarshal(m, b)
}
func (m *Pods) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Pods.Marshal(b, m, deterministic)
}
func (m *Pods) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Pods.Merge(m, src)
}
func (m *Pods) XXX_Size() int {
	return xxx_messageInfo_Pods.Size(m)
}
func (m *Pods) XXX_DiscardUnknown() {
	xxx_messageInfo_Pods.DiscardUnknown(m)
}

var xxx_messageInfo_Pods proto.InternalMessageInfo

type Endpoints struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Endpoints) Reset()         { *m = Endpoints{} }
func (m *Endpoints) String() string { return proto.CompactTextString(m) }
func (*Endpoints) ProtoMessage()    {}
func (*Endpoints) Descriptor() ([]byte, []int) {
	return fileDescriptor_b730109f5f77edc1, []int{1}
}

func (m *Endpoints) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Endpoints.Unmarshal(m, b)
}
func (m *Endpoints) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Endpoints.Marshal(b, m, deterministic)
}
func (m *Endpoints) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Endpoints.Merge(m, src)
}
func (m *Endpoints) XXX_Size() int {
	return xxx_messageInfo_Endpoints.Size(m)
}
func (m *Endpoints) XXX_DiscardUnknown() {
	xxx_messageInfo_Endpoints.DiscardUnknown(m)
}

var xxx_messageInfo_Endpoints proto.InternalMessageInfo

type Services struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Services) Reset()         { *m = Services{} }
func (m *Services) String() string { return proto.CompactTextString(m) }
func (*Services) ProtoMessage()    {}
func (*Services) Descriptor() ([]byte, []int) {
	return fileDescriptor_b730109f5f77edc1, []int{2}
}

func (m *Services) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Services.Unmarshal(m, b)
}
func (m *Services) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Services.Marshal(b, m, deterministic)
}
func (m *Services) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Services.Merge(m, src)
}
func (m *Services) XXX_Size() int {
	return xxx_messageInfo_Services.Size(m)
}
func (m *Services) XXX_DiscardUnknown() {
	xxx_messageInfo_Services.DiscardUnknown(m)
}

var xxx_messageInfo_Services proto.InternalMessageInfo

// Kubernetes provider config.
type ProviderConfig struct {
	// Namespace to list resources for. If not specified, we default to all
	// namespaces.
	Namespace *string `protobuf:"bytes,1,opt,name=namespace" json:"namespace,omitempty"`
	// Pods discovery options. This field should be declared for the pods
	// discovery to be enabled.
	Pods *Pods `protobuf:"bytes,2,opt,name=pods" json:"pods,omitempty"`
	// Endpoints discovery options. This field should be declared for the
	// endpoints discovery to be enabled.
	Endpoints *Endpoints `protobuf:"bytes,3,opt,name=endpoints" json:"endpoints,omitempty"`
	// Services discovery options. This field should be declared for the
	// services discovery to be enabled.
	Services *Services `protobuf:"bytes,4,opt,name=services" json:"services,omitempty"`
	// Kubernetes API server address. If not specified, we assume in-cluster mode
	// and get it from the local environment variables.
	ApiServerAddress *string `protobuf:"bytes,91,opt,name=api_server_address,json=apiServerAddress" json:"api_server_address,omitempty"`
	// TLS config to authenticate communication with the API server.
	TlsConfig *proto1.TLSConfig `protobuf:"bytes,93,opt,name=tls_config,json=tlsConfig" json:"tls_config,omitempty"`
	// How often resources should be evaluated/expanded.
	ReEvalSec            *int32   `protobuf:"varint,99,opt,name=re_eval_sec,json=reEvalSec,def=60" json:"re_eval_sec,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ProviderConfig) Reset()         { *m = ProviderConfig{} }
func (m *ProviderConfig) String() string { return proto.CompactTextString(m) }
func (*ProviderConfig) ProtoMessage()    {}
func (*ProviderConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_b730109f5f77edc1, []int{3}
}

func (m *ProviderConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ProviderConfig.Unmarshal(m, b)
}
func (m *ProviderConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ProviderConfig.Marshal(b, m, deterministic)
}
func (m *ProviderConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProviderConfig.Merge(m, src)
}
func (m *ProviderConfig) XXX_Size() int {
	return xxx_messageInfo_ProviderConfig.Size(m)
}
func (m *ProviderConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_ProviderConfig.DiscardUnknown(m)
}

var xxx_messageInfo_ProviderConfig proto.InternalMessageInfo

const Default_ProviderConfig_ReEvalSec int32 = 60

func (m *ProviderConfig) GetNamespace() string {
	if m != nil && m.Namespace != nil {
		return *m.Namespace
	}
	return ""
}

func (m *ProviderConfig) GetPods() *Pods {
	if m != nil {
		return m.Pods
	}
	return nil
}

func (m *ProviderConfig) GetEndpoints() *Endpoints {
	if m != nil {
		return m.Endpoints
	}
	return nil
}

func (m *ProviderConfig) GetServices() *Services {
	if m != nil {
		return m.Services
	}
	return nil
}

func (m *ProviderConfig) GetApiServerAddress() string {
	if m != nil && m.ApiServerAddress != nil {
		return *m.ApiServerAddress
	}
	return ""
}

func (m *ProviderConfig) GetTlsConfig() *proto1.TLSConfig {
	if m != nil {
		return m.TlsConfig
	}
	return nil
}

func (m *ProviderConfig) GetReEvalSec() int32 {
	if m != nil && m.ReEvalSec != nil {
		return *m.ReEvalSec
	}
	return Default_ProviderConfig_ReEvalSec
}

func init() {
	proto.RegisterType((*Pods)(nil), "cloudprober.rds.kubernetes.Pods")
	proto.RegisterType((*Endpoints)(nil), "cloudprober.rds.kubernetes.Endpoints")
	proto.RegisterType((*Services)(nil), "cloudprober.rds.kubernetes.Services")
	proto.RegisterType((*ProviderConfig)(nil), "cloudprober.rds.kubernetes.ProviderConfig")
}

func init() {
	proto.RegisterFile("github.com/google/cloudprober/rds/kubernetes/proto/config.proto", fileDescriptor_b730109f5f77edc1)
}

var fileDescriptor_b730109f5f77edc1 = []byte{
	// 324 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x50, 0xcf, 0x4b, 0xf3, 0x40,
	0x10, 0x25, 0xfd, 0xfa, 0x7d, 0x34, 0x53, 0xf8, 0x90, 0x3d, 0x85, 0xe2, 0x21, 0x04, 0x85, 0x1e,
	0x24, 0x11, 0x11, 0x0f, 0x5e, 0x6a, 0x29, 0xbd, 0x79, 0x28, 0x89, 0x37, 0x91, 0x90, 0xee, 0x8e,
	0x31, 0x98, 0x64, 0x96, 0x9d, 0x6d, 0xfe, 0x7c, 0x91, 0x6e, 0xfa, 0xc3, 0x43, 0xed, 0x71, 0x66,
	0xde, 0x7b, 0xf3, 0xde, 0x83, 0x59, 0x59, 0xd9, 0x8f, 0xcd, 0x3a, 0x96, 0xd4, 0x24, 0x25, 0x51,
	0x59, 0x63, 0x22, 0x6b, 0xda, 0x28, 0x6d, 0x68, 0x8d, 0x26, 0x31, 0x8a, 0x93, 0xcf, 0xcd, 0x1a,
	0x4d, 0x8b, 0x16, 0x39, 0xd1, 0x86, 0x2c, 0x25, 0x92, 0xda, 0xf7, 0xaa, 0x8c, 0xdd, 0x20, 0x26,
	0x3f, 0xe0, 0xb1, 0x51, 0x1c, 0x1f, 0xe1, 0x93, 0xf9, 0x79, 0x71, 0x49, 0x4d, 0x43, 0x6d, 0x62,
	0x6b, 0xee, 0x15, 0x4f, 0xc8, 0x47, 0xff, 0x60, 0xb8, 0x22, 0xc5, 0xd1, 0x18, 0xfc, 0x65, 0xab,
	0x34, 0x55, 0xad, 0xe5, 0x08, 0x60, 0x94, 0xa1, 0xe9, 0x2a, 0x89, 0x1c, 0x7d, 0x0d, 0xe0, 0xff,
	0xca, 0x50, 0x57, 0x29, 0x34, 0x0b, 0xc7, 0x14, 0x97, 0xe0, 0xb7, 0x45, 0x83, 0xac, 0x0b, 0x89,
	0x81, 0x17, 0x7a, 0x53, 0x3f, 0x3d, 0x2e, 0xc4, 0x3d, 0x0c, 0x35, 0x29, 0x0e, 0x06, 0xa1, 0x37,
	0x1d, 0xdf, 0x85, 0xf1, 0xef, 0xfe, 0xe3, 0xed, 0xe7, 0xd4, 0xa1, 0xc5, 0x02, 0x7c, 0xdc, 0xff,
	0x0f, 0xfe, 0x38, 0xea, 0xf5, 0x39, 0xea, 0xc1, 0x6c, 0x7a, 0xe4, 0x89, 0x27, 0x18, 0xf1, 0xce,
	0x77, 0x30, 0x74, 0x1a, 0x57, 0xe7, 0x34, 0xf6, 0x19, 0xd3, 0x03, 0x4b, 0xdc, 0x80, 0x28, 0x74,
	0x95, 0x6f, 0x67, 0x34, 0x79, 0xa1, 0x94, 0x41, 0xe6, 0xe0, 0xd5, 0x65, 0xbc, 0x28, 0x74, 0x95,
	0xb9, 0xc3, 0xbc, 0xdf, 0x8b, 0x19, 0x80, 0xad, 0x39, 0xef, 0x0b, 0x0d, 0xde, 0x4e, 0x04, 0x3e,
	0x74, 0x1f, 0xbf, 0x3c, 0x67, 0x7d, 0x7d, 0xa9, 0x6f, 0x6b, 0xde, 0x35, 0x19, 0xc1, 0xd8, 0x60,
	0x8e, 0x5d, 0x51, 0xe7, 0x8c, 0x32, 0x90, 0xa1, 0x37, 0xfd, 0xfb, 0x38, 0x78, 0xb8, 0x4d, 0x7d,
	0x83, 0xcb, 0xae, 0xa8, 0x33, 0x94, 0xdf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x78, 0x6d, 0xb2, 0x75,
	0x42, 0x02, 0x00, 0x00,
}
