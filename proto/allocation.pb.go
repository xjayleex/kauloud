// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.13.0
// source: allocation.proto

package proto

import (
	context "context"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type AllocResponse_Status int32

const (
	AllocResponse_FAILED              AllocResponse_Status = 0
	AllocResponse_PARTIALLY_SUCCEEDED AllocResponse_Status = 1
	AllocResponse_SUCCEEDED           AllocResponse_Status = 2
)

// Enum value maps for AllocResponse_Status.
var (
	AllocResponse_Status_name = map[int32]string{
		0: "FAILED",
		1: "PARTIALLY_SUCCEEDED",
		2: "SUCCEEDED",
	}
	AllocResponse_Status_value = map[string]int32{
		"FAILED":              0,
		"PARTIALLY_SUCCEEDED": 1,
		"SUCCEEDED":           2,
	}
)

func (x AllocResponse_Status) Enum() *AllocResponse_Status {
	p := new(AllocResponse_Status)
	*p = x
	return p
}

func (x AllocResponse_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AllocResponse_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_allocation_proto_enumTypes[0].Descriptor()
}

func (AllocResponse_Status) Type() protoreflect.EnumType {
	return &file_allocation_proto_enumTypes[0]
}

func (x AllocResponse_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AllocResponse_Status.Descriptor instead.
func (AllocResponse_Status) EnumDescriptor() ([]byte, []int) {
	return file_allocation_proto_rawDescGZIP(), []int{1, 0}
}

type Availability_Status int32

const (
	Availability_UN_AVAILABLE        Availability_Status = 0
	Availability_PARTIALLY_AVAILABLE Availability_Status = 1
	Availability_AVAILABLE           Availability_Status = 2
)

// Enum value maps for Availability_Status.
var (
	Availability_Status_name = map[int32]string{
		0: "UN_AVAILABLE",
		1: "PARTIALLY_AVAILABLE",
		2: "AVAILABLE",
	}
	Availability_Status_value = map[string]int32{
		"UN_AVAILABLE":        0,
		"PARTIALLY_AVAILABLE": 1,
		"AVAILABLE":           2,
	}
)

func (x Availability_Status) Enum() *Availability_Status {
	p := new(Availability_Status)
	*p = x
	return p
}

func (x Availability_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Availability_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_allocation_proto_enumTypes[1].Descriptor()
}

func (Availability_Status) Type() protoreflect.EnumType {
	return &file_allocation_proto_enumTypes[1]
}

func (x Availability_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Availability_Status.Descriptor instead.
func (Availability_Status) EnumDescriptor() ([]byte, []int) {
	return file_allocation_proto_rawDescGZIP(), []int{2, 0}
}

type AllocRequestDescription struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// TODO :: Add User Info, it would be needed.
	Descriptions []*ContainerResource `protobuf:"bytes,1,rep,name=descriptions,proto3" json:"descriptions,omitempty"`
}

func (x *AllocRequestDescription) Reset() {
	*x = AllocRequestDescription{}
	if protoimpl.UnsafeEnabled {
		mi := &file_allocation_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AllocRequestDescription) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AllocRequestDescription) ProtoMessage() {}

func (x *AllocRequestDescription) ProtoReflect() protoreflect.Message {
	mi := &file_allocation_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AllocRequestDescription.ProtoReflect.Descriptor instead.
func (*AllocRequestDescription) Descriptor() ([]byte, []int) {
	return file_allocation_proto_rawDescGZIP(), []int{0}
}

func (x *AllocRequestDescription) GetDescriptions() []*ContainerResource {
	if x != nil {
		return x.Descriptions
	}
	return nil
}

type AllocResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Results []*Result `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
}

func (x *AllocResponse) Reset() {
	*x = AllocResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_allocation_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AllocResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AllocResponse) ProtoMessage() {}

func (x *AllocResponse) ProtoReflect() protoreflect.Message {
	mi := &file_allocation_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AllocResponse.ProtoReflect.Descriptor instead.
func (*AllocResponse) Descriptor() ([]byte, []int) {
	return file_allocation_proto_rawDescGZIP(), []int{1}
}

func (x *AllocResponse) GetResults() []*Result {
	if x != nil {
		return x.Results
	}
	return nil
}

type Availability struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Results []*Result `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
}

func (x *Availability) Reset() {
	*x = Availability{}
	if protoimpl.UnsafeEnabled {
		mi := &file_allocation_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Availability) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Availability) ProtoMessage() {}

func (x *Availability) ProtoReflect() protoreflect.Message {
	mi := &file_allocation_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Availability.ProtoReflect.Descriptor instead.
func (*Availability) Descriptor() ([]byte, []int) {
	return file_allocation_proto_rawDescGZIP(), []int{2}
}

func (x *Availability) GetResults() []*Result {
	if x != nil {
		return x.Results
	}
	return nil
}

type ContainerResource struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type                      bool         `protobuf:"varint,1,opt,name=type,proto3" json:"type,omitempty"`
	Cpu                       int32        `protobuf:"varint,2,opt,name=cpu,proto3" json:"cpu,omitempty"`
	MemoryInMb                int32        `protobuf:"varint,3,opt,name=memory_in_mb,json=memoryInMb,proto3" json:"memory_in_mb,omitempty"`
	EphemeralStorageInGb      int32        `protobuf:"varint,4,opt,name=ephemeral_storage_in_gb,json=ephemeralStorageInGb,proto3" json:"ephemeral_storage_in_gb,omitempty"`
	PersistentVolumeClaimCode string       `protobuf:"bytes,5,opt,name=persistent_volume_claim_code,json=persistentVolumeClaimCode,proto3" json:"persistent_volume_claim_code,omitempty"`
	HostAliases               []*HostAlias `protobuf:"bytes,6,rep,name=host_aliases,json=hostAliases,proto3" json:"host_aliases,omitempty"`
	Image                     string       `protobuf:"bytes,7,opt,name=image,proto3" json:"image,omitempty"` //string token;
}

func (x *ContainerResource) Reset() {
	*x = ContainerResource{}
	if protoimpl.UnsafeEnabled {
		mi := &file_allocation_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ContainerResource) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContainerResource) ProtoMessage() {}

func (x *ContainerResource) ProtoReflect() protoreflect.Message {
	mi := &file_allocation_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContainerResource.ProtoReflect.Descriptor instead.
func (*ContainerResource) Descriptor() ([]byte, []int) {
	return file_allocation_proto_rawDescGZIP(), []int{3}
}

func (x *ContainerResource) GetType() bool {
	if x != nil {
		return x.Type
	}
	return false
}

func (x *ContainerResource) GetCpu() int32 {
	if x != nil {
		return x.Cpu
	}
	return 0
}

func (x *ContainerResource) GetMemoryInMb() int32 {
	if x != nil {
		return x.MemoryInMb
	}
	return 0
}

func (x *ContainerResource) GetEphemeralStorageInGb() int32 {
	if x != nil {
		return x.EphemeralStorageInGb
	}
	return 0
}

func (x *ContainerResource) GetPersistentVolumeClaimCode() string {
	if x != nil {
		return x.PersistentVolumeClaimCode
	}
	return ""
}

func (x *ContainerResource) GetHostAliases() []*HostAlias {
	if x != nil {
		return x.HostAliases
	}
	return nil
}

func (x *ContainerResource) GetImage() string {
	if x != nil {
		return x.Image
	}
	return ""
}

type HostAlias struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	IpAddress string   `protobuf:"bytes,1,opt,name=ip_address,json=ipAddress,proto3" json:"ip_address,omitempty"`
	HostNames []string `protobuf:"bytes,2,rep,name=host_names,json=hostNames,proto3" json:"host_names,omitempty"`
}

func (x *HostAlias) Reset() {
	*x = HostAlias{}
	if protoimpl.UnsafeEnabled {
		mi := &file_allocation_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HostAlias) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HostAlias) ProtoMessage() {}

func (x *HostAlias) ProtoReflect() protoreflect.Message {
	mi := &file_allocation_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HostAlias.ProtoReflect.Descriptor instead.
func (*HostAlias) Descriptor() ([]byte, []int) {
	return file_allocation_proto_rawDescGZIP(), []int{4}
}

func (x *HostAlias) GetIpAddress() string {
	if x != nil {
		return x.IpAddress
	}
	return ""
}

func (x *HostAlias) GetHostNames() []string {
	if x != nil {
		return x.HostNames
	}
	return nil
}

type Result struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ok         bool   `protobuf:"varint,1,opt,name=ok,proto3" json:"ok,omitempty"`
	Additional string `protobuf:"bytes,2,opt,name=additional,proto3" json:"additional,omitempty"`
}

func (x *Result) Reset() {
	*x = Result{}
	if protoimpl.UnsafeEnabled {
		mi := &file_allocation_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Result) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Result) ProtoMessage() {}

func (x *Result) ProtoReflect() protoreflect.Message {
	mi := &file_allocation_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Result.ProtoReflect.Descriptor instead.
func (*Result) Descriptor() ([]byte, []int) {
	return file_allocation_proto_rawDescGZIP(), []int{5}
}

func (x *Result) GetOk() bool {
	if x != nil {
		return x.Ok
	}
	return false
}

func (x *Result) GetAdditional() string {
	if x != nil {
		return x.Additional
	}
	return ""
}

var File_allocation_proto protoreflect.FileDescriptor

var file_allocation_proto_rawDesc = []byte{
	0x0a, 0x10, 0x61, 0x6c, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x57, 0x0a, 0x17, 0x41, 0x6c, 0x6c,
	0x6f, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x3c, 0x0a, 0x0c, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x52, 0x65, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x52, 0x0c, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x22, 0x76, 0x0a, 0x0d, 0x41, 0x6c, 0x6c, 0x6f, 0x63, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x27, 0x0a, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x73,
	0x75, 0x6c, 0x74, 0x52, 0x07, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x73, 0x22, 0x3c, 0x0a, 0x06,
	0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x0a, 0x0a, 0x06, 0x46, 0x41, 0x49, 0x4c, 0x45, 0x44,
	0x10, 0x00, 0x12, 0x17, 0x0a, 0x13, 0x50, 0x41, 0x52, 0x54, 0x49, 0x41, 0x4c, 0x4c, 0x59, 0x5f,
	0x53, 0x55, 0x43, 0x43, 0x45, 0x45, 0x44, 0x45, 0x44, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x53,
	0x55, 0x43, 0x43, 0x45, 0x45, 0x44, 0x45, 0x44, 0x10, 0x02, 0x22, 0x7b, 0x0a, 0x0c, 0x41, 0x76,
	0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x12, 0x27, 0x0a, 0x07, 0x72, 0x65,
	0x73, 0x75, 0x6c, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x52, 0x07, 0x72, 0x65, 0x73, 0x75,
	0x6c, 0x74, 0x73, 0x22, 0x42, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x10, 0x0a,
	0x0c, 0x55, 0x4e, 0x5f, 0x41, 0x56, 0x41, 0x49, 0x4c, 0x41, 0x42, 0x4c, 0x45, 0x10, 0x00, 0x12,
	0x17, 0x0a, 0x13, 0x50, 0x41, 0x52, 0x54, 0x49, 0x41, 0x4c, 0x4c, 0x59, 0x5f, 0x41, 0x56, 0x41,
	0x49, 0x4c, 0x41, 0x42, 0x4c, 0x45, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x41, 0x56, 0x41, 0x49,
	0x4c, 0x41, 0x42, 0x4c, 0x45, 0x10, 0x02, 0x22, 0x9e, 0x02, 0x0a, 0x11, 0x43, 0x6f, 0x6e, 0x74,
	0x61, 0x69, 0x6e, 0x65, 0x72, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x70, 0x75, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03,
	0x63, 0x70, 0x75, 0x12, 0x20, 0x0a, 0x0c, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x5f, 0x69, 0x6e,
	0x5f, 0x6d, 0x62, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x6d, 0x65, 0x6d, 0x6f, 0x72,
	0x79, 0x49, 0x6e, 0x4d, 0x62, 0x12, 0x35, 0x0a, 0x17, 0x65, 0x70, 0x68, 0x65, 0x6d, 0x65, 0x72,
	0x61, 0x6c, 0x5f, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x5f, 0x69, 0x6e, 0x5f, 0x67, 0x62,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x05, 0x52, 0x14, 0x65, 0x70, 0x68, 0x65, 0x6d, 0x65, 0x72, 0x61,
	0x6c, 0x53, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x49, 0x6e, 0x47, 0x62, 0x12, 0x3f, 0x0a, 0x1c,
	0x70, 0x65, 0x72, 0x73, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x74, 0x5f, 0x76, 0x6f, 0x6c, 0x75, 0x6d,
	0x65, 0x5f, 0x63, 0x6c, 0x61, 0x69, 0x6d, 0x5f, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x19, 0x70, 0x65, 0x72, 0x73, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x74, 0x56, 0x6f,
	0x6c, 0x75, 0x6d, 0x65, 0x43, 0x6c, 0x61, 0x69, 0x6d, 0x43, 0x6f, 0x64, 0x65, 0x12, 0x33, 0x0a,
	0x0c, 0x68, 0x6f, 0x73, 0x74, 0x5f, 0x61, 0x6c, 0x69, 0x61, 0x73, 0x65, 0x73, 0x18, 0x06, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x48, 0x6f, 0x73, 0x74,
	0x41, 0x6c, 0x69, 0x61, 0x73, 0x52, 0x0b, 0x68, 0x6f, 0x73, 0x74, 0x41, 0x6c, 0x69, 0x61, 0x73,
	0x65, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65, 0x22, 0x49, 0x0a, 0x09, 0x48, 0x6f, 0x73, 0x74,
	0x41, 0x6c, 0x69, 0x61, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x69, 0x70, 0x5f, 0x61, 0x64, 0x64, 0x72,
	0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x69, 0x70, 0x41, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x12, 0x1d, 0x0a, 0x0a, 0x68, 0x6f, 0x73, 0x74, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x09, 0x68, 0x6f, 0x73, 0x74, 0x4e, 0x61,
	0x6d, 0x65, 0x73, 0x22, 0x38, 0x0a, 0x06, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x0e, 0x0a,
	0x02, 0x6f, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x02, 0x6f, 0x6b, 0x12, 0x1e, 0x0a,
	0x0a, 0x61, 0x64, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0a, 0x61, 0x64, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x32, 0x64, 0x0a,
	0x18, 0x41, 0x6c, 0x6c, 0x6f, 0x63, 0x41, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c, 0x69,
	0x74, 0x79, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x48, 0x0a, 0x11, 0x43, 0x68, 0x65,
	0x63, 0x6b, 0x41, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x12, 0x1e,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x41, 0x6c, 0x6c, 0x6f, 0x63, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x1a, 0x13,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x41, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x69, 0x6c,
	0x69, 0x74, 0x79, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_allocation_proto_rawDescOnce sync.Once
	file_allocation_proto_rawDescData = file_allocation_proto_rawDesc
)

func file_allocation_proto_rawDescGZIP() []byte {
	file_allocation_proto_rawDescOnce.Do(func() {
		file_allocation_proto_rawDescData = protoimpl.X.CompressGZIP(file_allocation_proto_rawDescData)
	})
	return file_allocation_proto_rawDescData
}

var file_allocation_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_allocation_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_allocation_proto_goTypes = []interface{}{
	(AllocResponse_Status)(0),       // 0: proto.AllocResponse.Status
	(Availability_Status)(0),        // 1: proto.Availability.Status
	(*AllocRequestDescription)(nil), // 2: proto.AllocRequestDescription
	(*AllocResponse)(nil),           // 3: proto.AllocResponse
	(*Availability)(nil),            // 4: proto.Availability
	(*ContainerResource)(nil),       // 5: proto.ContainerResource
	(*HostAlias)(nil),               // 6: proto.HostAlias
	(*Result)(nil),                  // 7: proto.Result
}
var file_allocation_proto_depIdxs = []int32{
	5, // 0: proto.AllocRequestDescription.descriptions:type_name -> proto.ContainerResource
	7, // 1: proto.AllocResponse.results:type_name -> proto.Result
	7, // 2: proto.Availability.results:type_name -> proto.Result
	6, // 3: proto.ContainerResource.host_aliases:type_name -> proto.HostAlias
	2, // 4: proto.AllocAvailabilityService.CheckAvailability:input_type -> proto.AllocRequestDescription
	4, // 5: proto.AllocAvailabilityService.CheckAvailability:output_type -> proto.Availability
	5, // [5:6] is the sub-list for method output_type
	4, // [4:5] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_allocation_proto_init() }
func file_allocation_proto_init() {
	if File_allocation_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_allocation_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AllocRequestDescription); i {
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
		file_allocation_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AllocResponse); i {
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
		file_allocation_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Availability); i {
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
		file_allocation_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ContainerResource); i {
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
		file_allocation_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HostAlias); i {
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
		file_allocation_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Result); i {
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
			RawDescriptor: file_allocation_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_allocation_proto_goTypes,
		DependencyIndexes: file_allocation_proto_depIdxs,
		EnumInfos:         file_allocation_proto_enumTypes,
		MessageInfos:      file_allocation_proto_msgTypes,
	}.Build()
	File_allocation_proto = out.File
	file_allocation_proto_rawDesc = nil
	file_allocation_proto_goTypes = nil
	file_allocation_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// AllocAvailabilityServiceClient is the client API for AllocAvailabilityService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type AllocAvailabilityServiceClient interface {
	CheckAvailability(ctx context.Context, in *AllocRequestDescription, opts ...grpc.CallOption) (*Availability, error)
}

type allocAvailabilityServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAllocAvailabilityServiceClient(cc grpc.ClientConnInterface) AllocAvailabilityServiceClient {
	return &allocAvailabilityServiceClient{cc}
}

func (c *allocAvailabilityServiceClient) CheckAvailability(ctx context.Context, in *AllocRequestDescription, opts ...grpc.CallOption) (*Availability, error) {
	out := new(Availability)
	err := c.cc.Invoke(ctx, "/proto.AllocAvailabilityService/CheckAvailability", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AllocAvailabilityServiceServer is the server API for AllocAvailabilityService service.
type AllocAvailabilityServiceServer interface {
	CheckAvailability(context.Context, *AllocRequestDescription) (*Availability, error)
}

// UnimplementedAllocAvailabilityServiceServer can be embedded to have forward compatible implementations.
type UnimplementedAllocAvailabilityServiceServer struct {
}

func (*UnimplementedAllocAvailabilityServiceServer) CheckAvailability(context.Context, *AllocRequestDescription) (*Availability, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckAvailability not implemented")
}

func RegisterAllocAvailabilityServiceServer(s *grpc.Server, srv AllocAvailabilityServiceServer) {
	s.RegisterService(&_AllocAvailabilityService_serviceDesc, srv)
}

func _AllocAvailabilityService_CheckAvailability_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AllocRequestDescription)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AllocAvailabilityServiceServer).CheckAvailability(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.AllocAvailabilityService/CheckAvailability",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AllocAvailabilityServiceServer).CheckAvailability(ctx, req.(*AllocRequestDescription))
	}
	return interceptor(ctx, in, info, handler)
}

var _AllocAvailabilityService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "proto.AllocAvailabilityService",
	HandlerType: (*AllocAvailabilityServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CheckAvailability",
			Handler:    _AllocAvailabilityService_CheckAvailability_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "allocation.proto",
}