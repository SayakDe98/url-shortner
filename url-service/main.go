package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"urlshortener/migration"
	pb "urlshortener/proto"
	"urlshortener/server"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx := context.Background()

	// ---- MySQL ----
	dsn := "root:@tcp(127.0.0.1:3306)/urlshortener?parseTime=true&loc=Asia%2FKolkata"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal("DB unreachable:", err)
	}

	// ---- Redis ----
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis unreachable:", err)
	}

	// ---- Migrations ----
	migration.RunMigrations(db)

	// ---- gRPC server ----
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggingInterceptor, // replaces the old gin middleware.RequestLogger()
		),
	)

	pb.RegisterURLShortenerServer(grpcServer, &server.URLShortenerServer{
		DB:  db,
		RDB: rdb,
		Ctx: ctx,
	})

	reflection.Register(grpcServer)

	log.Println("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("Server error:", err)
	}
}
