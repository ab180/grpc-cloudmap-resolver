package cloudmap

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
)

type Opt func(*builder)

func WithSession(sess *session.Session) Opt {
	return func(b *builder) {
		b.sess = sess
	}
}

func WithHealthStatusFilter(healthStatusFilter string) Opt {
	return func(b *builder) {
		b.healthStatusFilter = healthStatusFilter
	}
}

func WithMaxResults(maxResults int64) Opt {
	return func(b *builder) {
		b.maxResults = maxResults
	}
}

func WithRefreshInterval(refreshInterval time.Duration) Opt {
	return func(b *builder) {
		b.refreshInterval = refreshInterval
	}
}
