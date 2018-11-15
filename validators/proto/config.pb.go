// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/google/cloudprober/validators/proto/config.proto

package proto

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import proto1 "github.com/google/cloudprober/validators/http/proto"

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
	Type                 isValidator_Type `protobuf_oneof:"type"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *Validator) Reset()         { *m = Validator{} }
func (m *Validator) String() string { return proto.CompactTextString(m) }
func (*Validator) ProtoMessage()    {}
func (*Validator) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_7fbb24b4ab62dfdf, []int{0}
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

func (*Validator_HttpValidator) isValidator_Type() {}

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

// XXX_OneofFuncs is for the internal use of the proto package.
func (*Validator) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _Validator_OneofMarshaler, _Validator_OneofUnmarshaler, _Validator_OneofSizer, []interface{}{
		(*Validator_HttpValidator)(nil),
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
	proto.RegisterFile("github.com/google/cloudprober/validators/proto/config.proto", fileDescriptor_config_7fbb24b4ab62dfdf)
}

var fileDescriptor_config_7fbb24b4ab62dfdf = []byte{
	// 166 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xb2, 0x4e, 0xcf, 0x2c, 0xc9,
	0x28, 0x4d, 0xd2, 0x4b, 0xce, 0xcf, 0xd5, 0x4f, 0xcf, 0xcf, 0x4f, 0xcf, 0x49, 0xd5, 0x4f, 0xce,
	0xc9, 0x2f, 0x4d, 0x29, 0x28, 0xca, 0x4f, 0x4a, 0x2d, 0xd2, 0x2f, 0x4b, 0xcc, 0xc9, 0x4c, 0x49,
	0x2c, 0xc9, 0x2f, 0x2a, 0xd6, 0x2f, 0x28, 0xca, 0x2f, 0xc9, 0xd7, 0x4f, 0xce, 0xcf, 0x4b, 0xcb,
	0x4c, 0xd7, 0x03, 0x73, 0x84, 0xc4, 0x90, 0x94, 0xea, 0x21, 0x94, 0x4a, 0x39, 0x10, 0x6d, 0x68,
	0x46, 0x49, 0x49, 0x01, 0x16, 0x93, 0x95, 0x2a, 0xb8, 0x38, 0xc3, 0x60, 0xaa, 0x84, 0x84, 0xb8,
	0x58, 0xf2, 0x12, 0x73, 0x53, 0x25, 0x18, 0x15, 0x98, 0x34, 0x38, 0x83, 0xc0, 0x6c, 0x21, 0x7f,
	0x2e, 0x3e, 0x90, 0xde, 0x78, 0xb8, 0x59, 0x12, 0x4c, 0x0a, 0x8c, 0x1a, 0xdc, 0x46, 0x6a, 0x7a,
	0xd8, 0xdd, 0xa4, 0x07, 0x52, 0xad, 0x07, 0x37, 0xd3, 0x83, 0x21, 0x88, 0x17, 0x24, 0x02, 0x17,
	0x70, 0x62, 0xe3, 0x62, 0x29, 0xa9, 0x2c, 0x48, 0x05, 0x04, 0x00, 0x00, 0xff, 0xff, 0x66, 0x84,
	0x66, 0xfb, 0x11, 0x01, 0x00, 0x00,
}
