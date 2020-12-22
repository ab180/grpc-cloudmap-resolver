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

	cloudmap "github.com/KimMachineGun/grpc-cloudmap-resolver"

	"google.golang.org/grpc"

	"github.com/aws/aws-sdk-go/aws/session"
)

func main() {
	builder, err := cloudmap.NewBuilder(cloudmap.Config{
		Session:         session.Must(session.NewSession()),
		Namespace:       "grpc-servers",
		Service:         "ab-service",
		RefreshInterval: 15 * time.Second,
	})
	if err != nil {
		log.Fatal("cannot create a cloudmap resolver")
	}

	conn, err := grpc.Dial(
		cloudmap.Target,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		grpc.WithResolvers(builder),
	)
	if err != nil {
		log.Fatal("cannot create a grpc client connection")
	}

	_ = conn
}
```
