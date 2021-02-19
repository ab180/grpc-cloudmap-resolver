package cloudmap

import (
	"errors"
	"fmt"
	"net/url"

	grpcresolver "google.golang.org/grpc/resolver"
)

// BuildTarget builds grpc target string with given namespace and service.
func BuildTarget(namespace, service string) string {
	return fmt.Sprintf(
		"%s://%s/%s", Scheme, url.PathEscape(namespace), url.PathEscape(service),
	)
}

type target struct {
	namespace string
	service   string
}

func parseTarget(t grpcresolver.Target) (*target, error) {
	if t.Scheme != Scheme {
		return nil, fmt.Errorf("unexpected scheme: %s", t.Scheme)
	}

	namespace, err := url.PathUnescape(t.Authority)
	if err != nil {
		return nil, fmt.Errorf("cannot parse namespace: %v", err)
	}
	if namespace == "" {
		return nil, errors.New("namespace is required")
	}

	service, err := url.PathUnescape(t.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("cannot parse service: %v", err)
	}
	if service == "" {
		return nil, errors.New("service is required")
	}

	return &target{
		namespace: namespace,
		service:   service,
	}, nil
}
