// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        (unknown)
// source: infrabin/v1/infrabin.proto

package infrabinv1

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

type HeadersRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *HeadersRequest) Reset() {
	*x = HeadersRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infrabin_v1_infrabin_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HeadersRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HeadersRequest) ProtoMessage() {}

func (x *HeadersRequest) ProtoReflect() protoreflect.Message {
	mi := &file_infrabin_v1_infrabin_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HeadersRequest.ProtoReflect.Descriptor instead.
func (*HeadersRequest) Descriptor() ([]byte, []int) {
	return file_infrabin_v1_infrabin_proto_rawDescGZIP(), []int{0}
}

type HeadersResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Headers map[string]string `protobuf:"bytes,1,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *HeadersResponse) Reset() {
	*x = HeadersResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infrabin_v1_infrabin_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HeadersResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HeadersResponse) ProtoMessage() {}

func (x *HeadersResponse) ProtoReflect() protoreflect.Message {
	mi := &file_infrabin_v1_infrabin_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HeadersResponse.ProtoReflect.Descriptor instead.
func (*HeadersResponse) Descriptor() ([]byte, []int) {
	return file_infrabin_v1_infrabin_proto_rawDescGZIP(), []int{1}
}

func (x *HeadersResponse) GetHeaders() map[string]string {
	if x != nil {
		return x.Headers
	}
	return nil
}

type EnvRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *EnvRequest) Reset() {
	*x = EnvRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infrabin_v1_infrabin_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvRequest) ProtoMessage() {}

func (x *EnvRequest) ProtoReflect() protoreflect.Message {
	mi := &file_infrabin_v1_infrabin_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvRequest.ProtoReflect.Descriptor instead.
func (*EnvRequest) Descriptor() ([]byte, []int) {
	return file_infrabin_v1_infrabin_proto_rawDescGZIP(), []int{2}
}

func (x *EnvRequest) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

type EnvResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Environment map[string]string `protobuf:"bytes,1,rep,name=environment,proto3" json:"environment,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *EnvResponse) Reset() {
	*x = EnvResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_infrabin_v1_infrabin_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvResponse) ProtoMessage() {}

func (x *EnvResponse) ProtoReflect() protoreflect.Message {
	mi := &file_infrabin_v1_infrabin_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvResponse.ProtoReflect.Descriptor instead.
func (*EnvResponse) Descriptor() ([]byte, []int) {
	return file_infrabin_v1_infrabin_proto_rawDescGZIP(), []int{3}
}

func (x *EnvResponse) GetEnvironment() map[string]string {
	if x != nil {
		return x.Environment
	}
	return nil
}

var File_infrabin_v1_infrabin_proto protoreflect.FileDescriptor

var file_infrabin_v1_infrabin_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x62, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x2f, 0x69, 0x6e,
	0x66, 0x72, 0x61, 0x62, 0x69, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x69, 0x6e,
	0x66, 0x72, 0x61, 0x62, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x22, 0x10, 0x0a, 0x0e, 0x48, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x92, 0x01, 0x0a, 0x0f,
	0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x43, 0x0a, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x29, 0x2e, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x62, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x48,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x48,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x68, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x73, 0x1a, 0x3a, 0x0a, 0x0c, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x22, 0x1e, 0x0a, 0x0a, 0x45, 0x6e, 0x76, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x22, 0x9a, 0x01, 0x0a, 0x0b, 0x45, 0x6e, 0x76, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x4b, 0x0a, 0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x62, 0x69, 0x6e,
	0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e,
	0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x1a, 0x3e, 0x0a,
	0x10, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x32, 0x95, 0x01,
	0x0a, 0x0f, 0x49, 0x6e, 0x66, 0x72, 0x61, 0x62, 0x69, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x12, 0x46, 0x0a, 0x07, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x12, 0x1b, 0x2e, 0x69,
	0x6e, 0x66, 0x72, 0x61, 0x62, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65,
	0x72, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1c, 0x2e, 0x69, 0x6e, 0x66, 0x72,
	0x61, 0x62, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x3a, 0x0a, 0x03, 0x45, 0x6e, 0x76,
	0x12, 0x17, 0x2e, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x62, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x45,
	0x6e, 0x76, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e, 0x69, 0x6e, 0x66, 0x72,
	0x61, 0x62, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x3b, 0x5a, 0x39, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x6d, 0x61, 0x72, 0x75, 0x69, 0x6e, 0x61, 0x2f, 0x67, 0x6f, 0x2d, 0x69,
	0x6e, 0x66, 0x72, 0x61, 0x62, 0x69, 0x6e, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x69, 0x6e, 0x66, 0x72,
	0x61, 0x62, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x3b, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x62, 0x69, 0x6e,
	0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_infrabin_v1_infrabin_proto_rawDescOnce sync.Once
	file_infrabin_v1_infrabin_proto_rawDescData = file_infrabin_v1_infrabin_proto_rawDesc
)

