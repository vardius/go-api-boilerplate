// Code generated by protoc-gen-go. DO NOT EDIT.
// source: user.proto

package proto

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// DispatchCommandRequest is passed when dispatching
type DispatchCommandRequest struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Payload              []byte   `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DispatchCommandRequest) Reset()         { *m = DispatchCommandRequest{} }
func (m *DispatchCommandRequest) String() string { return proto.CompactTextString(m) }
func (*DispatchCommandRequest) ProtoMessage()    {}
func (*DispatchCommandRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_116e343673f7ffaf, []int{0}
}

func (m *DispatchCommandRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DispatchCommandRequest.Unmarshal(m, b)
}
func (m *DispatchCommandRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DispatchCommandRequest.Marshal(b, m, deterministic)
}
func (m *DispatchCommandRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DispatchCommandRequest.Merge(m, src)
}
func (m *DispatchCommandRequest) XXX_Size() int {
	return xxx_messageInfo_DispatchCommandRequest.Size(m)
}
func (m *DispatchCommandRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DispatchCommandRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DispatchCommandRequest proto.InternalMessageInfo

func (m *DispatchCommandRequest) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *DispatchCommandRequest) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

// User object
type User struct {
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Provider             string   `protobuf:"bytes,2,opt,name=provider,proto3" json:"provider,omitempty"`
	Name                 string   `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Email                string   `protobuf:"bytes,4,opt,name=email,proto3" json:"email,omitempty"`
	Nickname             string   `protobuf:"bytes,5,opt,name=nickname,proto3" json:"nickname,omitempty"`
	Location             string   `protobuf:"bytes,6,opt,name=location,proto3" json:"location,omitempty"`
	Avatarurl            string   `protobuf:"bytes,7,opt,name=avatarurl,proto3" json:"avatarurl,omitempty"`
	Description          string   `protobuf:"bytes,8,opt,name=description,proto3" json:"description,omitempty"`
	Userid               string   `protobuf:"bytes,9,opt,name=userid,proto3" json:"userid,omitempty"`
	Refreshtoken         string   `protobuf:"bytes,10,opt,name=refreshtoken,proto3" json:"refreshtoken,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *User) Reset()         { *m = User{} }
func (m *User) String() string { return proto.CompactTextString(m) }
func (*User) ProtoMessage()    {}
func (*User) Descriptor() ([]byte, []int) {
	return fileDescriptor_116e343673f7ffaf, []int{1}
}

func (m *User) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_User.Unmarshal(m, b)
}
func (m *User) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_User.Marshal(b, m, deterministic)
}
func (m *User) XXX_Merge(src proto.Message) {
	xxx_messageInfo_User.Merge(m, src)
}
func (m *User) XXX_Size() int {
	return xxx_messageInfo_User.Size(m)
}
func (m *User) XXX_DiscardUnknown() {
	xxx_messageInfo_User.DiscardUnknown(m)
}

var xxx_messageInfo_User proto.InternalMessageInfo

func (m *User) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *User) GetProvider() string {
	if m != nil {
		return m.Provider
	}
	return ""
}

func (m *User) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *User) GetEmail() string {
	if m != nil {
		return m.Email
	}
	return ""
}

func (m *User) GetNickname() string {
	if m != nil {
		return m.Nickname
	}
	return ""
}

func (m *User) GetLocation() string {
	if m != nil {
		return m.Location
	}
	return ""
}

func (m *User) GetAvatarurl() string {
	if m != nil {
		return m.Avatarurl
	}
	return ""
}

func (m *User) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *User) GetUserid() string {
	if m != nil {
		return m.Userid
	}
	return ""
}

func (m *User) GetRefreshtoken() string {
	if m != nil {
		return m.Refreshtoken
	}
	return ""
}

// GetUserRequest is a request data to read user
type GetUserRequest struct {
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetUserRequest) Reset()         { *m = GetUserRequest{} }
func (m *GetUserRequest) String() string { return proto.CompactTextString(m) }
func (*GetUserRequest) ProtoMessage()    {}
func (*GetUserRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_116e343673f7ffaf, []int{2}
}

func (m *GetUserRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetUserRequest.Unmarshal(m, b)
}
func (m *GetUserRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetUserRequest.Marshal(b, m, deterministic)
}
func (m *GetUserRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetUserRequest.Merge(m, src)
}
func (m *GetUserRequest) XXX_Size() int {
	return xxx_messageInfo_GetUserRequest.Size(m)
}
func (m *GetUserRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetUserRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetUserRequest proto.InternalMessageInfo

func (m *GetUserRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

// ListUserRequest is a request data to read all user for a given page
type ListUserRequest struct {
	Page                 int32    `protobuf:"varint,1,opt,name=page,proto3" json:"page,omitempty"`
	Limit                int32    `protobuf:"varint,2,opt,name=limit,proto3" json:"limit,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListUserRequest) Reset()         { *m = ListUserRequest{} }
