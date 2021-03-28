# AWS Cloud Map Resolver for [grpc-go](https://github.com/grpc/grpc-go)

[![Go Reference](https://pkg.go.dev/badge/github.com/KimMachineGun/grpc-cloudmap-resolver.svg)](https://pkg.go.dev/github.com/KimMachineGun/grpc-cloudmap-resolver)

**grpc-cloudmap-resolver** is an implementation
of [`grpc-go.Resolver`](https://pkg.go.dev/google.golang.org/grpc/resolver#Resolver)
using [AWS Cloud Map](https://aws.amazon.com/cloud-map/).

## Installation

```shell
go get github.com/KimMachineGun/grpc-cloudmap-resolver

```

## Example

See [godoc](https://pkg.go.dev/github.com/KimMachineGun/grpc-cloudmap-resolver) for more details.

```go
package main

import (
	"log"

	"google.golang.org/grpc"

	cloudmap "github.com/KimMachineGun/grpc-cloudmap-resolver"
)

func main() {
	// register custom builder
	// cloudmap.Register(
	// 	cloudmap.WithSession(your_session),
	// 	cloudmap.WithRefreshInterval(1 * time.Minute),
	// )

	conn, err := grpc.Dial(
		cloudmap.BuildTarget("your-namespace", "your-service"),
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
