package cloudmap

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
)

const (
	Scheme = "cloudmap"
	Target = Scheme + ":///"

	HealthStatusFilterAll       = servicediscovery.HealthStatusFilterAll
	HealthStatusFilterHealthy   = servicediscovery.HealthStatusFilterHealthy
	HealthStatusFilterUnhealthy = servicediscovery.HealthStatusFilterUnhealthy
)

type Config struct {
	// required
	Session   *session.Session
	Namespace string
	Service   string

	HealthStatusFilter string        // default: HEALTHY
	RefreshInterval    time.Duration // default: 30s
	MaxAddrs           int64         // default: 100
}

func NewBuilder(c Config) (resolver.Builder, error) {
	if c.Session == nil {
		return nil, errors.New("session is required")
	}
	if c.Namespace == "" {
		return nil, errors.New("namespace is required")
	}
	if c.Service == "" {
		return nil, errors.New("service is required")
	}
	if c.HealthStatusFilter == "" {
		c.HealthStatusFilter = HealthStatusFilterHealthy
	}
	if c.RefreshInterval == 0 {
		c.RefreshInterval = 30 * time.Second
	}
	if c.MaxAddrs == 0 {
		c.MaxAddrs = 100
	}
	return &cmBuilder{
		config: c,
	}, nil
}

type cmBuilder struct {
	config Config
}

func (c *cmBuilder) Scheme() string {
	return Scheme
}

func (c *cmBuilder) Build(_ resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	r := &cmResolver{
		mu: &sync.RWMutex{},

		logger: grpclog.Component(c.Scheme()),

		cc: cc,

		ticker: time.NewTicker(c.config.RefreshInterval),

		sd: servicediscovery.New(c.config.Session),

		healthStatusFilter: c.config.HealthStatusFilter,
		maxAddrs:           c.config.MaxAddrs,
		namespace:          c.config.Namespace,
		service:            c.config.Service,
	}

	go r.watch()

	return r, nil
}

type cmResolver struct {
	mu       *sync.RWMutex
	isClosed bool

	logger grpclog.LoggerV2

	cc resolver.ClientConn

	ticker *time.Ticker

	sd                 *servicediscovery.ServiceDiscovery
	healthStatusFilter string
	maxAddrs           int64
	namespace          string
	service            string
}

func (c *cmResolver) ResolveNow(resolver.ResolveNowOptions) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isClosed {
		return
	}

	output, err := c.sd.DiscoverInstances(&servicediscovery.DiscoverInstancesInput{
		HealthStatus:  aws.String(c.healthStatusFilter),
		MaxResults:    aws.Int64(c.maxAddrs),
		NamespaceName: aws.String(c.namespace),
		ServiceName:   aws.String(c.service),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case servicediscovery.ErrCodeServiceNotFound:
				c.logger.Errorln(servicediscovery.ErrCodeServiceNotFound, aerr.Error())
			case servicediscovery.ErrCodeNamespaceNotFound:
				c.logger.Errorln(servicediscovery.ErrCodeNamespaceNotFound, aerr.Error())
			case servicediscovery.ErrCodeInvalidInput:
				c.logger.Errorln(servicediscovery.ErrCodeInvalidInput, aerr.Error())
			case servicediscovery.ErrCodeRequestLimitExceeded:
				c.logger.Errorln(servicediscovery.ErrCodeRequestLimitExceeded, aerr.Error())
			default:
				c.logger.Errorln(aerr.Error())
			}
		} else {
			c.logger.Errorln(err.Error())
		}
		c.cc.ReportError(err)
		return
	}

	addrs := make([]resolver.Address, 0, len(output.Instances))
	for _, instance := range output.Instances {
		addrs = append(addrs, httpInstanceSummaryToAddr(instance))
	}

	c.cc.UpdateState(resolver.State{Addresses: addrs})
}

func (c *cmResolver) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return
	}

	c.isClosed = true
	c.ticker.Stop()
}

func (c *cmResolver) watch() {
	for {
		c.ResolveNow(resolver.ResolveNowOptions{})
		<-c.ticker.C
	}
}

func httpInstanceSummaryToAddr(s *servicediscovery.HttpInstanceSummary) resolver.Address {
	ip := s.Attributes["AWS_INSTANCE_IPV4"]
	port := s.Attributes["AWS_INSTANCE_PORT"]
	attrs := attributes.New()
	for k, v := range s.Attributes {
		attrs = attrs.WithValues(k, v)
	}

	return resolver.Address{
		Addr:       fmt.Sprintf("%s:%s", *ip, *port),
		Attributes: attrs,
	}
}