func (m *ListUserRequest) String() string { return proto.CompactTextString(m) }
func (*ListUserRequest) ProtoMessage()    {}
func (*ListUserRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_116e343673f7ffaf, []int{3}
}

func (m *ListUserRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListUserRequest.Unmarshal(m, b)
}
func (m *ListUserRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListUserRequest.Marshal(b, m, deterministic)
}
func (m *ListUserRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListUserRequest.Merge(m, src)
}
func (m *ListUserRequest) XXX_Size() int {
	return xxx_messageInfo_ListUserRequest.Size(m)
}
func (m *ListUserRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListUserRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListUserRequest proto.InternalMessageInfo

func (m *ListUserRequest) GetPage() int32 {
	if m != nil {
		return m.Page
	}
	return 0
}

func (m *ListUserRequest) GetLimit() int32 {
	if m != nil {
		return m.Limit
	}
	return 0
}

// ListUserResponse list of all users
type ListUserResponse struct {
	Users                []*User  `protobuf:"bytes,1,rep,name=users,proto3" json:"users,omitempty"`
	Page                 int32    `protobuf:"varint,2,opt,name=page,proto3" json:"page,omitempty"`
	Limit                int32    `protobuf:"varint,3,opt,name=limit,proto3" json:"limit,omitempty"`
	Total                int32    `protobuf:"varint,4,opt,name=total,proto3" json:"total,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListUserResponse) Reset()         { *m = ListUserResponse{} }
func (m *ListUserResponse) String() string { return proto.CompactTextString(m) }
func (*ListUserResponse) ProtoMessage()    {}
func (*ListUserResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_116e343673f7ffaf, []int{4}
}

func (m *ListUserResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListUserResponse.Unmarshal(m, b)
}
func (m *ListUserResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListUserResponse.Marshal(b, m, deterministic)
}
func (m *ListUserResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListUserResponse.Merge(m, src)
}
func (m *ListUserResponse) XXX_Size() int {
	return xxx_messageInfo_ListUserResponse.Size(m)
}
func (m *ListUserResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ListUserResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ListUserResponse proto.InternalMessageInfo

func (m *ListUserResponse) GetUsers() []*User {
	if m != nil {
		return m.Users
	}
	return nil
}

func (m *ListUserResponse) GetPage() int32 {
	if m != nil {
		return m.Page
	}
	return 0
}

func (m *ListUserResponse) GetLimit() int32 {
	if m != nil {
		return m.Limit
	}
	return 0
}

func (m *ListUserResponse) GetTotal() int32 {
	if m != nil {
		return m.Total
	}
	return 0
}

func init() {
	proto.RegisterType((*DispatchCommandRequest)(nil), "proto.DispatchCommandRequest")
	proto.RegisterType((*User)(nil), "proto.User")
	proto.RegisterType((*GetUserRequest)(nil), "proto.GetUserRequest")
	proto.RegisterType((*ListUserRequest)(nil), "proto.ListUserRequest")
	proto.RegisterType((*ListUserResponse)(nil), "proto.ListUserResponse")
}

func init() { proto.RegisterFile("user.proto", fileDescriptor_116e343673f7ffaf) }

var fileDescriptor_116e343673f7ffaf = []byte{
	// 454 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x52, 0xcb, 0x6e, 0xdb, 0x30,
	0x10, 0x84, 0x6c, 0xcb, 0x8e, 0xd6, 0x41, 0x52, 0x2c, 0x5a, 0x57, 0x50, 0x5a, 0x40, 0xd5, 0xc9,
	0x28, 0x50, 0x05, 0x48, 0x8f, 0xed, 0x29, 0x7d, 0x1e, 0x7a, 0x08, 0x54, 0xf4, 0x03, 0x68, 0x69,
	0xa3, 0x10, 0x91, 0x44, 0x96, 0xa4, 0x0c, 0xe4, 0x2f, 0xfa, 0x51, 0xfd, 0xb0, 0x82, 0xa4, 0xe4,
	0xd8, 0x6e, 0x4e, 0xe2, 0xcc, 0xec, 0xee, 0x88, 0xc3, 0x05, 0xe8, 0x35, 0xa9, 0x5c, 0x2a, 0x61,
	0x04, 0x86, 0xee, 0x93, 0x5c, 0xd4, 0x42, 0xd4, 0x0d, 0x5d, 0x3a, 0xb4, 0xe9, 0x6f, 0x2f, 0xa9,
	0x95, 0xe6, 0xc1, 0xd7, 0x64, 0x5f, 0x61, 0xf5, 0x99, 0x6b, 0xc9, 0x4c, 0x79, 0xf7, 0x49, 0xb4,
	0x2d, 0xeb, 0xaa, 0x82, 0x7e, 0xf7, 0xa4, 0x0d, 0x22, 0xcc, 0x3a, 0xd6, 0x52, 0x1c, 0xa4, 0xc1,
	0x3a, 0x2a, 0xdc, 0x19, 0x63, 0x58, 0x48, 0xf6, 0xd0, 0x08, 0x56, 0xc5, 0x93, 0x34, 0x58, 0x9f,
	0x16, 0x23, 0xcc, 0xfe, 0x4c, 0x60, 0xf6, 0x4b, 0x93, 0xc2, 0x33, 0x98, 0xf0, 0x6a, 0x68, 0x9a,
	0xf0, 0x0a, 0x13, 0x38, 0x91, 0x4a, 0x6c, 0x79, 0x45, 0xca, 0xf5, 0x44, 0xc5, 0x0e, 0xef, 0x2c,
	0xa6, 0x7b, 0x16, 0xcf, 0x21, 0xa4, 0x96, 0xf1, 0x26, 0x9e, 0x39, 0xd2, 0x03, 0x3b, 0xa5, 0xe3,
	0xe5, 0xbd, 0xab, 0x0e, 0xfd, 0x94, 0x11, 0x5b, 0xad, 0x11, 0x25, 0x33, 0x5c, 0x74, 0xf1, 0xdc,
	0x6b, 0x23, 0xc6, 0x57, 0x10, 0xb1, 0x2d, 0x33, 0x4c, 0xf5, 0xaa, 0x89, 0x17, 0x4e, 0x7c, 0x24,
	0x30, 0x85, 0x65, 0x45, 0xba, 0x54, 0x5c, 0xba, 0xe6, 0x13, 0xa7, 0xef, 0x53, 0xb8, 0x82, 0xb9,
	0x0d, 0x94, 0x57, 0x71, 0xe4, 0xc4, 0x01, 0x61, 0x06, 0xa7, 0x8a, 0x6e, 0x15, 0xe9, 0x3b, 0x23,
	0xee, 0xa9, 0x8b, 0xc1, 0xa9, 0x07, 0x5c, 0x96, 0xc2, 0xd9, 0x37, 0x32, 0x36, 0x94, 0x31, 0xd2,
	0xa3, 0x6c, 0xb2, 0x0f, 0x70, 0xfe, 0x83, 0xeb, 0x83, 0x12, 0x84, 0x99, 0x64, 0xb5, 0x4f, 0x3d,
	0x2c, 0xdc, 0xd9, 0x46, 0xd2, 0xf0, 0x96, 0x1b, 0x97, 0x5f, 0x58, 0x78, 0x90, 0xf5, 0xf0, 0xec,
	0xb1, 0x59, 0x4b, 0xd1, 0x69, 0xc2, 0x37, 0x10, 0xda, 0x1f, 0xd4, 0x71, 0x90, 0x4e, 0xd7, 0xcb,
	0xab, 0xa5, 0x7f, 0xe4, 0xdc, 0xd5, 0x78, 0x65, 0x67, 0x30, 0x79, 0xca, 0x60, 0xba, 0x67, 0x60,
	0x59, 0x23, 0x0c, 0xf3, 0x2f, 0x11, 0x16, 0x1e, 0x5c, 0xfd, 0x0d, 0x60, 0x69, 0xe7, 0xfd, 0x24,
	0xb5, 0xe5, 0x25, 0xe1, 0x77, 0x38, 0x3f, 0x5a, 0x20, 0x7c, 0x3d, 0xd8, 0x3e, 0xbd, 0x58, 0xc9,
	0x2a, 0xf7, 0x0b, 0x99, 0x8f, 0x0b, 0x99, 0x7f, 0xb1, 0x0b, 0x89, 0xef, 0x60, 0x31, 0xe4, 0x85,
	0x2f, 0x86, 0x09, 0x87, 0xf9, 0x25, 0xfb, 0xf7, 0xc1, 0x8f, 0x10, 0x8d, 0xf7, 0xd7, 0xb8, 0x1a,
	0x94, 0xa3, 0x38, 0x93, 0x97, 0xff, 0xf1, 0x3e, 0xa9, 0xeb, 0xb7, 0x70, 0x51, 0x0b, 0x26, 0xf9,
	0x46, 0xf0, 0x86, 0x94, 0x6c, 0x98, 0xa1, 0xbc, 0x56, 0xb2, 0xf4, 0xf5, 0xd7, 0x91, 0x2d, 0xbe,
	0xb1, 0xc7, 0x9b, 0x60, 0x33, 0x77, 0xdc, 0xfb, 0x7f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x7d, 0x15,
	0x30, 0xe5, 0x5c, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// UserServiceClient is the client API for UserService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type UserServiceClient interface {
	DispatchCommand(ctx context.Context, in *DispatchCommandRequest, opts ...grpc.CallOption) (*empty.Empty, error)
	GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*User, error)
	ListUsers(ctx context.Context, in *ListUserRequest, opts ...grpc.CallOption) (*ListUserResponse, error)
}

type userServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewUserServiceClient(cc grpc.ClientConnInterface) UserServiceClient {
	return &userServiceClient{cc}
}

func (c *userServiceClient) DispatchCommand(ctx context.Context, in *DispatchCommandRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/proto.UserService/DispatchCommand", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) GetUser(ctx context.Context, in *GetUserRequest, opts ...grpc.CallOption) (*User, error) {
	out := new(User)
	err := c.cc.Invoke(ctx, "/proto.UserService/GetUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userServiceClient) ListUsers(ctx context.Context, in *ListUserRequest, opts ...grpc.CallOption) (*ListUserResponse, error) {
	out := new(ListUserResponse)
	err := c.cc.Invoke(ctx, "/proto.UserService/ListUsers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UserServiceServer is the server API for UserService service.
type UserServiceServer interface {
	DispatchCommand(context.Context, *DispatchCommandRequest) (*empty.Empty, error)
	GetUser(context.Context, *GetUserRequest) (*User, error)
	ListUsers(context.Context, *ListUserRequest) (*ListUserResponse, error)
}

// UnimplementedUserServiceServer can be embedded to have forward compatible implementations.
type UnimplementedUserServiceServer struct {
}

func (*UnimplementedUserServiceServer) DispatchCommand(ctx context.Context, req *DispatchCommandRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DispatchCommand not implemented")
}
func (*UnimplementedUserServiceServer) GetUser(ctx context.Context, req *GetUserRequest) (*User, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUser not implemented")
}
func (*UnimplementedUserServiceServer) ListUsers(ctx context.Context, req *ListUserRequest) (*ListUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUsers not implemented")
}

func RegisterUserServiceServer(s *grpc.Server, srv UserServiceServer) {
	s.RegisterService(&_UserService_serviceDesc, srv)
}

func _UserService_DispatchCommand_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DispatchCommandRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).DispatchCommand(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.UserService/DispatchCommand",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).DispatchCommand(ctx, req.(*DispatchCommandRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_GetUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).GetUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.UserService/GetUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).GetUser(ctx, req.(*GetUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserService_ListUsers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServiceServer).ListUsers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/proto.UserService/ListUsers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServiceServer).ListUsers(ctx, req.(*ListUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _UserService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "proto.UserService",
	HandlerType: (*UserServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "DispatchCommand",
			Handler:    _UserService_DispatchCommand_Handler,
		},
		{
			MethodName: "GetUser",
			Handler:    _UserService_GetUser_Handler,
		},
		{
			MethodName: "ListUsers",
			Handler:    _UserService_ListUsers_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "user.proto",
}