func file_infrabin_v1_infrabin_proto_rawDescGZIP() []byte {
	file_infrabin_v1_infrabin_proto_rawDescOnce.Do(func() {
		file_infrabin_v1_infrabin_proto_rawDescData = protoimpl.X.CompressGZIP(file_infrabin_v1_infrabin_proto_rawDescData)
	})
	return file_infrabin_v1_infrabin_proto_rawDescData
}

var file_infrabin_v1_infrabin_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_infrabin_v1_infrabin_proto_goTypes = []interface{}{
	(*HeadersRequest)(nil),  // 0: infrabin.v1.HeadersRequest
	(*HeadersResponse)(nil), // 1: infrabin.v1.HeadersResponse
	(*EnvRequest)(nil),      // 2: infrabin.v1.EnvRequest
	(*EnvResponse)(nil),     // 3: infrabin.v1.EnvResponse
	nil,                     // 4: infrabin.v1.HeadersResponse.HeadersEntry
	nil,                     // 5: infrabin.v1.EnvResponse.EnvironmentEntry
}
var file_infrabin_v1_infrabin_proto_depIdxs = []int32{
	4, // 0: infrabin.v1.HeadersResponse.headers:type_name -> infrabin.v1.HeadersResponse.HeadersEntry
	5, // 1: infrabin.v1.EnvResponse.environment:type_name -> infrabin.v1.EnvResponse.EnvironmentEntry
	0, // 2: infrabin.v1.InfrabinService.Headers:input_type -> infrabin.v1.HeadersRequest
	2, // 3: infrabin.v1.InfrabinService.Env:input_type -> infrabin.v1.EnvRequest
	1, // 4: infrabin.v1.InfrabinService.Headers:output_type -> infrabin.v1.HeadersResponse
	3, // 5: infrabin.v1.InfrabinService.Env:output_type -> infrabin.v1.EnvResponse
	4, // [4:6] is the sub-list for method output_type
	2, // [2:4] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_infrabin_v1_infrabin_proto_init() }
func file_infrabin_v1_infrabin_proto_init() {
	if File_infrabin_v1_infrabin_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_infrabin_v1_infrabin_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HeadersRequest); i {
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
		file_infrabin_v1_infrabin_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HeadersResponse); i {
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
		file_infrabin_v1_infrabin_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvRequest); i {
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
		file_infrabin_v1_infrabin_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvResponse); i {
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
			RawDescriptor: file_infrabin_v1_infrabin_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_infrabin_v1_infrabin_proto_goTypes,
		DependencyIndexes: file_infrabin_v1_infrabin_proto_depIdxs,
		MessageInfos:      file_infrabin_v1_infrabin_proto_msgTypes,
	}.Build()
	File_infrabin_v1_infrabin_proto = out.File
	file_infrabin_v1_infrabin_proto_rawDesc = nil
	file_infrabin_v1_infrabin_proto_goTypes = nil
	file_infrabin_v1_infrabin_proto_depIdxs = nil
}
