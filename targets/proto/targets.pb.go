// Provides all configuration necessary to list targets for a cloudprober probe.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.11.2
// source: github.com/google/cloudprober/targets/proto/targets.proto

package proto

import (
	proto "github.com/google/cloudprober/rds/client/proto"
	proto1 "github.com/google/cloudprober/rds/proto"
	proto3 "github.com/google/cloudprober/targets/file/proto"
	proto2 "github.com/google/cloudprober/targets/gce/proto"
	proto4 "github.com/google/cloudprober/targets/lameduck/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
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

type RDSTargets struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// RDS server options, for example:
	// rds_server_options {
	//   server_address: "rds-server.xyz:9314"
	//   oauth_config: {
	//     ...
	//   }
	// }
	RdsServerOptions *proto.ClientConf_ServerOptions `protobuf:"bytes,1,opt,name=rds_server_options,json=rdsServerOptions" json:"rds_server_options,omitempty"`
	// Resource path specifies the resources to return. Resources paths have the
	// following format:
	// <resource_provider>://<resource_type>/<additional_params>
	//
	// Examples:
	// For GCE instances in projectA: "gcp://gce_instances/<projectA>"
	// Kubernetes Pods : "k8s://pods"
	ResourcePath *string `protobuf:"bytes,2,opt,name=resource_path,json=resourcePath" json:"resource_path,omitempty"`
	// Filters to filter resources by.
	Filter []*proto1.Filter `protobuf:"bytes,3,rep,name=filter" json:"filter,omitempty"`
	// IP config to specify the IP address to pick for a resource.
	IpConfig *proto1.IPConfig `protobuf:"bytes,4,opt,name=ip_config,json=ipConfig" json:"ip_config,omitempty"`
}

func (x *RDSTargets) Reset() {
	*x = RDSTargets{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSTargets) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSTargets) ProtoMessage() {}

func (x *RDSTargets) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSTargets.ProtoReflect.Descriptor instead.
func (*RDSTargets) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescGZIP(), []int{0}
}

func (x *RDSTargets) GetRdsServerOptions() *proto.ClientConf_ServerOptions {
	if x != nil {
		return x.RdsServerOptions
	}
	return nil
}

func (x *RDSTargets) GetResourcePath() string {
	if x != nil && x.ResourcePath != nil {
		return *x.ResourcePath
	}
	return ""
}

func (x *RDSTargets) GetFilter() []*proto1.Filter {
	if x != nil {
		return x.Filter
	}
	return nil
}

func (x *RDSTargets) GetIpConfig() *proto1.IPConfig {
	if x != nil {
		return x.IpConfig
	}
	return nil
}

type TargetsDef struct {
	state           protoimpl.MessageState
	sizeCache       protoimpl.SizeCache
	unknownFields   protoimpl.UnknownFields
	extensionFields protoimpl.ExtensionFields

	// Types that are assignable to Type:
	//	*TargetsDef_HostNames
	//	*TargetsDef_SharedTargets
	//	*TargetsDef_GceTargets
	//	*TargetsDef_RdsTargets
	//	*TargetsDef_FileTargets
	//	*TargetsDef_DummyTargets
	Type isTargetsDef_Type `protobuf_oneof:"type"`
	// Regex to apply on the targets.
	Regex *string `protobuf:"bytes,21,opt,name=regex" json:"regex,omitempty"`
	// Exclude lameducks. Lameduck targets can be set through RTC (realtime
	// configurator) service. This functionality works only if lame_duck_options
	// are specified.
	ExcludeLameducks *bool `protobuf:"varint,22,opt,name=exclude_lameducks,json=excludeLameducks,def=1" json:"exclude_lameducks,omitempty"`
}

// Default values for TargetsDef fields.
const (
	Default_TargetsDef_ExcludeLameducks = bool(true)
)

func (x *TargetsDef) Reset() {
	*x = TargetsDef{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TargetsDef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TargetsDef) ProtoMessage() {}

func (x *TargetsDef) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TargetsDef.ProtoReflect.Descriptor instead.
func (*TargetsDef) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescGZIP(), []int{1}
}

var extRange_TargetsDef = []protoiface.ExtensionRangeV1{
	{Start: 200, End: 536870911},
}

// Deprecated: Use TargetsDef.ProtoReflect.Descriptor.ExtensionRanges instead.
func (*TargetsDef) ExtensionRangeArray() []protoiface.ExtensionRangeV1 {
	return extRange_TargetsDef
}

