package cloudmap

import (
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/grpclog"
	grpcresolver "google.golang.org/grpc/resolver"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
)

type resolver struct {
	mu       *sync.RWMutex
	isClosed bool

	logger grpclog.LoggerV2

	cc grpcresolver.ClientConn

	ticker *time.Ticker

	sd                 *servicediscovery.ServiceDiscovery
	namespace          string
	service            string
	healthStatusFilter string
	maxResults         int64
}

func (c *resolver) ResolveNow(grpcresolver.ResolveNowOptions) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.isClosed {
		return
	}

	output, err := c.sd.DiscoverInstances(&servicediscovery.DiscoverInstancesInput{
		NamespaceName: aws.String(c.namespace),
		ServiceName:   aws.String(c.service),
		HealthStatus:  aws.String(c.healthStatusFilter),
		MaxResults:    aws.Int64(c.maxResults),
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

	addrs := make([]grpcresolver.Address, 0, len(output.Instances))
	for _, instance := range output.Instances {
		addrs = append(addrs, httpInstanceSummaryToAddr(instance))
	}

	c.cc.UpdateState(grpcresolver.State{Addresses: addrs})
}

func (c *resolver) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return
	}

	c.isClosed = true
	c.ticker.Stop()
}

func (c *resolver) watch() {
	for {
		c.ResolveNow(grpcresolver.ResolveNowOptions{})
		<-c.ticker.C
	}
}

func httpInstanceSummaryToAddr(s *servicediscovery.HttpInstanceSummary) grpcresolver.Address {
	ip := s.Attributes["AWS_INSTANCE_IPV4"]
	port := s.Attributes["AWS_INSTANCE_PORT"]
	attrs := attributes.New()
	for k, v := range s.Attributes {
		attrs = attrs.WithValues(k, v)
	}

	return grpcresolver.Address{
		Addr:       fmt.Sprintf("%s:%s", *ip, *port),
		Attributes: attrs,
	}
}
