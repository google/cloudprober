// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.22.0
// 	protoc        v3.11.2
// source: github.com/google/cloudprober/probes/grpc/proto/config.proto

package proto

import (
	proto "github.com/golang/protobuf/proto"
	proto1 "github.com/google/cloudprober/common/oauth/proto"
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

type ProbeConf_MethodType int32

const (
	ProbeConf_ECHO  ProbeConf_MethodType = 1
	ProbeConf_READ  ProbeConf_MethodType = 2
	ProbeConf_WRITE ProbeConf_MethodType = 3
)

// Enum value maps for ProbeConf_MethodType.
var (
	ProbeConf_MethodType_name = map[int32]string{
		1: "ECHO",
		2: "READ",
		3: "WRITE",
	}
	ProbeConf_MethodType_value = map[string]int32{
		"ECHO":  1,
		"READ":  2,
		"WRITE": 3,
	}
)

func (x ProbeConf_MethodType) Enum() *ProbeConf_MethodType {
	p := new(ProbeConf_MethodType)
	*p = x
	return p
}

func (x ProbeConf_MethodType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ProbeConf_MethodType) Descriptor() protoreflect.EnumDescriptor {
	return file_github_com_google_cloudprober_probes_grpc_proto_config_proto_enumTypes[0].Descriptor()
}

func (ProbeConf_MethodType) Type() protoreflect.EnumType {
	return &file_github_com_google_cloudprober_probes_grpc_proto_config_proto_enumTypes[0]
}

func (x ProbeConf_MethodType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *ProbeConf_MethodType) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = ProbeConf_MethodType(num)
	return nil
}

// Deprecated: Use ProbeConf_MethodType.Descriptor instead.
func (ProbeConf_MethodType) EnumDescriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescGZIP(), []int{0, 0}
}

type ProbeConf struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Optional oauth config. For GOOGLE_DEFAULT_CREDENTIALS, use:
	// oauth_config: { bearer_token { gce_service_account: "default" } }
	OauthConfig *proto1.Config `protobuf:"bytes,1,opt,name=oauth_config,json=oauthConfig" json:"oauth_config,omitempty"`
	// If alts_config is provided, gRPC client uses ALTS for authentication and
	// encryption. For default alts configs, use:
	// alts_config: {}
	AltsConfig *ProbeConf_ALTSConfig `protobuf:"bytes,2,opt,name=alts_config,json=altsConfig" json:"alts_config,omitempty"`
	Method     *ProbeConf_MethodType `protobuf:"varint,3,opt,name=method,enum=cloudprober.probes.grpc.ProbeConf_MethodType,def=1" json:"method,omitempty"`
	BlobSize   *int32                `protobuf:"varint,4,opt,name=blob_size,json=blobSize,def=1024" json:"blob_size,omitempty"`
	NumConns   *int32                `protobuf:"varint,5,opt,name=num_conns,json=numConns,def=2" json:"num_conns,omitempty"`
	KeepAlive  *bool                 `protobuf:"varint,6,opt,name=keep_alive,json=keepAlive,def=1" json:"keep_alive,omitempty"`
	// If connect_timeout is not specified, reuse probe timeout.
	ConnectTimeoutMsec *int32 `protobuf:"varint,7,opt,name=connect_timeout_msec,json=connectTimeoutMsec" json:"connect_timeout_msec,omitempty"`
}

// Default values for ProbeConf fields.
const (
	Default_ProbeConf_Method    = ProbeConf_ECHO
	Default_ProbeConf_BlobSize  = int32(1024)
	Default_ProbeConf_NumConns  = int32(2)
	Default_ProbeConf_KeepAlive = bool(true)
)

func (x *ProbeConf) Reset() {
	*x = ProbeConf{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_probes_grpc_proto_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProbeConf) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProbeConf) ProtoMessage() {}

func (x *ProbeConf) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_probes_grpc_proto_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProbeConf.ProtoReflect.Descriptor instead.
func (*ProbeConf) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescGZIP(), []int{0}
}

func (x *ProbeConf) GetOauthConfig() *proto1.Config {
	if x != nil {
		return x.OauthConfig
	}
	return nil
}

func (x *ProbeConf) GetAltsConfig() *ProbeConf_ALTSConfig {
	if x != nil {
		return x.AltsConfig
	}
	return nil
}

func (x *ProbeConf) GetMethod() ProbeConf_MethodType {
	if x != nil && x.Method != nil {
		return *x.Method
	}
	return Default_ProbeConf_Method
}

