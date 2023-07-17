package cloudmap

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"google.golang.org/grpc/grpclog"
	grpcresolver "google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
	"testing"
	"time"
)

type mockCC struct{}

func (m mockCC) UpdateState(state grpcresolver.State) error { return nil }

func (m mockCC) ReportError(err error) {}

func (m mockCC) NewAddress(addresses []grpcresolver.Address) {}

func (m mockCC) NewServiceConfig(serviceConfig string) {}

func (m mockCC) ParseServiceConfig(serviceConfigJSON string) *serviceconfig.ParseResult {
	return nil
}

type mockDiscovery struct{}

func (m mockDiscovery) DiscoverInstances(input *servicediscovery.DiscoverInstancesInput) (*servicediscovery.DiscoverInstancesOutput, error) {
	time.Sleep(1 * time.Second)
	fmt.Println("DiscoverInstances called")
	return &servicediscovery.DiscoverInstancesOutput{
		Instances: make([]*servicediscovery.HttpInstanceSummary, 0),
	}, nil
}

func Test_resolver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := &resolver{
		logger: grpclog.Component("test"),

		cc: mockCC{},
		sd: mockDiscovery{},

		ctx:        ctx,
		cancel:     cancel,
		ticker:     time.NewTicker(10 * time.Second),
		resolveCmd: make(chan struct{}, 1),
	}

	r.wg.Add(1)
	go r.watcher()

	timeout := time.After(100 * time.Millisecond)
	done := make(chan bool)
	go func() {
		for i := 0; i < 10; i++ {
			r.ResolveNow(grpcresolver.ResolveNowOptions{})
		}
		done <- true
	}()
	select {
	case <-timeout:
		t.Error("timeout")
	case <-done:
		t.Log("done")
	}
	r.Close()
}
