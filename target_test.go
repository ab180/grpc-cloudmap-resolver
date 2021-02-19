package cloudmap

import (
	"reflect"
	"testing"

	grpcresolver "google.golang.org/grpc/resolver"
)

func TestBuildTarget(t *testing.T) {
	type args struct {
		namespace string
		service   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "base",
			args: args{
				namespace: "test-namespace",
				service:   "test-service",
			},
			want: "cloudmap://test-namespace/test-service",
		},
		{
			name: "with slash",
			args: args{
				namespace: "test-namespace",
				service:   "test/service",
			},
			want: "cloudmap://test-namespace/test%2Fservice",
		},
		{
			name: "with whitespace",
			args: args{
				namespace: "test-namespace",
				service:   "test service",
			},
			want: "cloudmap://test-namespace/test%20service",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildTarget(tt.args.namespace, tt.args.service); got != tt.want {
				t.Errorf("BuildTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseTarget(t *testing.T) {
	type args struct {
		t grpcresolver.Target
	}
	tests := []struct {
		name    string
		args    args
		want    *target
		wantErr bool
	}{
		{
			name: "unexpected scheme",
			args: args{
				t: grpcresolver.Target{
					Scheme: "https",
				},
			},
			wantErr: true,
		},
		{
			name: "empty namespace",
			args: args{
				t: grpcresolver.Target{
					Scheme: "https",
				},
			},
			wantErr: true,
		},
		{
			name: "empty service",
			args: args{
				t: grpcresolver.Target{
					Scheme:    Scheme,
					Authority: "test-namespace",
				},
			},
			wantErr: true,
		},
		{
			name: "normal",
			args: args{
				t: grpcresolver.Target{
					Scheme:    Scheme,
					Authority: "test-namespace",
					Endpoint:  "test-service",
				},
			},
			want: &target{
				namespace: "test-namespace",
				service:   "test-service",
			},
		},
		{
			name: "with slash",
			args: args{
				t: grpcresolver.Target{
					Scheme:    Scheme,
					Authority: "test-namespace",
					Endpoint:  "test/service",
				},
			},
			want: &target{
				namespace: "test-namespace",
				service:   "test/service",
			},
		},
		{
			name: "with whitespace",
			args: args{
				t: grpcresolver.Target{
					Scheme:    Scheme,
					Authority: "test-namespace",
					Endpoint:  "test%20service",
				},
			},
			want: &target{
				namespace: "test-namespace",
				service:   "test service",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTarget(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTarget() got = %v, want %v", got, tt.want)
			}
		})
	}
}