func (m *TargetsDef) GetType() isTargetsDef_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *TargetsDef) GetHostNames() string {
	if x, ok := x.GetType().(*TargetsDef_HostNames); ok {
		return x.HostNames
	}
	return ""
}

func (x *TargetsDef) GetSharedTargets() string {
	if x, ok := x.GetType().(*TargetsDef_SharedTargets); ok {
		return x.SharedTargets
	}
	return ""
}

func (x *TargetsDef) GetGceTargets() *proto2.TargetsConf {
	if x, ok := x.GetType().(*TargetsDef_GceTargets); ok {
		return x.GceTargets
	}
	return nil
}

func (x *TargetsDef) GetRdsTargets() *RDSTargets {
	if x, ok := x.GetType().(*TargetsDef_RdsTargets); ok {
		return x.RdsTargets
	}
	return nil
}

func (x *TargetsDef) GetFileTargets() *proto3.TargetsConf {
	if x, ok := x.GetType().(*TargetsDef_FileTargets); ok {
		return x.FileTargets
	}
	return nil
}

func (x *TargetsDef) GetDummyTargets() *DummyTargets {
	if x, ok := x.GetType().(*TargetsDef_DummyTargets); ok {
		return x.DummyTargets
	}
	return nil
}

func (x *TargetsDef) GetRegex() string {
	if x != nil && x.Regex != nil {
		return *x.Regex
	}
	return ""
}

func (x *TargetsDef) GetExcludeLameducks() bool {
	if x != nil && x.ExcludeLameducks != nil {
		return *x.ExcludeLameducks
	}
	return Default_TargetsDef_ExcludeLameducks
}

type isTargetsDef_Type interface {
	isTargetsDef_Type()
}

type TargetsDef_HostNames struct {
	// Static host names, for example:
	// host_name: "www.google.com,8.8.8.8,en.wikipedia.org"
	HostNames string `protobuf:"bytes,1,opt,name=host_names,json=hostNames,oneof"`
}

type TargetsDef_SharedTargets struct {
	// Shared targets are accessed through their names.
	// Example:
	// shared_targets {
	//   name:"backend-vms"
	//   targets {
	//     rds_targets {
	//       ..
	//     }
	//   }
	// }
	//
	// probe {
	//   targets {
	//     shared_targets: "backend-vms"
	//   }
	// }
	SharedTargets string `protobuf:"bytes,5,opt,name=shared_targets,json=sharedTargets,oneof"`
}

type TargetsDef_GceTargets struct {
	// GCE targets: instances and forwarding_rules, for example:
	// gce_targets {
	//   instances {}
	// }
	GceTargets *proto2.TargetsConf `protobuf:"bytes,2,opt,name=gce_targets,json=gceTargets,oneof"`
}

type TargetsDef_RdsTargets struct {
	// ResourceDiscovery service based targets.
	// Example:
	// rds_targets {
	//   resource_path: "gcp://gce_instances/{{.project}}"
	//   filter {
	//     key: "name"
	//     value: ".*backend.*"
	//   }
	// }
	RdsTargets *RDSTargets `protobuf:"bytes,3,opt,name=rds_targets,json=rdsTargets,oneof"`
}

type TargetsDef_FileTargets struct {
	// File based targets.
	// Example:
	// file_targets {
	//   file_path: "/var/run/cloudprober/vips.textpb"
	// }
	FileTargets *proto3.TargetsConf `protobuf:"bytes,4,opt,name=file_targets,json=fileTargets,oneof"`
}

type TargetsDef_DummyTargets struct {
	// Empty targets to meet the probe definition requirement where there are
	// actually no targets, for example in case of some external probes.
	DummyTargets *DummyTargets `protobuf:"bytes,20,opt,name=dummy_targets,json=dummyTargets,oneof"`
}

func (*TargetsDef_HostNames) isTargetsDef_Type() {}

func (*TargetsDef_SharedTargets) isTargetsDef_Type() {}

func (*TargetsDef_GceTargets) isTargetsDef_Type() {}

func (*TargetsDef_RdsTargets) isTargetsDef_Type() {}

func (*TargetsDef_FileTargets) isTargetsDef_Type() {}

func (*TargetsDef_DummyTargets) isTargetsDef_Type() {}

