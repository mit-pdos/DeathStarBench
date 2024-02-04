package dialer

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/harlow/go-micro-services/tls"
	consul "github.com/hashicorp/consul/api"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// DialOption allows optional config for dialer
type DialOption func(name string) (grpc.DialOption, error)

// WithTracer traces rpc calls
func WithTracer(tracer opentracing.Tracer) DialOption {
	return func(name string) (grpc.DialOption, error) {
		return grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer)), nil
	}
}

// WithBalancer enables client side load balancing
//func WithBalancer(registry *consul.Client) DialOption {
//	return func(name string) (grpc.DialOption, error) {
//		r, err := lb.NewResolver(registry, name, "")
//		if err != nil {
//			return nil, err
//		}
//		return grpc.WithBalancer(grpc.RoundRobin(r)), nil
//	}
//}

// Dial returns a load balanced grpc client conn with tracing interceptor
func Dial(name string, registry *consul.Client, opts ...DialOption) (*grpc.ClientConn, error) {

	dialopts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Timeout:             120 * time.Hour,
			PermitWithoutStream: true,
		}),
		grpc.WithReadBufferSize(65536),
		grpc.WithWriteBufferSize(65536),
	}
	if tlsopt := tls.GetDialOpt(); tlsopt != nil {
		dialopts = append(dialopts, tlsopt)
	} else {
		dialopts = append(dialopts, grpc.WithInsecure())
	}

	for _, fn := range opts {
		opt, err := fn(name)
		if err != nil {
			return nil, fmt.Errorf("config error: %v", err)
		}
		dialopts = append(dialopts, opt)
	}

	addr := name
	if registry != nil {
		srvs, _, err := registry.Health().Service(name, "", true, &consul.QueryOptions{
			WaitIndex: 0,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to dial 1 %s: %v", name, err)
		}

		i := len(srvs) - 1
		addr = net.JoinHostPort(srvs[i].Service.Address, strconv.Itoa(srvs[i].Service.Port))
	}
	log.Printf("Dialing %v addr %v", name, addr)
	conn, err := grpc.Dial(addr, dialopts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %v", name, err)
	}
	return conn, nil
}