func (x *ProbeConf) GetBlobSize() int32 {
	if x != nil && x.BlobSize != nil {
		return *x.BlobSize
	}
	return Default_ProbeConf_BlobSize
}

func (x *ProbeConf) GetNumConns() int32 {
	if x != nil && x.NumConns != nil {
		return *x.NumConns
	}
	return Default_ProbeConf_NumConns
}

func (x *ProbeConf) GetKeepAlive() bool {
	if x != nil && x.KeepAlive != nil {
		return *x.KeepAlive
	}
	return Default_ProbeConf_KeepAlive
}

func (x *ProbeConf) GetConnectTimeoutMsec() int32 {
	if x != nil && x.ConnectTimeoutMsec != nil {
		return *x.ConnectTimeoutMsec
	}
	return 0
}

// ALTS is a gRPC security method supported by some Google services.
// If enabled, peers, with the help of a handshaker service (e.g. metadata
// server of GCE instances), use credentials attached to the service accounts
// to authenticate each other. See
// https://cloud.google.com/security/encryption-in-transit/#service_integrity_encryption
// for more details.
type ProbeConf_ALTSConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// If provided, ALTS verifies that peer is using one of the given service
	// accounts.
	TargetServiceAccount []string `protobuf:"bytes,1,rep,name=target_service_account,json=targetServiceAccount" json:"target_service_account,omitempty"`
	// Handshaker service address. Default is to use the local metadata server.
	// For most of the ALTS use cases, default address should be okay.
	HandshakerServiceAddress *string `protobuf:"bytes,2,opt,name=handshaker_service_address,json=handshakerServiceAddress" json:"handshaker_service_address,omitempty"`
}

func (x *ProbeConf_ALTSConfig) Reset() {
	*x = ProbeConf_ALTSConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_probes_grpc_proto_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProbeConf_ALTSConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProbeConf_ALTSConfig) ProtoMessage() {}

func (x *ProbeConf_ALTSConfig) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_probes_grpc_proto_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProbeConf_ALTSConfig.ProtoReflect.Descriptor instead.
func (*ProbeConf_ALTSConfig) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescGZIP(), []int{0, 0}
}

func (x *ProbeConf_ALTSConfig) GetTargetServiceAccount() []string {
	if x != nil {
		return x.TargetServiceAccount
	}
	return nil
}

func (x *ProbeConf_ALTSConfig) GetHandshakerServiceAddress() string {
	if x != nil && x.HandshakerServiceAddress != nil {
		return *x.HandshakerServiceAddress
	}
	return ""
}

var File_github_com_google_cloudprober_probes_grpc_proto_config_proto protoreflect.FileDescriptor

