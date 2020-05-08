// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.22.0
// 	protoc        v3.11.2
// source: github.com/google/cloudprober/surfacers/proto/config.proto

package proto

import (
	proto "github.com/golang/protobuf/proto"
	proto3 "github.com/google/cloudprober/surfacers/file/proto"
	proto4 "github.com/google/cloudprober/surfacers/postgres/proto"
	proto1 "github.com/google/cloudprober/surfacers/prometheus/proto"
	proto2 "github.com/google/cloudprober/surfacers/stackdriver/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

// Enumeration for each type of surfacer we can parse and create
type Type int32

const (
	Type_NONE         Type = 0
	Type_PROMETHEUS   Type = 1
	Type_STACKDRIVER  Type = 2
	Type_FILE         Type = 3
	Type_POSTGRES     Type = 4
	Type_USER_DEFINED Type = 99
)

// Enum value maps for Type.
var (
	Type_name = map[int32]string{
		0:  "NONE",
		1:  "PROMETHEUS",
		2:  "STACKDRIVER",
		3:  "FILE",
		4:  "POSTGRES",
		99: "USER_DEFINED",
	}
	Type_value = map[string]int32{
		"NONE":         0,
		"PROMETHEUS":   1,
		"STACKDRIVER":  2,
		"FILE":         3,
		"POSTGRES":     4,
		"USER_DEFINED": 99,
	}
)

func (x Type) Enum() *Type {
	p := new(Type)
	*p = x
	return p
}

func (x Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Type) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_google_cloudprober_surfacers_proto_config_proto_enumTypes[0].Descriptor()
}

func (Type) Type() protoreflect.EnumType {
	return &file_github_com_google_cloudprober_surfacers_proto_config_proto_enumTypes[0]
}

func (x Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *Type) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = Type(num)
	return nil
}

// Deprecated: Use Type.Descriptor instead.
func (Type) EnumDescriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDescGZIP(), []int{0}
}

type SurfacerDef struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// This name is used for logging. If not defined, it's derived from the type.
	// Note that this field is required for the USER_DEFINED surfacer type and
	// should match with the name that you used while registering the user defined
	// surfacer.
	Name *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Type *Type   `protobuf:"varint,2,opt,name=type,enum=cloudprober.surfacer.Type" json:"type,omitempty"`
	// Matching surfacer specific configuration (one for each type in the above
	// enum)
	//
	// Types that are assignable to Surfacer:
	//	*SurfacerDef_PrometheusSurfacer
	//	*SurfacerDef_StackdriverSurfacer
	//	*SurfacerDef_FileSurfacer
	//	*SurfacerDef_PostgresSurfacer
	Surfacer isSurfacerDef_Surfacer `protobuf_oneof:"surfacer"`
}

func (x *SurfacerDef) Reset() {
	*x = SurfacerDef{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_surfacers_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SurfacerDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerDef) ProtoMessage() {}

func (x *SurfacerDef) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_surfacers_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SurfacerDef.ProtoReflect.Descriptor instead.
func (*SurfacerDef) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *SurfacerDef) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *SurfacerDef) GetType() Type {
	if x != nil && x.Type != nil {
		return *x.Type
	}
	return Type_NONE
}

func (m *SurfacerDef) GetSurfacer() isSurfacerDef_Surfacer {
	if m != nil {
		return m.Surfacer
	}
	return nil
}

func (x *SurfacerDef) GetPrometheusSurfacer() *proto1.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_PrometheusSurfacer); ok {
		return x.PrometheusSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetStackdriverSurfacer() *proto2.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_StackdriverSurfacer); ok {
		return x.StackdriverSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetFileSurfacer() *proto3.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_FileSurfacer); ok {
		return x.FileSurfacer
	}
	return nil
}

func (x *SurfacerDef) GetPostgresSurfacer() *proto4.SurfacerConf {
	if x, ok := x.GetSurfacer().(*SurfacerDef_PostgresSurfacer); ok {
		return x.PostgresSurfacer
	}
	return nil
}

type isSurfacerDef_Surfacer interface {
	isSurfacerDef_Surfacer()
}

type SurfacerDef_PrometheusSurfacer struct {
	PrometheusSurfacer *proto1.SurfacerConf `protobuf:"bytes,10,opt,name=prometheus_surfacer,json=prometheusSurfacer,oneof"`
}

type SurfacerDef_StackdriverSurfacer struct {
	StackdriverSurfacer *proto2.SurfacerConf `protobuf:"bytes,11,opt,name=stackdriver_surfacer,json=stackdriverSurfacer,oneof"`
}

type SurfacerDef_FileSurfacer struct {
	FileSurfacer *proto3.SurfacerConf `protobuf:"bytes,12,opt,name=file_surfacer,json=fileSurfacer,oneof"`
}

type SurfacerDef_PostgresSurfacer struct {
	PostgresSurfacer *proto4.SurfacerConf `protobuf:"bytes,13,opt,name=postgres_surfacer,json=postgresSurfacer,oneof"`
}

func (*SurfacerDef_PrometheusSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_StackdriverSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_FileSurfacer) isSurfacerDef_Surfacer() {}

func (*SurfacerDef_PostgresSurfacer) isSurfacerDef_Surfacer() {}

var File_github_com_google_cloudprober_surfacers_proto_config_proto protoreflect.FileDescriptor