// DummyTargets represent empty targets, which are useful for external
// probes that do not have any "proper" targets.  Such as ilbprober.
type DummyTargets struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *DummyTargets) Reset() {
	*x = DummyTargets{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DummyTargets) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DummyTargets) ProtoMessage() {}

func (x *DummyTargets) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DummyTargets.ProtoReflect.Descriptor instead.
func (*DummyTargets) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescGZIP(), []int{2}
}

// Global targets options. These options are independent of the per-probe
// targets which are defined by the "Targets" type above.
//
// Currently these options are used only for GCE targets to control things like
// how often to re-evaluate the targets and whether to check for lame ducks or
// not.
type GlobalTargetsOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// RDS server address
	// Deprecated: This option is now deprecated, please use rds_server_options
	// instead.
	//
	// Deprecated: Do not use.
	RdsServerAddress *string `protobuf:"bytes,3,opt,name=rds_server_address,json=rdsServerAddress" json:"rds_server_address,omitempty"`
	// RDS server options, for example:
	// rds_server_options {
	//   server_address: "rds-server.xyz:9314"
	//   oauth_config: {
	//     ...
	//   }
	// }
	RdsServerOptions *proto.ClientConf_ServerOptions `protobuf:"bytes,4,opt,name=rds_server_options,json=rdsServerOptions" json:"rds_server_options,omitempty"`
	// GCE targets options.
	GlobalGceTargetsOptions *proto2.GlobalOptions `protobuf:"bytes,1,opt,name=global_gce_targets_options,json=globalGceTargetsOptions" json:"global_gce_targets_options,omitempty"`
	// Lame duck options. If provided, targets module checks for the lame duck
	// targets and removes them from the targets list.
	LameDuckOptions *proto4.Options `protobuf:"bytes,2,opt,name=lame_duck_options,json=lameDuckOptions" json:"lame_duck_options,omitempty"`
}

func (x *GlobalTargetsOptions) Reset() {
	*x = GlobalTargetsOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GlobalTargetsOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GlobalTargetsOptions) ProtoMessage() {}

func (x *GlobalTargetsOptions) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GlobalTargetsOptions.ProtoReflect.Descriptor instead.
func (*GlobalTargetsOptions) Descriptor() ([]byte, []int) {
	return file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescGZIP(), []int{3}
}

// Deprecated: Do not use.
func (x *GlobalTargetsOptions) GetRdsServerAddress() string {
	if x != nil && x.RdsServerAddress != nil {
		return *x.RdsServerAddress
	}
	return ""
}

func (x *GlobalTargetsOptions) GetRdsServerOptions() *proto.ClientConf_ServerOptions {
	if x != nil {
		return x.RdsServerOptions
	}
	return nil
}

func (x *GlobalTargetsOptions) GetGlobalGceTargetsOptions() *proto2.GlobalOptions {
	if x != nil {
		return x.GlobalGceTargetsOptions
	}
	return nil
}

func (x *GlobalTargetsOptions) GetLameDuckOptions() *proto4.Options {
	if x != nil {
		return x.LameDuckOptions
	}
	return nil
}

var File_github_com_google_cloudprober_targets_proto_targets_proto protoreflect.FileDescriptor

