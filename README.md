# AWS Cloud Map Resolver for [grpc-go](https://github.com/grpc/grpc-go)

**grpc-cloudmap-resolver** is an implementation
of [`grpc-go.Resolver`](https://pkg.go.dev/google.golang.org/grpc/resolver#Resolver)
using [AWS Cloud Map](https://aws.amazon.com/cloud-map/).

## Installation

```shell
go get github.com/KimMachineGun/grpc-cloudmap-resolver

```

## Example

```go
package main

import (
	"log"
	"time"

	"google.golang.org/grpc"

	cloudmap "github.com/KimMachineGun/grpc-cloudmap-resolver"
)

func main() {
	// use custom aws session for cloudmap api
	// cloudmap.UseSession(sess)

	conn, err := grpc.Dial(
		// you can use target string directly
		// "cloudmap://grpc-servers/ab-service?refreshInterval=15s"
		cloudmap.BuildTarget(cloudmap.Config{
			Namespace:       "grpc-servers",
			Service:         "ab-service",
			RefreshInterval: 15 * time.Second,
		}),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		log.Fatal("cannot create a grpc client connection")
	}

	_ = conn
}
```
