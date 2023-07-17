package cloudmap

import (
	"context"
	"time"

	"google.golang.org/grpc/grpclog"
	grpcresolver "google.golang.org/grpc/resolver"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
)

const (
	Scheme = "cloudmap"

	HealthStatusFilterAll       = servicediscovery.HealthStatusFilterAll
	HealthStatusFilterHealthy   = servicediscovery.HealthStatusFilterHealthy
	HealthStatusFilterUnhealthy = servicediscovery.HealthStatusFilterUnhealthy
)

func init() {
	Register()
}

type builder struct {
	sess               *session.Session // default: session.NewSession()
	healthStatusFilter string           // default: HEALTHY
	maxResults         int64            // default: 100
	refreshInterval    time.Duration    // default: 30s
}

// Register builds builder with given opts and register it to the resolver map.
// If you don't give any options, the builder will be registered with default options listed below.
//
// The default builder was already registered by the init function,
// so you don't need to call this function to register the default builder.
//
// Default Options:
//
//	Session: session.NewSession()
//	HealthStatusFilter: HealthStatusFilterHealthy
//	MaxResults: 100
//	RefreshInterval: 30s
func Register(opts ...Opt) {
	b := &builder{
		healthStatusFilter: HealthStatusFilterHealthy,
		maxResults:         100,
		refreshInterval:    30 * time.Second,
	}
	for _, opt := range opts {
		opt(b)
	}
	grpcresolver.Register(b)
}

func (b *builder) Scheme() string {
	return Scheme
}

func (b *builder) Build(t grpcresolver.Target, cc grpcresolver.ClientConn, _ grpcresolver.BuildOptions) (grpcresolver.Resolver, error) {
	cmT, err := parseTarget(t)
	if err != nil {
		return nil, err
	}

	sess := b.sess
	if sess == nil {
		sess, err = session.NewSession()
		if err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &resolver{
		logger: grpclog.Component(b.Scheme()),

		cc: cc,

		sd:                 servicediscovery.New(sess),
		namespace:          cmT.namespace,
		service:            cmT.service,
		healthStatusFilter: b.healthStatusFilter,
		maxResults:         b.maxResults,

		ctx:        ctx,
		cancel:     cancel,
		ticker:     time.NewTicker(b.refreshInterval),
		resolveCmd: make(chan struct{}, 1),
	}

	r.wg.Add(1)
	go r.watcher()

	return r, nil
}
