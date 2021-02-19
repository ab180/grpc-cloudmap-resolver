package cloudmap

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
)

func WithSession(sess *session.Session) func(*builder) {
	return func(b *builder) {
		b.sess = sess
	}
}

func WithHealthStatusFilter(healthStatusFilter string) func(*builder) {
	return func(b *builder) {
		b.healthStatusFilter = healthStatusFilter
	}
}

func WithMaxResults(maxResults int64) func(*builder) {
	return func(b *builder) {
		b.maxResults = maxResults
	}
}

func WithRefreshInterval(refreshInterval time.Duration) func(*builder) {
	return func(b *builder) {
		b.refreshInterval = refreshInterval
	}
}
