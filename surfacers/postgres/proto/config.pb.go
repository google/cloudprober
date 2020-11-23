// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.11.2
// source: github.com/google/cloudprober/surfacers/postgres/proto/config.proto

package proto

import (
	proto "github.com/golang/protobuf/proto"
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

type SurfacerConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ConnectionString  *string `protobuf:"bytes,1,req,name=connection_string,json=connectionString" json:"connection_string,omitempty"`
	MetricsTableName  *string `protobuf:"bytes,2,req,name=metrics_table_name,json=metricsTableName" json:"metrics_table_name,omitempty"`
	MetricsBufferSize *int64  `protobuf:"varint,3,opt,name=metrics_buffer_size,json=metricsBufferSize,def=10000" json:"metrics_buffer_size,omitempty"`
}

// Default values for SurfacerConf fields.
const (
	Default_SurfacerConf_MetricsBufferSize = int64(10000)
)

func (x *SurfacerConf) Reset() {
	*x = SurfacerConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SurfacerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SurfacerConf) ProtoMessage() {}

func (x *SurfacerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SurfacerConf.ProtoReflect.Descriptor instead.
func (*SurfacerConf) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *SurfacerConf) GetConnectionString() string {
	if x != nil && x.ConnectionString != nil {
		return *x.ConnectionString
	}
	return ""
}

func (x *SurfacerConf) GetMetricsTableName() string {
	if x != nil && x.MetricsTableName != nil {
		return *x.MetricsTableName
	}
	return ""
}

func (x *SurfacerConf) GetMetricsBufferSize() int64 {
	if x != nil && x.MetricsBufferSize != nil {
		return *x.MetricsBufferSize
	}
	return Default_SurfacerConf_MetricsBufferSize
}

var File_github_com_google_cloudprober_surfacers_postgres_proto_config_proto protoreflect.FileDescriptor

var file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDesc = []byte{
	0x0a, 0x43, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x73, 0x2f, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72,
	0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1d, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2e, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65, 0x72, 0x2e, 0x70, 0x6f, 0x73, 0x74,
	0x67, 0x72, 0x65, 0x73, 0x22, 0xa0, 0x01, 0x0a, 0x0c, 0x53, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65,
	0x72, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x2b, 0x0a, 0x11, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09,
	0x52, 0x10, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x72, 0x69,
	0x6e, 0x67, 0x12, 0x2c, 0x0a, 0x12, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x74, 0x61,
	0x62, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x02, 0x28, 0x09, 0x52, 0x10,
	0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x4e, 0x61, 0x6d, 0x65,
	0x12, 0x35, 0x0a, 0x13, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x5f, 0x62, 0x75, 0x66, 0x66,
	0x65, 0x72, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x3a, 0x05, 0x31,
	0x30, 0x30, 0x30, 0x30, 0x52, 0x11, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x42, 0x75, 0x66,
	0x66, 0x65, 0x72, 0x53, 0x69, 0x7a, 0x65, 0x42, 0x38, 0x5a, 0x36, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x75, 0x72, 0x66, 0x61, 0x63, 0x65,
	0x72, 0x73, 0x2f, 0x70, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f,
}

var (
	file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDescOnce sync.Once
	file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDescData = file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDesc
)

func file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDescData)
	})
	return file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDescData
}

var file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_goTypes = []interface{}{
	(*SurfacerConf)(nil), // 0: cloudprober.surfacer.postgres.SurfacerConf
}
var file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_init() }
func file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_init() {
	if File_github_com_google_cloudprober_surfacers_postgres_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SurfacerConf); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_depIdxs,
		MessageInfos:      file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_google_cloudprober_surfacers_postgres_proto_config_proto = out.File
	file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_rawDesc = nil
	file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_goTypes = nil
	file_github_com_google_cloudprober_surfacers_postgres_proto_config_proto_depIdxs = nil
}
