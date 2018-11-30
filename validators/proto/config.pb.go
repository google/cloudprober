// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/validators/proto/config.proto

package proto

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import proto1 "github.com/google/cloudprober/validators/http/proto"
import proto2 "github.com/google/cloudprober/validators/integrity/proto"
import proto3 "github.com/google/cloudprober/validators/regex/proto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Validator struct {
	Name *string `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	// Types that are valid to be assigned to Type:
	//	*Validator_HttpValidator
	//	*Validator_IntegrityValidator
	//	*Validator_RegexValidator
	Type                 isValidator_Type `protobuf_oneof:"type"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *Validator) Reset()         { *m = Validator{} }
func (m *Validator) String() string { return proto.CompactTextString(m) }
func (*Validator) ProtoMessage()    {}
func (*Validator) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_3c16e264a9886a15, []int{0}
}
func (m *Validator) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Validator.Unmarshal(m, b)
}
func (m *Validator) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Validator.Marshal(b, m, deterministic)
}
func (dst *Validator) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Validator.Merge(dst, src)
}
func (m *Validator) XXX_Size() int {
	return xxx_messageInfo_Validator.Size(m)
}
func (m *Validator) XXX_DiscardUnknown() {
	xxx_messageInfo_Validator.DiscardUnknown(m)
}

var xxx_messageInfo_Validator proto.InternalMessageInfo

func (m *Validator) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

type isValidator_Type interface {
	isValidator_Type()
}

type Validator_HttpValidator struct {
	HttpValidator *proto1.Validator `protobuf:"bytes,2,opt,name=http_validator,json=httpValidator,oneof"`
}

type Validator_IntegrityValidator struct {
	IntegrityValidator *proto2.Validator `protobuf:"bytes,3,opt,name=integrity_validator,json=integrityValidator,oneof"`
}

type Validator_RegexValidator struct {
	RegexValidator *proto3.Validator `protobuf:"bytes,4,opt,name=regex_validator,json=regexValidator,oneof"`
}

func (*Validator_HttpValidator) isValidator_Type() {}

func (*Validator_IntegrityValidator) isValidator_Type() {}

func (*Validator_RegexValidator) isValidator_Type() {}

func (m *Validator) GetType() isValidator_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *Validator) GetHttpValidator() *proto1.Validator {
	if x, ok := m.GetType().(*Validator_HttpValidator); ok {
		return x.HttpValidator
	}
	return nil
}

func (m *Validator) GetIntegrityValidator() *proto2.Validator {
	if x, ok := m.GetType().(*Validator_IntegrityValidator); ok {
		return x.IntegrityValidator
	}
	return nil
}

func (m *Validator) GetRegexValidator() *proto3.Validator {
	if x, ok := m.GetType().(*Validator_RegexValidator); ok {
		return x.RegexValidator
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*Validator) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _Validator_OneofMarshaler, _Validator_OneofUnmarshaler, _Validator_OneofSizer, []interface{}{
		(*Validator_HttpValidator)(nil),
		(*Validator_IntegrityValidator)(nil),
		(*Validator_RegexValidator)(nil),
	}
}

func _Validator_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*Validator)
	// type
	switch x := m.Type.(type) {
	case *Validator_HttpValidator:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.HttpValidator); err != nil {
			return err
		}
	case *Validator_IntegrityValidator:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.IntegrityValidator); err != nil {
			return err
		}
	case *Validator_RegexValidator:
		b.EncodeVarint(4<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.RegexValidator); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("Validator.Type has unexpected type %T", x)
	}
	return nil
}

func _Validator_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*Validator)
	switch tag {
	case 2: // type.http_validator
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(proto1.Validator)
		err := b.DecodeMessage(msg)
		m.Type = &Validator_HttpValidator{msg}
		return true, err
	case 3: // type.integrity_validator
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(proto2.Validator)
		err := b.DecodeMessage(msg)
		m.Type = &Validator_IntegrityValidator{msg}
		return true, err
	case 4: // type.regex_validator
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(proto3.Validator)
		err := b.DecodeMessage(msg)
		m.Type = &Validator_RegexValidator{msg}
		return true, err
	default:
		return false, nil
	}
}

