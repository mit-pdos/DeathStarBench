// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.20.0
// 	protoc        v3.12.4
// source: services/post/proto/post.proto

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

type POST_TYPE int32

const (
	POST_TYPE_UNKNOWN POST_TYPE = 0
	POST_TYPE_POST    POST_TYPE = 1
	POST_TYPE_REPOST  POST_TYPE = 2
	POST_TYPE_REPLY   POST_TYPE = 3
	POST_TYPE_DM      POST_TYPE = 4
)

// Enum value maps for POST_TYPE.
var (
	POST_TYPE_name = map[int32]string{
		0: "UNKNOWN",
		1: "POST",
		2: "REPOST",
		3: "REPLY",
		4: "DM",
	}
	POST_TYPE_value = map[string]int32{
		"UNKNOWN": 0,
		"POST":    1,
		"REPOST":  2,
		"REPLY":   3,
		"DM":      4,
	}
)

func (x POST_TYPE) Enum() *POST_TYPE {
	p := new(POST_TYPE)
	*p = x
	return p
}

func (x POST_TYPE) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (POST_TYPE) Descriptor() protoreflect.EnumDescriptor {
	return file_services_post_proto_post_proto_enumTypes[0].Descriptor()
}

func (POST_TYPE) Type() protoreflect.EnumType {
	return &file_services_post_proto_post_proto_enumTypes[0]
}

func (x POST_TYPE) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use POST_TYPE.Descriptor instead.
func (POST_TYPE) EnumDescriptor() ([]byte, []int) {
	return file_services_post_proto_post_proto_rawDescGZIP(), []int{0}
}

type StorePostRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Post *Post `protobuf:"bytes,1,opt,name=post,proto3" json:"post,omitempty"`
}

func (x *StorePostRequest) Reset() {
	*x = StorePostRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_services_post_proto_post_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StorePostRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StorePostRequest) ProtoMessage() {}

func (x *StorePostRequest) ProtoReflect() protoreflect.Message {
	mi := &file_services_post_proto_post_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StorePostRequest.ProtoReflect.Descriptor instead.
func (*StorePostRequest) Descriptor() ([]byte, []int) {
	return file_services_post_proto_post_proto_rawDescGZIP(), []int{0}
}

func (x *StorePostRequest) GetPost() *Post {
	if x != nil {
		return x.Post
	}
	return nil
}

type StorePostResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ok string `protobuf:"bytes,1,opt,name=ok,proto3" json:"ok,omitempty"`
}

func (x *StorePostResponse) Reset() {
	*x = StorePostResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_services_post_proto_post_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StorePostResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StorePostResponse) ProtoMessage() {}

func (x *StorePostResponse) ProtoReflect() protoreflect.Message {
	mi := &file_services_post_proto_post_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StorePostResponse.ProtoReflect.Descriptor instead.
func (*StorePostResponse) Descriptor() ([]byte, []int) {
	return file_services_post_proto_post_proto_rawDescGZIP(), []int{1}
}

func (x *StorePostResponse) GetOk() string {
	if x != nil {
		return x.Ok
	}
	return ""
}

type ReadPostsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Postids []int64 `protobuf:"varint,1,rep,packed,name=postids,proto3" json:"postids,omitempty"`
}

func (x *ReadPostsRequest) Reset() {
	*x = ReadPostsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_services_post_proto_post_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReadPostsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadPostsRequest) ProtoMessage() {}

func (x *ReadPostsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_services_post_proto_post_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReadPostsRequest.ProtoReflect.Descriptor instead.
func (*ReadPostsRequest) Descriptor() ([]byte, []int) {
	return file_services_post_proto_post_proto_rawDescGZIP(), []int{2}
}

func (x *ReadPostsRequest) GetPostids() []int64 {
	if x != nil {
		return x.Postids
	}
	return nil
}

type ReadPostsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ok    string  `protobuf:"bytes,1,opt,name=ok,proto3" json:"ok,omitempty"`
	Posts []*Post `protobuf:"bytes,2,rep,name=posts,proto3" json:"posts,omitempty"`
}

