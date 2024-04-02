package http2rpc

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport"
	jwtv4 "github.com/golang-jwt/jwt/v5"
)

func Client(keyProvider jwtv4.Keyfunc, opts ...jwt.Option) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if keyProvider == nil {
				return nil, jwt.ErrNeedTokenProvider
			}
			var fromContext jwtv4.Claims
			var ok bool
			if fromContext, ok = jwt.FromContext(ctx); !ok {
				fromContext = &jwtv4.MapClaims{}
			}
			token := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, fromContext)
			key, err := keyProvider(token)
			if err != nil {
				return nil, jwt.ErrGetKey
			}
			tokenStr, err := token.SignedString(key)
			if err != nil {
				return nil, jwt.ErrSignToken
			}
			if clientContext, ok := transport.FromClientContext(ctx); ok {
				clientContext.RequestHeader().Set("Authorization", fmt.Sprintf("Bearer %s", tokenStr))
				return handler(ctx, req)
			}
			return nil, jwt.ErrWrongContext
		}
	}
}
