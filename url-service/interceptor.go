package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// loggingInterceptor is a unary server interceptor that logs each RPC call,
// its duration, and whether it succeeded — replacing the old HTTP
// middleware.RequestLogger() that was wired into Gin.
func loggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("[gRPC] method=%s duration=%s err=%v",
		info.FullMethod, time.Since(start), err)
	return resp, err
}
