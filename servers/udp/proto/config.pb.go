// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: github.com/google/cloudprober/servers/udp/proto/config.proto

package proto

import (
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

type ServerConf_Type int32

const (
	// Echos the incoming packet back.
	// Note that UDP echo server limits reads to 4098 bytes. For messages longer
	// than 4098 bytes it won't work as expected.
	ServerConf_ECHO ServerConf_Type = 0
	// Discard the incoming packet. Return nothing.
	ServerConf_DISCARD ServerConf_Type = 1
)

// Enum value maps for ServerConf_Type.
var (
	ServerConf_Type_name = map[int32]string{
		0: "ECHO",
		1: "DISCARD",
	}
	ServerConf_Type_value = map[string]int32{
		"ECHO":    0,
		"DISCARD": 1,
	}
)

func (x ServerConf_Type) Enum() *ServerConf_Type {
	p := new(ServerConf_Type)
	*p = x
	return p
}

func (x ServerConf_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ServerConf_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_google_cloudprober_servers_udp_proto_config_proto_enumTypes[0].Descriptor()
}

func (ServerConf_Type) Type() protoreflect.EnumType {
	return &file_github_com_google_cloudprober_servers_udp_proto_config_proto_enumTypes[0]
}

func (x ServerConf_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *ServerConf_Type) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = ServerConf_Type(num)
	return nil
}

// Deprecated: Use ServerConf_Type.Descriptor instead.
func (ServerConf_Type) EnumDescriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDescGZIP(), []int{0, 0}
}

type ServerConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Port *int32           `protobuf:"varint,1,req,name=port" json:"port,omitempty"`
	Type *ServerConf_Type `protobuf:"varint,2,req,name=type,enum=cloudprober.servers.udp.ServerConf_Type" json:"type,omitempty"`
}

func (x *ServerConf) Reset() {
	*x = ServerConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_servers_udp_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ServerConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerConf) ProtoMessage() {}

func (x *ServerConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_servers_udp_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerConf.ProtoReflect.Descriptor instead.
func (*ServerConf) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *ServerConf) GetPort() int32 {
	if x != nil && x.Port != nil {
		return *x.Port
	}
	return 0
}

func (x *ServerConf) GetType() ServerConf_Type {
	if x != nil && x.Type != nil {
		return *x.Type
	}
	return ServerConf_ECHO
}

var File_github_com_google_cloudprober_servers_udp_proto_config_proto protoreflect.FileDescriptor

var file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDesc = []byte{
	0x0a, 0x3c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2f, 0x75, 0x64, 0x70, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x17,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x73, 0x2e, 0x75, 0x64, 0x70, 0x22, 0x7d, 0x0a, 0x0a, 0x53, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x43, 0x6f, 0x6e, 0x66, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x01, 0x20,
	0x02, 0x28, 0x05, 0x52, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x3c, 0x0a, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x18, 0x02, 0x20, 0x02, 0x28, 0x0e, 0x32, 0x28, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2e, 0x75, 0x64,
	0x70, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x54, 0x79, 0x70,
	0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x1d, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12,
	0x08, 0x0a, 0x04, 0x45, 0x43, 0x48, 0x4f, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x44, 0x49, 0x53,
	0x43, 0x41, 0x52, 0x44, 0x10, 0x01, 0x42, 0x31, 0x5a, 0x2f, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x2f,
	0x75, 0x64, 0x70, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDescOnce sync.Once
	file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDescData = file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDesc
)

func file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDescData)
	})
	return file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDescData
}

var file_github_com_google_cloudprober_servers_udp_proto_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_google_cloudprober_servers_udp_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_github_com_google_cloudprober_servers_udp_proto_config_proto_goTypes = []interface{}{
	(ServerConf_Type)(0), // 0: cloudprober.servers.udp.ServerConf.Type
	(*ServerConf)(nil),   // 1: cloudprober.servers.udp.ServerConf
}
var file_github_com_google_cloudprober_servers_udp_proto_config_proto_depIdxs = []int32{
	0, // 0: cloudprober.servers.udp.ServerConf.type:type_name -> cloudprober.servers.udp.ServerConf.Type
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_github_com_google_cloudprober_servers_udp_proto_config_proto_init() }
func file_github_com_google_cloudprober_servers_udp_proto_config_proto_init() {
	if File_github_com_google_cloudprober_servers_udp_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_google_cloudprober_servers_udp_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ServerConf); i {
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
			RawDescriptor: file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_google_cloudprober_servers_udp_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_google_cloudprober_servers_udp_proto_config_proto_depIdxs,
		EnumInfos:         file_github_com_google_cloudprober_servers_udp_proto_config_proto_enumTypes,
		MessageInfos:      file_github_com_google_cloudprober_servers_udp_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_google_cloudprober_servers_udp_proto_config_proto = out.File
	file_github_com_google_cloudprober_servers_udp_proto_config_proto_rawDesc = nil
	file_github_com_google_cloudprober_servers_udp_proto_config_proto_goTypes = nil
	file_github_com_google_cloudprober_servers_udp_proto_config_proto_depIdxs = nil
}