func (x *ReadPostsResponse) Reset() {
	*x = ReadPostsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_services_post_proto_post_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReadPostsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReadPostsResponse) ProtoMessage() {}

func (x *ReadPostsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_services_post_proto_post_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReadPostsResponse.ProtoReflect.Descriptor instead.
func (*ReadPostsResponse) Descriptor() ([]byte, []int) {
	return file_services_post_proto_post_proto_rawDescGZIP(), []int{3}
}

func (x *ReadPostsResponse) GetOk() string {
	if x != nil {
		return x.Ok
	}
	return ""
}

func (x *ReadPostsResponse) GetPosts() []*Post {
	if x != nil {
		return x.Posts
	}
	return nil
}

type Post struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Postid       int64     `protobuf:"varint,1,opt,name=postid,proto3" json:"postid,omitempty"`
	Posttype     POST_TYPE `protobuf:"varint,2,opt,name=posttype,proto3,enum=post.POST_TYPE" json:"posttype,omitempty"`
	Timestamp    int64     `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Creator      int64     `protobuf:"varint,4,opt,name=creator,proto3" json:"creator,omitempty"`
	Creatoruname string    `protobuf:"bytes,5,opt,name=creatoruname,proto3" json:"creatoruname,omitempty"`
	Text         string    `protobuf:"bytes,6,opt,name=text,proto3" json:"text,omitempty"`
	Usermentions []int64   `protobuf:"varint,7,rep,packed,name=usermentions,proto3" json:"usermentions,omitempty"`
	Medias       []int64   `protobuf:"varint,8,rep,packed,name=medias,proto3" json:"medias,omitempty"`
	Urls         []string  `protobuf:"bytes,9,rep,name=urls,proto3" json:"urls,omitempty"`
}

func (x *Post) Reset() {
	*x = Post{}
	if protoimpl.UnsafeEnabled {
		mi := &file_services_post_proto_post_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Post) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Post) ProtoMessage() {}

func (x *Post) ProtoReflect() protoreflect.Message {
	mi := &file_services_post_proto_post_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Post.ProtoReflect.Descriptor instead.
func (*Post) Descriptor() ([]byte, []int) {
	return file_services_post_proto_post_proto_rawDescGZIP(), []int{4}
}

func (x *Post) GetPostid() int64 {
	if x != nil {
		return x.Postid
	}
	return 0
}

func (x *Post) GetPosttype() POST_TYPE {
	if x != nil {
		return x.Posttype
	}
	return POST_TYPE_UNKNOWN
}

func (x *Post) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *Post) GetCreator() int64 {
	if x != nil {
		return x.Creator
	}
	return 0
}

func (x *Post) GetCreatoruname() string {
	if x != nil {
		return x.Creatoruname
	}
	return ""
}

func (x *Post) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

func (x *Post) GetUsermentions() []int64 {
	if x != nil {
		return x.Usermentions
	}
	return nil
}

func (x *Post) GetMedias() []int64 {
	if x != nil {
		return x.Medias
	}
	return nil
}

func (x *Post) GetUrls() []string {
	if x != nil {
		return x.Urls
	}
	return nil
}

var File_services_post_proto_post_proto protoreflect.FileDescriptor

var file_services_post_proto_post_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2f, 0x70, 0x6f, 0x73, 0x74, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x70, 0x6f, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x04, 0x70, 0x6f, 0x73, 0x74, 0x22, 0x32, 0x0a, 0x10, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x50,
	0x6f, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1e, 0x0a, 0x04, 0x70, 0x6f,
	0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x70, 0x6f, 0x73, 0x74, 0x2e,
	0x50, 0x6f, 0x73, 0x74, 0x52, 0x04, 0x70, 0x6f, 0x73, 0x74, 0x22, 0x23, 0x0a, 0x11, 0x53, 0x74,
	0x6f, 0x72, 0x65, 0x50, 0x6f, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x0e, 0x0a, 0x02, 0x6f, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x6f, 0x6b, 0x22,
	0x2c, 0x0a, 0x10, 0x52, 0x65, 0x61, 0x64, 0x50, 0x6f, 0x73, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x6f, 0x73, 0x74, 0x69, 0x64, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x03, 0x52, 0x07, 0x70, 0x6f, 0x73, 0x74, 0x69, 0x64, 0x73, 0x22, 0x45, 0x0a,
	0x11, 0x52, 0x65, 0x61, 0x64, 0x50, 0x6f, 0x73, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x6f, 0x6b, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02,
	0x6f, 0x6b, 0x12, 0x20, 0x0a, 0x05, 0x70, 0x6f, 0x73, 0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x0a, 0x2e, 0x70, 0x6f, 0x73, 0x74, 0x2e, 0x50, 0x6f, 0x73, 0x74, 0x52, 0x05, 0x70,
	0x6f, 0x73, 0x74, 0x73, 0x22, 0x8b, 0x02, 0x0a, 0x04, 0x50, 0x6f, 0x73, 0x74, 0x12, 0x16, 0x0a,
	0x06, 0x70, 0x6f, 0x73, 0x74, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x70,
	0x6f, 0x73, 0x74, 0x69, 0x64, 0x12, 0x2b, 0x0a, 0x08, 0x70, 0x6f, 0x73, 0x74, 0x74, 0x79, 0x70,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0f, 0x2e, 0x70, 0x6f, 0x73, 0x74, 0x2e, 0x50,
	0x4f, 0x53, 0x54, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x52, 0x08, 0x70, 0x6f, 0x73, 0x74, 0x74, 0x79,
	0x70, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x12, 0x18, 0x0a, 0x07, 0x63, 0x72, 0x65, 0x61, 0x74, 0x6f, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x07, 0x63, 0x72, 0x65, 0x61, 0x74, 0x6f, 0x72, 0x12, 0x22, 0x0a, 0x0c, 0x63, 0x72,
	0x65, 0x61, 0x74, 0x6f, 0x72, 0x75, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0c, 0x63, 0x72, 0x65, 0x61, 0x74, 0x6f, 0x72, 0x75, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x74, 0x65, 0x78, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x65,
	0x78, 0x74, 0x12, 0x22, 0x0a, 0x0c, 0x75, 0x73, 0x65, 0x72, 0x6d, 0x65, 0x6e, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28, 0x03, 0x52, 0x0c, 0x75, 0x73, 0x65, 0x72, 0x6d, 0x65,
	0x6e, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x64, 0x69, 0x61, 0x73,
	0x18, 0x08, 0x20, 0x03, 0x28, 0x03, 0x52, 0x06, 0x6d, 0x65, 0x64, 0x69, 0x61, 0x73, 0x12, 0x12,
	0x0a, 0x04, 0x75, 0x72, 0x6c, 0x73, 0x18, 0x09, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x75, 0x72,
	0x6c, 0x73, 0x2a, 0x41, 0x0a, 0x09, 0x50, 0x4f, 0x53, 0x54, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x12,
	0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04,
	0x50, 0x4f, 0x53, 0x54, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x52, 0x45, 0x50, 0x4f, 0x53, 0x54,
	0x10, 0x02, 0x12, 0x09, 0x0a, 0x05, 0x52, 0x45, 0x50, 0x4c, 0x59, 0x10, 0x03, 0x12, 0x06, 0x0a,
	0x02, 0x44, 0x4d, 0x10, 0x04, 0x32, 0x89, 0x01, 0x0a, 0x0b, 0x50, 0x6f, 0x73, 0x74, 0x53, 0x74,
	0x6f, 0x72, 0x61, 0x67, 0x65, 0x12, 0x3c, 0x0a, 0x09, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x50, 0x6f,
	0x73, 0x74, 0x12, 0x16, 0x2e, 0x70, 0x6f, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x50,
	0x6f, 0x73, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e, 0x70, 0x6f, 0x73,
	0x74, 0x2e, 0x53, 0x74, 0x6f, 0x72, 0x65, 0x50, 0x6f, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x3c, 0x0a, 0x09, 0x52, 0x65, 0x61, 0x64, 0x50, 0x6f, 0x73, 0x74, 0x73,
	0x12, 0x16, 0x2e, 0x70, 0x6f, 0x73, 0x74, 0x2e, 0x52, 0x65, 0x61, 0x64, 0x50, 0x6f, 0x73, 0x74,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e, 0x70, 0x6f, 0x73, 0x74, 0x2e,
	0x52, 0x65, 0x61, 0x64, 0x50, 0x6f, 0x73, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x42, 0x17, 0x5a, 0x15, 0x2e, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2f,
	0x70, 0x6f, 0x73, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_services_post_proto_post_proto_rawDescOnce sync.Once
	file_services_post_proto_post_proto_rawDescData = file_services_post_proto_post_proto_rawDesc
)

func file_services_post_proto_post_proto_rawDescGZIP() []byte {
	file_services_post_proto_post_proto_rawDescOnce.Do(func() {
		file_services_post_proto_post_proto_rawDescData = protoimpl.X.CompressGZIP(file_services_post_proto_post_proto_rawDescData)
	})
	return file_services_post_proto_post_proto_rawDescData
}

var file_services_post_proto_post_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_services_post_proto_post_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_services_post_proto_post_proto_goTypes = []interface{}{
	(POST_TYPE)(0),            // 0: post.POST_TYPE
	(*StorePostRequest)(nil),  // 1: post.StorePostRequest
	(*StorePostResponse)(nil), // 2: post.StorePostResponse
	(*ReadPostsRequest)(nil),  // 3: post.ReadPostsRequest
	(*ReadPostsResponse)(nil), // 4: post.ReadPostsResponse
	(*Post)(nil),              // 5: post.Post
}
var file_services_post_proto_post_proto_depIdxs = []int32{
	5, // 0: post.StorePostRequest.post:type_name -> post.Post
	5, // 1: post.ReadPostsResponse.posts:type_name -> post.Post
	0, // 2: post.Post.posttype:type_name -> post.POST_TYPE
	1, // 3: post.PostStorage.StorePost:input_type -> post.StorePostRequest
	3, // 4: post.PostStorage.ReadPosts:input_type -> post.ReadPostsRequest
	2, // 5: post.PostStorage.StorePost:output_type -> post.StorePostResponse
	4, // 6: post.PostStorage.ReadPosts:output_type -> post.ReadPostsResponse
	5, // [5:7] is the sub-list for method output_type
	3, // [3:5] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_services_post_proto_post_proto_init() }
func file_services_post_proto_post_proto_init() {
	if File_services_post_proto_post_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_services_post_proto_post_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StorePostRequest); i {
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
		file_services_post_proto_post_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StorePostResponse); i {
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
		file_services_post_proto_post_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReadPostsRequest); i {
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
		file_services_post_proto_post_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReadPostsResponse); i {
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
		file_services_post_proto_post_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Post); i {
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
			RawDescriptor: file_services_post_proto_post_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_services_post_proto_post_proto_goTypes,
		DependencyIndexes: file_services_post_proto_post_proto_depIdxs,
		EnumInfos:         file_services_post_proto_post_proto_enumTypes,
		MessageInfos:      file_services_post_proto_post_proto_msgTypes,
	}.Build()
	File_services_post_proto_post_proto = out.File
	file_services_post_proto_post_proto_rawDesc = nil
	file_services_post_proto_post_proto_goTypes = nil
	file_services_post_proto_post_proto_depIdxs = nil
}
