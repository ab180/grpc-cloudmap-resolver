package cloudmap

import (
	"context"
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

type serviceDiscovery interface {
	DiscoverInstances(input *servicediscovery.DiscoverInstancesInput) (*servicediscovery.DiscoverInstancesOutput, error)
}

type resolver struct {
	logger grpclog.LoggerV2
	cc     grpcresolver.ClientConn

	sd                 serviceDiscovery
	namespace          string
	service            string
	healthStatusFilter string
	maxResults         int64

	ctx        context.Context
	cancel     context.CancelFunc
	ticker     *time.Ticker
	resolveCmd chan struct{}
	wg         sync.WaitGroup
}

func (c *resolver) ResolveNow(grpcresolver.ResolveNowOptions) {
	select {
	case c.resolveCmd <- struct{}{}:
	default:
	}
}

func (c *resolver) cloudmapLookup() (*grpcresolver.State, error) {
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
		return nil, err
	}

	addrs := make([]grpcresolver.Address, len(output.Instances))
	for i, instance := range output.Instances {
		addrs[i] = httpInstanceSummaryToAddr(instance)
	}

	return &grpcresolver.State{Addresses: addrs}, nil
}

func (c *resolver) Close() {
	c.cancel()
	c.ticker.Stop()
	c.wg.Wait()
}

func (c *resolver) watcher() {
	defer c.wg.Done()

	for {
		state, err := c.cloudmapLookup()
		if err != nil {
			c.cc.ReportError(err)
		} else {
			err = c.cc.UpdateState(*state)
		}

		if err != nil {
			c.logger.Errorln(err)
			// wait for next iteration
		}

		select {
		case <-c.ctx.Done():
			return
		case <-c.ticker.C:
		case <-c.resolveCmd:
		}
	}
}

func httpInstanceSummaryToAddr(s *servicediscovery.HttpInstanceSummary) grpcresolver.Address {
	var attrs *attributes.Attributes
	for k, v := range s.Attributes {
		attrs = attrs.WithValue(k, v)
	}

	return grpcresolver.Address{
		Addr:       fmt.Sprintf("%s:%s", *s.Attributes["AWS_INSTANCE_IPV4"], *s.Attributes["AWS_INSTANCE_PORT"]),
		Attributes: attrs,
	}
}
