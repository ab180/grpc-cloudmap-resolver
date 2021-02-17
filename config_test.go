package cloudmap

import (
	"reflect"
	"testing"
	"time"

	grpcresolver "google.golang.org/grpc/resolver"
)

func TestBuildTarget(t *testing.T) {
	type args struct {
		config Config
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "without params",
			args: args{
				config: Config{
					Namespace: "namespace",
					Service:   "service-name",
				},
			},
			want: "cloudmap://namespace/service-name",
		},
		{
			name: "with params",
			args: args{
				config: Config{
					Namespace: "namespace",
					Service:   "service-name",

					HealthStatusFilter: HealthStatusFilterAll,
					MaxAddrs:           150,
					RefreshInterval:    50 * time.Second,
				},
			},
			want: "cloudmap://namespace/service-name?healthStatusFilter=ALL&maxAddrs=150&refreshInterval=50s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildTarget(tt.args.config); got != tt.want {
				t.Errorf("BuildTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configFromTarget(t *testing.T) {
	type args struct {
		target grpcresolver.Target
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "unexpected scheme",
			args: args{
				target: grpcresolver.Target{
					Scheme: "https",
				},
			},
			wantErr: true,
		},
		{
			name: "empty namespace",
			args: args{
				target: grpcresolver.Target{
					Scheme: Scheme,
				},
			},
			wantErr: true,
		},
		{
			name: "empty service",
			args: args{
				target: grpcresolver.Target{
					Scheme:    Scheme,
					Authority: "namespace",
				},
			},
			wantErr: true,
		},
		{
			name: "without params",
			args: args{
				target: grpcresolver.Target{
					Scheme:    Scheme,
					Authority: "namespace",
					Endpoint:  "service-name",
				},
			},
			want: &Config{
				Namespace:          "namespace",
				Service:            "service-name",
				HealthStatusFilter: HealthStatusFilterHealthy,
				MaxAddrs:           100,
				RefreshInterval:    30 * time.Second,
			},
		},
		{
			name: "with params",
			args: args{
				target: grpcresolver.Target{
					Scheme:    Scheme,
					Authority: "namespace",
					Endpoint:  "service-name?healthStatusFilter=ALL&maxAddrs=150&refreshInterval=50s",
				},
			},
			want: &Config{
				Namespace:          "namespace",
				Service:            "service-name",
				HealthStatusFilter: HealthStatusFilterAll,
				MaxAddrs:           150,
				RefreshInterval:    50 * time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configFromTarget(tt.args.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("configFromTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("configFromTarget() got = %v, want %v", got, tt.want)
			}
		})
	}
}
