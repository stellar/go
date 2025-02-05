// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.3
// 	protoc        v5.29.3
// source: address/address.proto

package address

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

// Address message with oneof for different address types
type Address struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to AddressType:
	//
	//	*Address_SmartContractAddress
	//	*Address_AccountAddress
	//	*Address_LiquidityPoolHash
	//	*Address_ClaimableBalanceId
	AddressType   isAddress_AddressType `protobuf_oneof:"address_type"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Address) Reset() {
	*x = Address{}
	mi := &file_address_address_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Address) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Address) ProtoMessage() {}

func (x *Address) ProtoReflect() protoreflect.Message {
	mi := &file_address_address_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Address.ProtoReflect.Descriptor instead.
func (*Address) Descriptor() ([]byte, []int) {
	return file_address_address_proto_rawDescGZIP(), []int{0}
}

func (x *Address) GetAddressType() isAddress_AddressType {
	if x != nil {
		return x.AddressType
	}
	return nil
}

func (x *Address) GetSmartContractAddress() string {
	if x != nil {
		if x, ok := x.AddressType.(*Address_SmartContractAddress); ok {
			return x.SmartContractAddress
		}
	}
	return ""
}

func (x *Address) GetAccountAddress() string {
	if x != nil {
		if x, ok := x.AddressType.(*Address_AccountAddress); ok {
			return x.AccountAddress
		}
	}
	return ""
}

func (x *Address) GetLiquidityPoolHash() string {
	if x != nil {
		if x, ok := x.AddressType.(*Address_LiquidityPoolHash); ok {
			return x.LiquidityPoolHash
		}
	}
	return ""
}

func (x *Address) GetClaimableBalanceId() string {
	if x != nil {
		if x, ok := x.AddressType.(*Address_ClaimableBalanceId); ok {
			return x.ClaimableBalanceId
		}
	}
	return ""
}

type isAddress_AddressType interface {
	isAddress_AddressType()
}

type Address_SmartContractAddress struct {
	SmartContractAddress string `protobuf:"bytes,1,opt,name=smartContractAddress,proto3,oneof"` // Smart Contract address
}

type Address_AccountAddress struct {
	AccountAddress string `protobuf:"bytes,2,opt,name=accountAddress,proto3,oneof"` // Account address
}

type Address_LiquidityPoolHash struct {
	LiquidityPoolHash string `protobuf:"bytes,3,opt,name=liquidityPoolHash,proto3,oneof"` // Liquidity Pool hash
}

type Address_ClaimableBalanceId struct {
	ClaimableBalanceId string `protobuf:"bytes,4,opt,name=claimableBalanceId,proto3,oneof"` // Claimable Balance ID
}

func (*Address_SmartContractAddress) isAddress_AddressType() {}

func (*Address_AccountAddress) isAddress_AddressType() {}

func (*Address_LiquidityPoolHash) isAddress_AddressType() {}

func (*Address_ClaimableBalanceId) isAddress_AddressType() {}

var File_address_address_proto protoreflect.FileDescriptor

var file_address_address_proto_rawDesc = []byte{
	0x0a, 0x15, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x2f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73,
	0x22, 0xdb, 0x01, 0x0a, 0x07, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x34, 0x0a, 0x14,
	0x73, 0x6d, 0x61, 0x72, 0x74, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x41, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x14, 0x73, 0x6d,
	0x61, 0x72, 0x74, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x12, 0x28, 0x0a, 0x0e, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x41, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0e, 0x61, 0x63,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x2e, 0x0a, 0x11,
	0x6c, 0x69, 0x71, 0x75, 0x69, 0x64, 0x69, 0x74, 0x79, 0x50, 0x6f, 0x6f, 0x6c, 0x48, 0x61, 0x73,
	0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x11, 0x6c, 0x69, 0x71, 0x75, 0x69,
	0x64, 0x69, 0x74, 0x79, 0x50, 0x6f, 0x6f, 0x6c, 0x48, 0x61, 0x73, 0x68, 0x12, 0x30, 0x0a, 0x12,
	0x63, 0x6c, 0x61, 0x69, 0x6d, 0x61, 0x62, 0x6c, 0x65, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65,
	0x49, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x12, 0x63, 0x6c, 0x61, 0x69,
	0x6d, 0x61, 0x62, 0x6c, 0x65, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x49, 0x64, 0x42, 0x0e,
	0x0a, 0x0c, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_address_address_proto_rawDescOnce sync.Once
	file_address_address_proto_rawDescData = file_address_address_proto_rawDesc
)

func file_address_address_proto_rawDescGZIP() []byte {
	file_address_address_proto_rawDescOnce.Do(func() {
		file_address_address_proto_rawDescData = protoimpl.X.CompressGZIP(file_address_address_proto_rawDescData)
	})
	return file_address_address_proto_rawDescData
}

var file_address_address_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_address_address_proto_goTypes = []any{
	(*Address)(nil), // 0: address.Address
}
var file_address_address_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_address_address_proto_init() }
func file_address_address_proto_init() {
	if File_address_address_proto != nil {
		return
	}
	file_address_address_proto_msgTypes[0].OneofWrappers = []any{
		(*Address_SmartContractAddress)(nil),
		(*Address_AccountAddress)(nil),
		(*Address_LiquidityPoolHash)(nil),
		(*Address_ClaimableBalanceId)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_address_address_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_address_address_proto_goTypes,
		DependencyIndexes: file_address_address_proto_depIdxs,
		MessageInfos:      file_address_address_proto_msgTypes,
	}.Build()
	File_address_address_proto = out.File
	file_address_address_proto_rawDesc = nil
	file_address_address_proto_goTypes = nil
	file_address_address_proto_depIdxs = nil
}
