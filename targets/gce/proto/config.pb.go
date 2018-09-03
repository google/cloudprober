// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/targets/gce/proto/config.proto

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

type Instances_NetworkInterface_IPType int32

const (
	// Private IP address.
	Instances_NetworkInterface_PRIVATE Instances_NetworkInterface_IPType = 0
	// IP address of the first access config.
	Instances_NetworkInterface_PUBLIC Instances_NetworkInterface_IPType = 1
	// First IP address from the first Alias IP range. For example, for
	// alias IP range "192.168.12.0/24", 192.168.12.0 will be returned.
	Instances_NetworkInterface_ALIAS Instances_NetworkInterface_IPType = 2
)

var Instances_NetworkInterface_IPType_name = map[int32]string{
	0: "PRIVATE",
	1: "PUBLIC",
	2: "ALIAS",
}
var Instances_NetworkInterface_IPType_value = map[string]int32{
	"PRIVATE": 0,
	"PUBLIC":  1,
	"ALIAS":   2,
}

func (x Instances_NetworkInterface_IPType) Enum() *Instances_NetworkInterface_IPType {
	p := new(Instances_NetworkInterface_IPType)
	*p = x
	return p
}
func (x Instances_NetworkInterface_IPType) String() string {
	return proto.EnumName(Instances_NetworkInterface_IPType_name, int32(x))
}
func (x *Instances_NetworkInterface_IPType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Instances_NetworkInterface_IPType_value, data, "Instances_NetworkInterface_IPType")
	if err != nil {
		return err
	}
	*x = Instances_NetworkInterface_IPType(value)
	return nil
}
func (Instances_NetworkInterface_IPType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_config_cc3819054e5449da, []int{1, 0, 0}
}