var file_github_com_google_cloudprober_targets_proto_targets_proto_rawDesc = []byte{
	0x0a, 0x39, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x13, 0x63, 0x6c, 0x6f,
	0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73,
	0x1a, 0x3b, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x72, 0x64, 0x73, 0x2f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x31, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x72, 0x64, 0x73,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x72, 0x64, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x3d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f,
	0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x3c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2f, 0x67, 0x63, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x41, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x73, 0x2f, 0x6c, 0x61, 0x6d, 0x65, 0x64, 0x75, 0x63, 0x6b, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0xf3, 0x01, 0x0a, 0x0a, 0x52, 0x44, 0x53, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12,
	0x57, 0x0a, 0x12, 0x72, 0x64, 0x73, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x6f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x43, 0x6c,
	0x69, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x10, 0x72, 0x64, 0x73, 0x53, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x23, 0x0a, 0x0d, 0x72, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0c, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50, 0x61, 0x74, 0x68, 0x12, 0x2f, 0x0a,
	0x06, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x17, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e,
	0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x52, 0x06, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x12, 0x36,
	0x0a, 0x09, 0x69, 0x70, 0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x19, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e,
	0x72, 0x64, 0x73, 0x2e, 0x49, 0x50, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x52, 0x08, 0x69, 0x70,
	0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22, 0xd5, 0x03, 0x0a, 0x0a, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x44, 0x65, 0x66, 0x12, 0x1f, 0x0a, 0x0a, 0x68, 0x6f, 0x73, 0x74, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x09, 0x68, 0x6f, 0x73,
	0x74, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x12, 0x27, 0x0a, 0x0e, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64,
	0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00,
	0x52, 0x0d, 0x73, 0x68, 0x61, 0x72, 0x65, 0x64, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12,
	0x47, 0x0a, 0x0b, 0x67, 0x63, 0x65, 0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62,
	0x65, 0x72, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x67, 0x63, 0x65, 0x2e, 0x54,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x0a, 0x67, 0x63,
	0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12, 0x42, 0x0a, 0x0b, 0x72, 0x64, 0x73, 0x5f,
	0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x2e, 0x52, 0x44, 0x53, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x48, 0x00,
	0x52, 0x0a, 0x72, 0x64, 0x73, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12, 0x4a, 0x0a, 0x0c,
	0x66, 0x69, 0x6c, 0x65, 0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x25, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x54, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x73, 0x43, 0x6f, 0x6e, 0x66, 0x48, 0x00, 0x52, 0x0b, 0x66, 0x69, 0x6c,
	0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12, 0x48, 0x0a, 0x0d, 0x64, 0x75, 0x6d, 0x6d,
	0x79, 0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x14, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x21, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x44, 0x75, 0x6d, 0x6d, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x48, 0x00, 0x52, 0x0c, 0x64, 0x75, 0x6d, 0x6d, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x72, 0x65, 0x67, 0x65, 0x78, 0x18, 0x15, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x72, 0x65, 0x67, 0x65, 0x78, 0x12, 0x31, 0x0a, 0x11, 0x65, 0x78, 0x63, 0x6c,
	0x75, 0x64, 0x65, 0x5f, 0x6c, 0x61, 0x6d, 0x65, 0x64, 0x75, 0x63, 0x6b, 0x73, 0x18, 0x16, 0x20,
	0x01, 0x28, 0x08, 0x3a, 0x04, 0x74, 0x72, 0x75, 0x65, 0x52, 0x10, 0x65, 0x78, 0x63, 0x6c, 0x75,
	0x64, 0x65, 0x4c, 0x61, 0x6d, 0x65, 0x64, 0x75, 0x63, 0x6b, 0x73, 0x2a, 0x09, 0x08, 0xc8, 0x01,
	0x10, 0x80, 0x80, 0x80, 0x80, 0x02, 0x42, 0x06, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x0e,
	0x0a, 0x0c, 0x44, 0x75, 0x6d, 0x6d, 0x79, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x22, 0xd9,
	0x02, 0x0a, 0x14, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x30, 0x0a, 0x12, 0x72, 0x64, 0x73, 0x5f, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x42, 0x02, 0x18, 0x01, 0x52, 0x10, 0x72, 0x64, 0x73, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x57, 0x0a, 0x12, 0x72, 0x64, 0x73,
	0x5f, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f,
	0x62, 0x65, 0x72, 0x2e, 0x72, 0x64, 0x73, 0x2e, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x43, 0x6f,
	0x6e, 0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x52, 0x10, 0x72, 0x64, 0x73, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x4f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x12, 0x63, 0x0a, 0x1a, 0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x5f, 0x67, 0x63, 0x65,
	0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72,
	0x6f, 0x62, 0x65, 0x72, 0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x67, 0x63, 0x65,
	0x2e, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x17,
	0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x47, 0x63, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x51, 0x0a, 0x11, 0x6c, 0x61, 0x6d, 0x65, 0x5f,
	0x64, 0x75, 0x63, 0x6b, 0x5f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x25, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72,
	0x2e, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x2e, 0x6c, 0x61, 0x6d, 0x65, 0x64, 0x75, 0x63,
	0x6b, 0x2e, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x0f, 0x6c, 0x61, 0x6d, 0x65, 0x44,
	0x75, 0x63, 0x6b, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x42, 0x2d, 0x5a, 0x2b, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x70, 0x72, 0x6f, 0x62, 0x65, 0x72, 0x2f, 0x74, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
}

var (
	file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescOnce sync.Once
	file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescData = file_github_com_google_cloudprober_targets_proto_targets_proto_rawDesc
)

