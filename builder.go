package cloudmap

import (
	"sync"
	"time"

	"google.golang.org/grpc/grpclog"
	grpcresolver "google.golang.org/grpc/resolver"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
)

func init() {
	grpcresolver.Register(&builder{})
}

// UseSession registers new builder with given session.
func UseSession(sess *session.Session) {
	grpcresolver.Register(&builder{sess: sess})
}

type builder struct {
	sess *session.Session
}

func (b *builder) Scheme() string {
	return Scheme
}

func (b *builder) Build(t grpcresolver.Target, cc grpcresolver.ClientConn, _ grpcresolver.BuildOptions) (grpcresolver.Resolver, error) {
	c, err := configFromTarget(t)
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

	r := &resolver{
		mu: &sync.RWMutex{},

		logger: grpclog.Component(b.Scheme()),

		cc: cc,

		ticker: time.NewTicker(c.RefreshInterval),

		sd: servicediscovery.New(b.sess),

		healthStatusFilter: c.HealthStatusFilter,
		maxAddrs:           c.MaxAddrs,
		namespace:          c.Namespace,
		service:            c.Service,
	}

	go r.watch()

	return r, nil
}