// TargetsConf represents GCE targets, e.g. instances, forwarding rules etc.
type TargetsConf struct {
	// If running on GCE, this defaults to the local project.
	// Note: Multiple projects support in targets is experimental and may go away
	// with future iterations.
	Project []string `protobuf:"bytes,1,rep,name=project" json:"project,omitempty"`
	// Types that are valid to be assigned to Type:
	//	*TargetsConf_Instances
	//	*TargetsConf_ForwardingRules
	Type                 isTargetsConf_Type `protobuf_oneof:"type"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *TargetsConf) Reset()         { *m = TargetsConf{} }
func (m *TargetsConf) String() string { return proto.CompactTextString(m) }
func (*TargetsConf) ProtoMessage()    {}
func (*TargetsConf) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_cc3819054e5449da, []int{0}
}
func (m *TargetsConf) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TargetsConf.Unmarshal(m, b)
}
func (m *TargetsConf) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TargetsConf.Marshal(b, m, deterministic)
}
func (dst *TargetsConf) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TargetsConf.Merge(dst, src)
}
func (m *TargetsConf) XXX_Size() int {
	return xxx_messageInfo_TargetsConf.Size(m)
}
func (m *TargetsConf) XXX_DiscardUnknown() {
	xxx_messageInfo_TargetsConf.DiscardUnknown(m)
}

var xxx_messageInfo_TargetsConf proto.InternalMessageInfo

func (m *TargetsConf) GetProject() []string {
	if m != nil {
		return m.Project
	}
	return nil
}

type isTargetsConf_Type interface {
	isTargetsConf_Type()
}

type TargetsConf_Instances struct {
	Instances *Instances `protobuf:"bytes,2,opt,name=instances,oneof"`
}

type TargetsConf_ForwardingRules struct {
	ForwardingRules *ForwardingRules `protobuf:"bytes,3,opt,name=forwarding_rules,json=forwardingRules,oneof"`
}

func (*TargetsConf_Instances) isTargetsConf_Type() {}

func (*TargetsConf_ForwardingRules) isTargetsConf_Type() {}

func (m *TargetsConf) GetType() isTargetsConf_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *TargetsConf) GetInstances() *Instances {
	if x, ok := m.GetType().(*TargetsConf_Instances); ok {
		return x.Instances
	}
	return nil
}

func (m *TargetsConf) GetForwardingRules() *ForwardingRules {
	if x, ok := m.GetType().(*TargetsConf_ForwardingRules); ok {
		return x.ForwardingRules
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*TargetsConf) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _TargetsConf_OneofMarshaler, _TargetsConf_OneofUnmarshaler, _TargetsConf_OneofSizer, []interface{}{
		(*TargetsConf_Instances)(nil),
		(*TargetsConf_ForwardingRules)(nil),
	}
}

func _TargetsConf_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*TargetsConf)
	// type
	switch x := m.Type.(type) {
	case *TargetsConf_Instances:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Instances); err != nil {
			return err
		}
	case *TargetsConf_ForwardingRules:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.ForwardingRules); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("TargetsConf.Type has unexpected type %T", x)
	}
	return nil
}

func _TargetsConf_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*TargetsConf)
	switch tag {
	case 2: // type.instances
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(Instances)
		err := b.DecodeMessage(msg)
		m.Type = &TargetsConf_Instances{msg}
		return true, err
	case 3: // type.forwarding_rules
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(ForwardingRules)
		err := b.DecodeMessage(msg)
		m.Type = &TargetsConf_ForwardingRules{msg}
		return true, err
	default:
		return false, nil
	}
}

func _TargetsConf_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*TargetsConf)
	// type
	switch x := m.Type.(type) {
	case *TargetsConf_Instances:
		s := proto.Size(x.Instances)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *TargetsConf_ForwardingRules:
		s := proto.Size(x.ForwardingRules)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

// Represents GCE instances
type Instances struct {
	// Use DNS to resolve target names (instances). If set to false (default),
	// IP addresses specified in the compute.Instance resource is used. If set
	// to true all the other resolving options are ignored.
	UseDnsToResolve      *bool                       `protobuf:"varint,1,opt,name=use_dns_to_resolve,json=useDnsToResolve,def=0" json:"use_dns_to_resolve,omitempty"`
	NetworkInterface     *Instances_NetworkInterface `protobuf:"bytes,2,opt,name=network_interface,json=networkInterface" json:"network_interface,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                    `json:"-"`
	XXX_unrecognized     []byte                      `json:"-"`
	XXX_sizecache        int32                       `json:"-"`
}

func (m *Instances) Reset()         { *m = Instances{} }
func (m *Instances) String() string { return proto.CompactTextString(m) }
func (*Instances) ProtoMessage()    {}
func (*Instances) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_cc3819054e5449da, []int{1}
}
func (m *Instances) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Instances.Unmarshal(m, b)
}
func (m *Instances) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Instances.Marshal(b, m, deterministic)
}
func (dst *Instances) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Instances.Merge(dst, src)
}
func (m *Instances) XXX_Size() int {
	return xxx_messageInfo_Instances.Size(m)
}
func (m *Instances) XXX_DiscardUnknown() {
	xxx_messageInfo_Instances.DiscardUnknown(m)
}

var xxx_messageInfo_Instances proto.InternalMessageInfo

const Default_Instances_UseDnsToResolve bool = false

func (m *Instances) GetUseDnsToResolve() bool {
	if m != nil && m.UseDnsToResolve != nil {
		return *m.UseDnsToResolve
	}
	return Default_Instances_UseDnsToResolve
}

func (m *Instances) GetNetworkInterface() *Instances_NetworkInterface {
	if m != nil {
		return m.NetworkInterface
	}
	return nil
}

