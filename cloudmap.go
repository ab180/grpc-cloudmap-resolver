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
)

type Config struct {
	// required
	Session   *session.Session
	Namespace string
	Service   string

	HealthStatusFilter string        // default: ALL
	RefreshInterval    time.Duration // default: 30s
	MaxAddrs           int64         // default: 10
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
		c.HealthStatusFilter = servicediscovery.HealthStatusFilterAll
	}
	if c.RefreshInterval == 0 {
		c.RefreshInterval = 30 * time.Second
	}
	if c.MaxAddrs == 0 {
		c.MaxAddrs = 10
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
		logger: grpclog.Component(c.Scheme()),

		cc: cc,

		wg:   &sync.WaitGroup{},
		stop: make(chan struct{}),
		now:  make(chan struct{}),
		tick: time.NewTicker(c.config.RefreshInterval),

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
	logger grpclog.LoggerV2

	cc resolver.ClientConn

	wg   *sync.WaitGroup
	stop chan struct{}
	now  chan struct{}
	tick *time.Ticker

	sd                 *servicediscovery.ServiceDiscovery
	healthStatusFilter string
	maxAddrs           int64
	namespace          string
	service            string
}

func (c *cmResolver) watch() {
	c.wg.Add(1)
	defer c.wg.Done()
	defer c.tick.Stop()

loop:
	for {
		select {
		case <-c.stop:
			break loop
		case <-c.tick.C:
			c.cc.UpdateState(resolver.State{Addresses: c.discover()})
		case <-c.now:
			c.cc.UpdateState(resolver.State{Addresses: c.discover()})
		}
	}
}

func (c *cmResolver) discover() []resolver.Address {
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
		return nil
	}

	addrs := make([]resolver.Address, 0, len(output.Instances))
	for _, instance := range output.Instances {
		addrs = append(addrs, httpInstanceSummaryToAddr(instance))
	}

	return addrs
}

func (c *cmResolver) ResolveNow(resolver.ResolveNowOptions) {
	select {
	case c.now <- struct{}{}:
	default:
	}
}

func (c *cmResolver) Close() {
	close(c.stop)
	c.wg.Wait()
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
