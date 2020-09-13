package firewall

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

var (
	ErrInvalidRole = status.Errorf(codes.PermissionDenied, "Invalid role")
)

const mdIdentityKey = "identity"

// AppendIdentityToOutgoingUnaryContext appends identity to outgoing context
//
// https://godoc.org/google.golang.org/grpc#WithUnaryInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithUnaryInterceptor(AppendIdentityToOutgoingUnaryContext()))
func AppendIdentityToOutgoingUnaryContext() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if i, ok := identity.FromContext(ctx); ok {
			jsn, err := json.Marshal(i)
			if err != nil {
				return err
			}

			ctx = metadata.AppendToOutgoingContext(ctx, mdIdentityKey, string(jsn))
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// AppendIdentityToOutgoingStreamContext appends identity to outgoing context
//
// https://godoc.org/google.golang.org/grpc#WithStreamInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithStreamInterceptor(AppendIdentityToOutgoingStreamContext()))
func AppendIdentityToOutgoingStreamContext() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		if i, ok := identity.FromContext(ctx); ok {
			jsn, err := json.Marshal(i)
			if err != nil {
				return nil, err
			}

			ctx = metadata.AppendToOutgoingContext(ctx, mdIdentityKey, string(jsn))
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

// SetIdentityFromStreamRequest updates context with identity
//
// 	https://godoc.org/google.golang.org/grpc#StreamInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(SetIdentityFromStreamRequest()),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func SetIdentityFromStreamRequest() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if md, ok := metadata.FromIncomingContext(ss.Context()); ok {
			if values := md.Get(mdIdentityKey); len(values) > 0 {
				var i identity.Identity
				if err := json.Unmarshal([]byte(values[0]), &i); err != nil {
					return err
				}

				// TODO: update server stream context
				// ctx := identity.ContextWithIdentity(ss.Context(), i)
			}
		}

		return handler(srv, ss)
	}
}

// SetIdentityFromUnaryRequest updates context with identity
//
// 	https://godoc.org/google.golang.org/grpc#UnaryInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(SetIdentityFromUnaryRequest()),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func SetIdentityFromUnaryRequest() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(mdIdentityKey); len(values) > 0 {
				var i identity.Identity
				if err := json.Unmarshal([]byte(values[0]), &i); err != nil {
					return nil, err
				}

				ctx = identity.ContextWithIdentity(ctx, &i)
			}
		}

		return handler(ctx, req)
	}
}

// GrantAccessForStreamRequest returns error if Identity not set within context or user does not have required role
//
// 	https://godoc.org/google.golang.org/grpc#StreamInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.StreamInterceptor(GrantAccessForStreamRequest("admin")),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func GrantAccessForStreamRequest(role identity.Role) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		i, ok := identity.FromContext(ss.Context())
		if ok && i.HasRole(role) {
			return handler(srv, ss)
		}

		return apperrors.Wrap(ErrInvalidRole)
	}
}

// CheckAccessForUnaryRequest returns error if Identity not set within context or user does not have required role
//
// 	https://godoc.org/google.golang.org/grpc#UnaryInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(CheckAccessForUnaryRequest("admin")),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func GrantAccessForUnaryRequest(role identity.Role) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		i, ok := identity.FromContext(ctx)
		if ok && i.HasRole(role) {
			return handler(ctx, req)
		}

		return nil, apperrors.Wrap(ErrInvalidRole)
	}
}