// Get the IP address from Network Interface
type Instances_NetworkInterface struct {
	Index                *int32                             `protobuf:"varint,1,opt,name=index,def=0" json:"index,omitempty"`
	IpType               *Instances_NetworkInterface_IPType `protobuf:"varint,2,opt,name=ip_type,json=ipType,enum=cloudprober.targets.gce.Instances_NetworkInterface_IPType,def=0" json:"ip_type,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                           `json:"-"`
	XXX_unrecognized     []byte                             `json:"-"`
	XXX_sizecache        int32                              `json:"-"`
}

func (m *Instances_NetworkInterface) Reset()         { *m = Instances_NetworkInterface{} }
func (m *Instances_NetworkInterface) String() string { return proto.CompactTextString(m) }
func (*Instances_NetworkInterface) ProtoMessage()    {}
func (*Instances_NetworkInterface) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_cc3819054e5449da, []int{1, 0}
}
func (m *Instances_NetworkInterface) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Instances_NetworkInterface.Unmarshal(m, b)
}
func (m *Instances_NetworkInterface) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Instances_NetworkInterface.Marshal(b, m, deterministic)
}
func (dst *Instances_NetworkInterface) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Instances_NetworkInterface.Merge(dst, src)
}
func (m *Instances_NetworkInterface) XXX_Size() int {
	return xxx_messageInfo_Instances_NetworkInterface.Size(m)
}
func (m *Instances_NetworkInterface) XXX_DiscardUnknown() {
	xxx_messageInfo_Instances_NetworkInterface.DiscardUnknown(m)
}

var xxx_messageInfo_Instances_NetworkInterface proto.InternalMessageInfo

const Default_Instances_NetworkInterface_Index int32 = 0
const Default_Instances_NetworkInterface_IpType Instances_NetworkInterface_IPType = Instances_NetworkInterface_PRIVATE

func (m *Instances_NetworkInterface) GetIndex() int32 {
	if m != nil && m.Index != nil {
		return *m.Index
	}
	return Default_Instances_NetworkInterface_Index
}

func (m *Instances_NetworkInterface) GetIpType() Instances_NetworkInterface_IPType {
	if m != nil && m.IpType != nil {
		return *m.IpType
	}
	return Default_Instances_NetworkInterface_IpType
}

// Represents GCE forwarding rules. Does not support multiple projects
type ForwardingRules struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ForwardingRules) Reset()         { *m = ForwardingRules{} }
func (m *ForwardingRules) String() string { return proto.CompactTextString(m) }
func (*ForwardingRules) ProtoMessage()    {}
func (*ForwardingRules) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_cc3819054e5449da, []int{2}
}
func (m *ForwardingRules) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ForwardingRules.Unmarshal(m, b)
}
func (m *ForwardingRules) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ForwardingRules.Marshal(b, m, deterministic)
}
func (dst *ForwardingRules) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ForwardingRules.Merge(dst, src)
}
func (m *ForwardingRules) XXX_Size() int {
	return xxx_messageInfo_ForwardingRules.Size(m)
}
func (m *ForwardingRules) XXX_DiscardUnknown() {
	xxx_messageInfo_ForwardingRules.DiscardUnknown(m)
}

var xxx_messageInfo_ForwardingRules proto.InternalMessageInfo

// Global GCE targets options. These options are independent of the per-probe
// targets which are defined by the "GCETargets" type above.
type GlobalOptions struct {
	// How often targets should be evaluated/expanded
	ReEvalSec *int32 `protobuf:"varint,1,opt,name=re_eval_sec,json=reEvalSec,def=900" json:"re_eval_sec,omitempty"`
	// Compute API version.
	ApiVersion           *string  `protobuf:"bytes,2,opt,name=api_version,json=apiVersion,def=v1" json:"api_version,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GlobalOptions) Reset()         { *m = GlobalOptions{} }
func (m *GlobalOptions) String() string { return proto.CompactTextString(m) }
func (*GlobalOptions) ProtoMessage()    {}
func (*GlobalOptions) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_cc3819054e5449da, []int{3}
}
func (m *GlobalOptions) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GlobalOptions.Unmarshal(m, b)
}
func (m *GlobalOptions) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GlobalOptions.Marshal(b, m, deterministic)
}
func (dst *GlobalOptions) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GlobalOptions.Merge(dst, src)
}
func (m *GlobalOptions) XXX_Size() int {
	return xxx_messageInfo_GlobalOptions.Size(m)
}
func (m *GlobalOptions) XXX_DiscardUnknown() {
	xxx_messageInfo_GlobalOptions.DiscardUnknown(m)
}

