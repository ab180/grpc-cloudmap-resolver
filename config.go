package cloudmap

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	grpcresolver "google.golang.org/grpc/resolver"

	"github.com/aws/aws-sdk-go/service/servicediscovery"
)

const (
	Scheme = "cloudmap"

	HealthStatusFilterAll       = servicediscovery.HealthStatusFilterAll
	HealthStatusFilterHealthy   = servicediscovery.HealthStatusFilterHealthy
	HealthStatusFilterUnhealthy = servicediscovery.HealthStatusFilterUnhealthy
)

// BuildTarget builds grpc target string with given config.
//
// Output: cloudmap://{Namespace}/{Service}?[healthStatusFilter={HealthStatusFilter}]&[maxAddrs={MaxAddrs}]&[refreshInterval={RefreshInterval}]
func BuildTarget(config Config) string {
	params := url.Values{}
	if config.HealthStatusFilter != "" {
		params.Set("healthStatusFilter", config.HealthStatusFilter)
	}
	if config.MaxAddrs != 0 {
		params.Set("maxAddrs", strconv.FormatInt(config.MaxAddrs, 10))
	}
	if config.RefreshInterval != 0 {
		params.Set("refreshInterval", config.RefreshInterval.String())
	}

	u := url.URL{
		Scheme:   Scheme,
		Host:     config.Namespace,
		Path:     config.Service,
		RawQuery: params.Encode(),
	}

	return u.String()
}

type Config struct {
	// required
	Namespace string
	Service   string

	HealthStatusFilter string        // default: HEALTHY
	MaxAddrs           int64         // default: 100
	RefreshInterval    time.Duration // default: 30s
}

func configFromTarget(target grpcresolver.Target) (*Config, error) {
	if target.Scheme != Scheme {
		return nil, fmt.Errorf("unexpected scheme: %s", target.Scheme)
	}

	targetURL, err := url.Parse(fmt.Sprintf("%s://%s/%s", target.Scheme, target.Authority, target.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("cannot construct url from target: %v", err)
	}

	c := Config{
		Namespace:          targetURL.Hostname(),
		Service:            strings.TrimPrefix(targetURL.Path, "/"),
		HealthStatusFilter: HealthStatusFilterHealthy,
		MaxAddrs:           100,
		RefreshInterval:    30 * time.Second,
	}
	if c.Namespace == "" {
		return nil, errors.New("namespace is required")
	}
	if c.Service == "" {
		return nil, errors.New("service is required")
	}

	q := targetURL.Query()
	if v := q.Get("healthStatusFilter"); v != "" {
		if v != HealthStatusFilterAll && v != HealthStatusFilterHealthy && v != HealthStatusFilterUnhealthy {
			return nil, errors.New("invalid healthStatusFilter")
		}
		c.HealthStatusFilter = v
	}
	if v := q.Get("maxAddrs"); v != "" {
		maxAddrs, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse maxAddrs: %v", err)
		}
		c.MaxAddrs = maxAddrs
	}
	if v := q.Get("refreshInterval"); v != "" {
		refreshInterval, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("cannot parse refreshInterval: %v", err)
		}
		c.RefreshInterval = refreshInterval
	}

	return &c, nil
}