func file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescGZIP() []byte {
	file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescOnce.Do(func() {
		file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescData)
	})
	return file_github_com_google_cloudprober_targets_proto_targets_proto_rawDescData
}

var file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_github_com_google_cloudprober_targets_proto_targets_proto_goTypes = []interface{}{
	(*RDSTargets)(nil),                     // 0: cloudprober.targets.RDSTargets
	(*TargetsDef)(nil),                     // 1: cloudprober.targets.TargetsDef
	(*DummyTargets)(nil),                   // 2: cloudprober.targets.DummyTargets
	(*GlobalTargetsOptions)(nil),           // 3: cloudprober.targets.GlobalTargetsOptions
	(*proto.ClientConf_ServerOptions)(nil), // 4: cloudprober.rds.ClientConf.ServerOptions
	(*proto1.Filter)(nil),                  // 5: cloudprober.rds.Filter
	(*proto1.IPConfig)(nil),                // 6: cloudprober.rds.IPConfig
	(*proto2.TargetsConf)(nil),             // 7: cloudprober.targets.gce.TargetsConf
	(*proto3.TargetsConf)(nil),             // 8: cloudprober.targets.file.TargetsConf
	(*proto2.GlobalOptions)(nil),           // 9: cloudprober.targets.gce.GlobalOptions
	(*proto4.Options)(nil),                 // 10: cloudprober.targets.lameduck.Options
}
var file_github_com_google_cloudprober_targets_proto_targets_proto_depIdxs = []int32{
	4,  // 0: cloudprober.targets.RDSTargets.rds_server_options:type_name -> cloudprober.rds.ClientConf.ServerOptions
	5,  // 1: cloudprober.targets.RDSTargets.filter:type_name -> cloudprober.rds.Filter
	6,  // 2: cloudprober.targets.RDSTargets.ip_config:type_name -> cloudprober.rds.IPConfig
	7,  // 3: cloudprober.targets.TargetsDef.gce_targets:type_name -> cloudprober.targets.gce.TargetsConf
	0,  // 4: cloudprober.targets.TargetsDef.rds_targets:type_name -> cloudprober.targets.RDSTargets
	8,  // 5: cloudprober.targets.TargetsDef.file_targets:type_name -> cloudprober.targets.file.TargetsConf
	2,  // 6: cloudprober.targets.TargetsDef.dummy_targets:type_name -> cloudprober.targets.DummyTargets
	4,  // 7: cloudprober.targets.GlobalTargetsOptions.rds_server_options:type_name -> cloudprober.rds.ClientConf.ServerOptions
	9,  // 8: cloudprober.targets.GlobalTargetsOptions.global_gce_targets_options:type_name -> cloudprober.targets.gce.GlobalOptions
	10, // 9: cloudprober.targets.GlobalTargetsOptions.lame_duck_options:type_name -> cloudprober.targets.lameduck.Options
	10, // [10:10] is the sub-list for method output_type
	10, // [10:10] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_github_com_google_cloudprober_targets_proto_targets_proto_init() }
func file_github_com_google_cloudprober_targets_proto_targets_proto_init() {
	if File_github_com_google_cloudprober_targets_proto_targets_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSTargets); i {
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
		file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TargetsDef); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			case 3:
				return &v.extensionFields
			default:
				return nil
			}
		}
		file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DummyTargets); i {
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
		file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GlobalTargetsOptions); i {
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
	file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*TargetsDef_HostNames)(nil),
		(*TargetsDef_SharedTargets)(nil),
		(*TargetsDef_GceTargets)(nil),
		(*TargetsDef_RdsTargets)(nil),
		(*TargetsDef_FileTargets)(nil),
		(*TargetsDef_DummyTargets)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_google_cloudprober_targets_proto_targets_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_google_cloudprober_targets_proto_targets_proto_goTypes,
		DependencyIndexes: file_github_com_google_cloudprober_targets_proto_targets_proto_depIdxs,
		MessageInfos:      file_github_com_google_cloudprober_targets_proto_targets_proto_msgTypes,
	}.Build()
	File_github_com_google_cloudprober_targets_proto_targets_proto = out.File
	file_github_com_google_cloudprober_targets_proto_targets_proto_rawDesc = nil
	file_github_com_google_cloudprober_targets_proto_targets_proto_goTypes = nil
	file_github_com_google_cloudprober_targets_proto_targets_proto_depIdxs = nil
}