var xxx_messageInfo_GlobalOptions proto.InternalMessageInfo

const Default_GlobalOptions_ReEvalSec int32 = 900
const Default_GlobalOptions_ApiVersion string = "v1"

func (m *GlobalOptions) GetReEvalSec() int32 {
	if m != nil && m.ReEvalSec != nil {
		return *m.ReEvalSec
	}
	return Default_GlobalOptions_ReEvalSec
}

func (m *GlobalOptions) GetApiVersion() string {
	if m != nil && m.ApiVersion != nil {
		return *m.ApiVersion
	}
	return Default_GlobalOptions_ApiVersion
}

func init() {
	proto.RegisterType((*TargetsConf)(nil), "cloudprober.targets.gce.TargetsConf")
	proto.RegisterType((*Instances)(nil), "cloudprober.targets.gce.Instances")
	proto.RegisterType((*Instances_NetworkInterface)(nil), "cloudprober.targets.gce.Instances.NetworkInterface")
	proto.RegisterType((*ForwardingRules)(nil), "cloudprober.targets.gce.ForwardingRules")
	proto.RegisterType((*GlobalOptions)(nil), "cloudprober.targets.gce.GlobalOptions")
	proto.RegisterEnum("cloudprober.targets.gce.Instances_NetworkInterface_IPType", Instances_NetworkInterface_IPType_name, Instances_NetworkInterface_IPType_value)
}

func init() {
	proto.RegisterFile("github.com/google/cloudprober/targets/gce/proto/config.proto", fileDescriptor_config_cc3819054e5449da)
}