var file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDesc = []byte{
	0x0a, 0x3c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x17,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x73, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x1a, 0x3d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2f, 0x6f, 0x61,
	0x75, 0x74, 0x68, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xb0, 0x04, 0x0a, 0x09, 0x50, 0x72, 0x6f, 0x62, 0x65,
	0x43, 0x6f, 0x6e, 0x66, 0x12, 0x3c, 0x0a, 0x0c, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x5f, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x2e, 0x43,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x0b, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x12, 0x4e, 0x0a, 0x0b, 0x61, 0x6c, 0x74, 0x73, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70,
	0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x67, 0x72, 0x70,
	0x63, 0x2e, 0x50, 0x72, 0x6f, 0x62, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x41, 0x4c, 0x54, 0x53,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x0a, 0x61, 0x6c, 0x74, 0x73, 0x43, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x12, 0x4b, 0x0a, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x2d, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x73, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x50, 0x72, 0x6f,
	0x62, 0x65, 0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x54, 0x79, 0x70,
	0x65, 0x3a, 0x04, 0x45, 0x43, 0x48, 0x4f, 0x52, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12,
	0x21, 0x0a, 0x09, 0x62, 0x6c, 0x6f, 0x62, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x05, 0x3a, 0x04, 0x31, 0x30, 0x32, 0x34, 0x52, 0x08, 0x62, 0x6c, 0x6f, 0x62, 0x53, 0x69,
	0x7a, 0x65, 0x12, 0x1e, 0x0a, 0x09, 0x6e, 0x75, 0x6d, 0x5f, 0x63, 0x6f, 0x6e, 0x6e, 0x73, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x05, 0x3a, 0x01, 0x32, 0x52, 0x08, 0x6e, 0x75, 0x6d, 0x43, 0x6f, 0x6e,
	0x6e, 0x73, 0x12, 0x23, 0x0a, 0x0a, 0x6b, 0x65, 0x65, 0x70, 0x5f, 0x61, 0x6c, 0x69, 0x76, 0x65,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x3a, 0x04, 0x74, 0x72, 0x75, 0x65, 0x52, 0x09, 0x6b, 0x65,
	0x65, 0x70, 0x41, 0x6c, 0x69, 0x76, 0x65, 0x12, 0x30, 0x0a, 0x14, 0x63, 0x6f, 0x6e, 0x6e, 0x65,
	0x63, 0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x5f, 0x6d, 0x73, 0x65, 0x63, 0x18,
	0x07, 0x20, 0x01, 0x28, 0x05, 0x52, 0x12, 0x63, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x54, 0x69,
	0x6d, 0x65, 0x6f, 0x75, 0x74, 0x4d, 0x73, 0x65, 0x63, 0x1a, 0x80, 0x01, 0x0a, 0x0a, 0x41, 0x4c,
	0x54, 0x53, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x34, 0x0a, 0x16, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x61, 0x63, 0x63, 0x6f, 0x75,
	0x6e, 0x74, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x14, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x3c,
	0x0a, 0x1a, 0x68, 0x61, 0x6e, 0x64, 0x73, 0x68, 0x61, 0x6b, 0x65, 0x72, 0x5f, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x18, 0x68, 0x61, 0x6e, 0x64, 0x73, 0x68, 0x61, 0x6b, 0x65, 0x72, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0x2b, 0x0a, 0x0a,
	0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x54, 0x79, 0x70, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x45, 0x43,
	0x48, 0x4f, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x52, 0x45, 0x41, 0x44, 0x10, 0x02, 0x12, 0x09,
	0x0a, 0x05, 0x57, 0x52, 0x49, 0x54, 0x45, 0x10, 0x03, 0x42, 0x31, 0x5a, 0x2f, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x63,
	0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x62, 0x65,
	0x73, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescOnce sync.Once
	file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescData = file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDesc
)

func file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescGZIP() []byte {
	file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescOnce.Do(func() {
		file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescData)
	})
	return file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDescData
}

var file_github_com_google_cloudprober_probes_grpc_proto_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_github_com_google_cloudprober_probes_grpc_proto_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_google_cloudprober_probes_grpc_proto_config_proto_goTypes = []interface{}{
	(ProbeConf_MethodType)(0),    // 0: cloudprober.probes.grpc.ProbeConf.MethodType
	(*ProbeConf)(nil),            // 1: cloudprober.probes.grpc.ProbeConf
	(*ProbeConf_ALTSConfig)(nil), // 2: cloudprober.probes.grpc.ProbeConf.ALTSConfig
	(*proto1.Config)(nil),        // 3: cloudprober.oauth.Config
}
var file_github_com_google_cloudprober_probes_grpc_proto_config_proto_depIdxs = []int32{
	3, // 0: cloudprober.probes.grpc.ProbeConf.oauth_config:type_name -> cloudprober.oauth.Config
	2, // 1: cloudprober.probes.grpc.ProbeConf.alts_config:type_name -> cloudprober.probes.grpc.ProbeConf.ALTSConfig
	0, // 2: cloudprober.probes.grpc.ProbeConf.method:type_name -> cloudprober.probes.grpc.ProbeConf.MethodType
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_github_com_google_cloudprober_probes_grpc_proto_config_proto_init() }
func file_github_com_google_cloudprober_probes_grpc_proto_config_proto_init() {
	if File_github_com_google_cloudprober_probes_grpc_proto_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_google_cloudprober_probes_grpc_proto_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProbeConf); i {
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
		file_github_com_google_cloudprober_probes_grpc_proto_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProbeConf_ALTSConfig); i {
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
			RawDescriptor: file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_google_cloudprober_probes_grpc_proto_config_proto_goTypes,
		DependencyIndexes: file_github_com_google_cloudprober_probes_grpc_proto_config_proto_depIdxs,
		EnumInfos:         file_github_com_google_cloudprober_probes_grpc_proto_config_proto_enumTypes,
		MessageInfos:      file_github_com_google_cloudprober_probes_grpc_proto_config_proto_msgTypes,
	}.Build()
	File_github_com_google_cloudprober_probes_grpc_proto_config_proto = out.File
	file_github_com_google_cloudprober_probes_grpc_proto_config_proto_rawDesc = nil
	file_github_com_google_cloudprober_probes_grpc_proto_config_proto_goTypes = nil
	file_github_com_google_cloudprober_probes_grpc_proto_config_proto_depIdxs = nil
}