func _Validator_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*Validator)
	// type
	switch x := m.Type.(type) {
	case *Validator_HttpValidator:
		s := proto.Size(x.HttpValidator)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *Validator_IntegrityValidator:
		s := proto.Size(x.IntegrityValidator)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *Validator_RegexValidator:
		s := proto.Size(x.RegexValidator)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

func init() {
	proto.RegisterType((*Validator)(nil), "cloudprober.validators.Validator")
}

func init() {
	proto.RegisterFile("github.com/google/cloudprober/validators/proto/config.proto", fileDescriptor_config_3c16e264a9886a15)
}

var fileDescriptor_config_3c16e264a9886a15 = []byte{
	// 240 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0xd0, 0xcf, 0x4b, 0x86, 0x30,
	0x1c, 0x06, 0xf0, 0x5e, 0x93, 0xe0, 0x5d, 0xf4, 0x06, 0x0b, 0x42, 0x3c, 0x49, 0x87, 0x12, 0x82,
	0x0d, 0xba, 0x76, 0xa9, 0x20, 0xe8, 0x16, 0x78, 0xe8, 0x58, 0xf8, 0x63, 0xcd, 0x81, 0xfa, 0x1d,
	0xeb, 0x6b, 0xe4, 0xff, 0xd4, 0x1f, 0x19, 0x0e, 0x9a, 0x0a, 0x13, 0xbc, 0xe9, 0xc3, 0xb3, 0xcf,
	0x78, 0x46, 0xee, 0xa5, 0xc2, 0xba, 0x2f, 0x58, 0x09, 0x2d, 0x97, 0x00, 0xb2, 0x11, 0xbc, 0x6c,
	0xa0, 0xaf, 0xb4, 0x81, 0x42, 0x18, 0xfe, 0x9d, 0x37, 0xaa, 0xca, 0x11, 0xcc, 0x17, 0xd7, 0x06,
	0x10, 0x78, 0x09, 0xdd, 0xa7, 0x92, 0xcc, 0xfe, 0xd0, 0xcb, 0x59, 0x95, 0x4d, 0xd5, 0xf8, 0x61,
	0x33, 0x5a, 0x23, 0x6a, 0x8f, 0x1c, 0x3f, 0x6f, 0x16, 0x54, 0x87, 0x42, 0x1a, 0x85, 0x83, 0x8f,
	0x79, 0xdc, 0xcc, 0x18, 0x21, 0xc5, 0x8f, 0x87, 0xb8, 0xfa, 0x0d, 0xc8, 0xfe, 0xed, 0xbf, 0x47,
	0x29, 0x09, 0xbb, 0xbc, 0x15, 0xd1, 0x2e, 0x09, 0xd2, 0x7d, 0x66, 0xbf, 0xe9, 0x2b, 0x39, 0x8c,
	0x33, 0x3e, 0x9c, 0x16, 0x05, 0xc9, 0x2e, 0x3d, 0xbd, 0xbb, 0x66, 0xfe, 0xe7, 0x61, 0x63, 0x9b,
	0x39, 0xf3, 0xe5, 0x28, 0x3b, 0x1b, 0x93, 0xe9, 0x92, 0x77, 0x72, 0xe1, 0x56, 0xcd, 0xd4, 0x63,
	0xab, 0xde, 0xae, 0xa9, 0xee, 0xc8, 0x82, 0xa6, 0x2e, 0x9e, 0xfc, 0x8c, 0x9c, 0xdb, 0xb9, 0x33,
	0x3b, 0xb4, 0xf6, 0xcd, 0x9a, 0x6d, 0xeb, 0x0b, 0xf7, 0x60, 0x23, 0x97, 0x3c, 0x9d, 0x90, 0x10,
	0x07, 0x2d, 0xfe, 0x02, 0x00, 0x00, 0xff, 0xff, 0x18, 0x5b, 0xba, 0x59, 0x50, 0x02, 0x00, 0x00,
}