var fileDescriptor_config_cc3819054e5449da = []byte{
	// 446 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x91, 0xd1, 0x6e, 0xd3, 0x30,
	0x14, 0x86, 0x9b, 0x96, 0xb6, 0xe4, 0x54, 0xd0, 0xcc, 0x37, 0xab, 0xb8, 0xaa, 0xc2, 0x4d, 0x2f,
	0x50, 0x52, 0xca, 0x15, 0x15, 0x37, 0xed, 0x18, 0x2c, 0xd2, 0x04, 0x95, 0xd7, 0x4d, 0x42, 0x42,
	0x0a, 0xae, 0x7b, 0x12, 0x0c, 0xc1, 0xb6, 0x6c, 0x37, 0x63, 0x4f, 0xc6, 0x3b, 0xf0, 0x0a, 0xbc,
	0x0c, 0x6a, 0xd2, 0x0e, 0xa8, 0x34, 0x21, 0xed, 0xf2, 0xff, 0xcf, 0xf7, 0x1f, 0x9f, 0x5f, 0x86,
	0x57, 0xb9, 0x70, 0x9f, 0x37, 0xab, 0x88, 0xab, 0x6f, 0x71, 0xae, 0x54, 0x5e, 0x60, 0xcc, 0x0b,
	0xb5, 0x59, 0x6b, 0xa3, 0x56, 0x68, 0x62, 0xc7, 0x4c, 0x8e, 0xce, 0xc6, 0x39, 0xc7, 0x58, 0x1b,
	0xe5, 0x54, 0xcc, 0x95, 0xcc, 0x44, 0x1e, 0x55, 0x82, 0x1c, 0xff, 0xc5, 0x46, 0x3b, 0x36, 0xca,
	0x39, 0x86, 0x3f, 0x3d, 0xe8, 0x2d, 0x6b, 0x7d, 0xa2, 0x64, 0x46, 0x06, 0xd0, 0xd5, 0x46, 0x7d,
	0x41, 0xee, 0x06, 0xde, 0xb0, 0x35, 0xf2, 0xe9, 0x5e, 0x92, 0x39, 0xf8, 0x42, 0x5a, 0xc7, 0x24,
	0x47, 0x3b, 0x68, 0x0e, 0xbd, 0x51, 0x6f, 0x12, 0x46, 0x77, 0xac, 0x8d, 0x92, 0x3d, 0x79, 0xd6,
	0xa0, 0x7f, 0x62, 0xe4, 0x12, 0x82, 0x4c, 0x99, 0x6b, 0x66, 0xd6, 0x42, 0xe6, 0xa9, 0xd9, 0x14,
	0x68, 0x07, 0xad, 0x6a, 0xd5, 0xe8, 0xce, 0x55, 0x6f, 0x6e, 0x03, 0x74, 0xcb, 0x9f, 0x35, 0x68,
	0x3f, 0xfb, 0xd7, 0x9a, 0x77, 0xe0, 0x81, 0xbb, 0xd1, 0x18, 0xfe, 0x6a, 0x82, 0x7f, 0xfb, 0x32,
	0x99, 0x00, 0xd9, 0x58, 0x4c, 0xd7, 0xd2, 0xa6, 0x4e, 0xa5, 0x06, 0xad, 0x2a, 0x4a, 0x1c, 0x78,
	0x43, 0x6f, 0xf4, 0x70, 0xda, 0xce, 0x58, 0x61, 0x91, 0xf6, 0x37, 0x16, 0x5f, 0x4b, 0xbb, 0x54,
	0xb4, 0x9e, 0x92, 0x4f, 0x70, 0x24, 0xd1, 0x5d, 0x2b, 0xf3, 0x35, 0x15, 0xd2, 0xa1, 0xc9, 0x18,
	0xc7, 0x5d, 0xd9, 0x17, 0xff, 0x2f, 0x1b, 0xbd, 0xab, 0xb3, 0xc9, 0x3e, 0x4a, 0x03, 0x79, 0xe0,
	0x3c, 0xf9, 0xe1, 0x41, 0x70, 0x88, 0x91, 0x63, 0x68, 0x0b, 0xb9, 0xc6, 0xef, 0xd5, 0x75, 0xed,
	0xa9, 0x37, 0xa6, 0xb5, 0x26, 0x1f, 0xa1, 0x2b, 0x74, 0xba, 0x2d, 0x57, 0x5d, 0xf1, 0x78, 0x32,
	0xbd, 0xc7, 0x15, 0x51, 0xb2, 0x58, 0xde, 0x68, 0x9c, 0x76, 0x17, 0x34, 0xb9, 0x9a, 0x2d, 0x4f,
	0x69, 0x47, 0xe8, 0xad, 0x11, 0x3e, 0x83, 0x4e, 0x3d, 0x22, 0x3d, 0xd8, 0x0f, 0x83, 0x06, 0x01,
	0xe8, 0x2c, 0x2e, 0xe7, 0xe7, 0xc9, 0x49, 0xe0, 0x11, 0x1f, 0xda, 0xb3, 0xf3, 0x64, 0x76, 0x11,
	0x34, 0xc3, 0x23, 0xe8, 0x1f, 0xfc, 0x45, 0xf8, 0x01, 0x1e, 0xbd, 0x2d, 0xd4, 0x8a, 0x15, 0xef,
	0xb5, 0x13, 0x4a, 0x5a, 0xf2, 0x14, 0x7a, 0x06, 0x53, 0x2c, 0x59, 0x91, 0x5a, 0xe4, 0xbb, 0x3a,
	0xad, 0x97, 0xe3, 0x31, 0xf5, 0x0d, 0x9e, 0x96, 0xac, 0xb8, 0x40, 0xbe, 0x85, 0x98, 0x16, 0x69,
	0x89, 0xc6, 0x0a, 0x25, 0xab, 0x62, 0xfe, 0xb4, 0x59, 0x3e, 0xa7, 0xc0, 0xb4, 0xb8, 0xaa, 0xdd,
	0xdf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x49, 0x50, 0x3f, 0x19, 0xf0, 0x02, 0x00, 0x00,
}