var file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDesc = []byte{
	0x0a, 0x3a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x14, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x1a, 0x3f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x43, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x70, 0x6f, 0x73,
	0x74, 0x67, 0x72, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x45, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72,
	0x73, 0x2f, 0x70, 0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73,
	0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72,
	0x69, 0x76, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xd0, 0x03, 0x0a, 0x0b, 0x53, 0x75, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x72, 0x44, 0x65, 0x66, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x2e, 0x0a, 0x04, 0x74,
	0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1a, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72,
	0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x60, 0x0a, 0x13, 0x70,
	0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x5f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e,
	0x70, 0x72, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x65, 0x75, 0x73, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61,
	0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x12, 0x70, 0x72, 0x6f, 0x6d, 0x65,
	0x74, 0x68, 0x65, 0x75, 0x73, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x12, 0x63, 0x0a,
	0x14, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72, 0x5f, 0x73, 0x75, 0x72,
	0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x2e, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72, 0x2e, 0x53,
	0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x13, 0x73,
	0x74, 0x61, 0x63, 0x6b, 0x64, 0x72, 0x69, 0x76, 0x65, 0x72, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x12, 0x4e, 0x0a, 0x0d, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x73, 0x75, 0x72, 0x66, 0x61,
	0x63, 0x65, 0x72, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72,
	0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f,
	0x6e, 0x66, 0x48, 0x00, 0x52, 0x0c, 0x66, 0x69, 0x6c, 0x65, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63,
	0x65, 0x72, 0x12, 0x5a, 0x0a, 0x11, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x5f, 0x73,
	0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2b, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66,
	0x61, 0x63, 0x65, 0x72, 0x2e, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x2e, 0x53, 0x75,
	0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x10, 0x70, 0x6f,
	0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x42, 0x0a,
	0x0a, 0x08, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2a, 0x5b, 0x0a, 0x04, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x4e, 0x4f, 0x4e, 0x45, 0x10, 0x00, 0x12, 0x0e, 0x0a, 0x0a,
	0x50, 0x52, 0x4f, 0x4d, 0x45, 0x54, 0x48, 0x45, 0x55, 0x53, 0x10, 0x01, 0x12, 0x0f, 0x0a, 0x0b,
	0x53, 0x54, 0x41, 0x43, 0x4b, 0x44, 0x52, 0x49, 0x56, 0x45, 0x52, 0x10, 0x02, 0x12, 0x08, 0x0a,
	0x04, 0x46, 0x49, 0x4c, 0x45, 0x10, 0x03, 0x12, 0x0c, 0x0a, 0x08, 0x50, 0x4f, 0x53, 0x54, 0x47,
	0x52, 0x45, 0x53, 0x10, 0x04, 0x12, 0x10, 0x0a, 0x0c, 0x55, 0x53, 0x45, 0x52, 0x5f, 0x44, 0x45,
	0x46, 0x49, 0x4e, 0x45, 0x44, 0x10, 0x63, 0x42, 0x2f, 0x5a, 0x2d, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65,
	0x72, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDescOnce sync.Once
	file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDescData = file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDesc
)

func file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDescData)
	})
	return file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDescData
}

var file_github_com_google_cloudprober_surfacers_proto_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_google_cloudprober_surfacers_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_google_cloudprober_surfacers_proto_config_proto_goTypes = []interface{}{
	(Type)(0),                   // 0: cloudprober.surfacer.Type
	(*SurfacerDef)(nil),         // 1: cloudprober.surfacer.SurfacerDef
	(*proto1.SurfacerConf)(nil), // 2: cloudprober.surfacer.prometheus.SurfacerConf
	(*proto2.SurfacerConf)(nil), // 3: cloudprober.surfacer.stackdriver.SurfacerConf
	(*proto3.SurfacerConf)(nil), // 4: cloudprober.surfacer.file.SurfacerConf
	(*proto4.SurfacerConf)(nil), // 5: cloudprober.surfacer.postgres.SurfacerConf
}
var file_github_com_google_cloudprober_surfacers_proto_config_proto_depIdxs = []int32{
	0, // 0: cloudprober.surfacer.SurfacerDef.type:type_name -> cloudprober.surfacer.Type
	2, // 1: cloudprober.surfacer.SurfacerDef.prometheus_surfacer:type_name -> cloudprober.surfacer.prometheus.SurfacerConf
	3, // 2: cloudprober.surfacer.SurfacerDef.stackdriver_surfacer:type_name -> cloudprober.surfacer.stackdriver.SurfacerConf
	4, // 3: cloudprober.surfacer.SurfacerDef.file_surfacer:type_name -> cloudprober.surfacer.file.SurfacerConf
	5, // 4: cloudprober.surfacer.SurfacerDef.postgres_surfacer:type_name -> cloudprober.surfacer.postgres.SurfacerConf
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_github_com_google_cloudprober_surfacers_proto_config_proto_init() }
func file_github_com_google_cloudprober_surfacers_proto_config_proto_init() {
	if File_github_com_google_cloudprober_surfacers_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_google_cloudprober_surfacers_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SurfacerDef); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_github_com_google_cloudprober_surfacers_proto_config_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*SurfacerDef_PrometheusSurfacer)(nil),
		(*SurfacerDef_StackdriverSurfacer)(nil),
		(*SurfacerDef_FileSurfacer)(nil),
		(*SurfacerDef_PostgresSurfacer)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_google_cloudprober_surfacers_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_google_cloudprober_surfacers_proto_config_proto_depIdxs,
		EnumInfos:         file_github_com_google_cloudprober_surfacers_proto_config_proto_enumTypes,
		MessageInfos:      file_github_com_google_cloudprober_surfacers_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_google_cloudprober_surfacers_proto_config_proto = out.File
	file_github_com_google_cloudprober_surfacers_proto_config_proto_rawDesc = nil
	file_github_com_google_cloudprober_surfacers_proto_config_proto_goTypes = nil
	file_github_com_google_cloudprober_surfacers_proto_config_proto_depIdxs = nil
}
