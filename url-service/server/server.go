package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"time"

	pb "urlshortener/proto"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CachedURL mirrors the shape stored as JSON in Redis.
type CachedURL struct {
	LongURL   string    `json:"long_url"`
	IsActive  bool      `json:"is_active"`
	ExpiresAt time.Time `json:"expires_at"`
}

// URLShortenerServer implements proto.URLShortenerServer.
type URLShortenerServer struct {
	pb.UnimplementedURLShortenerServer
	DB  *sql.DB
	RDB *redis.Client
	Ctx context.Context
}

// generateShortCode produces a 6-character URL-safe hash of the input URL.
func generateShortCode(url string) string {
	hash := sha256.Sum256([]byte(url))
	return base64.URLEncoding.EncodeToString(hash[:])[:6]
}

// ShortenURL handles the creation of a new short URL.
//
// Flow:
//  1. Check Redis — if a valid (active, non-expired) entry exists, reuse it.
//  2. If found but inactive/expired, reactivate + extend expiry in DB and cache.
//  3. If not found at all, insert into MySQL and write through to Redis.
func (s *URLShortenerServer) ShortenURL(
	ctx context.Context,
	req *pb.ShortenURLRequest,
) (*pb.ShortenURLResponse, error) {

	if req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}

	expiryMinutes := int(req.ExpiryMinutes)
	if expiryMinutes <= 0 {
		expiryMinutes = 1440
	}

	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		loc = time.FixedZone("IST", 5*60*60+30*60)
	}
	expiresAt := time.Now().In(loc).Add(time.Duration(expiryMinutes) * time.Minute)
	code := generateShortCode(req.Url)

	// 1. Check Redis for an existing entry.
	val, redisErr := s.RDB.Get(ctx, code).Result()
	if redisErr == nil {
		var cached CachedURL
		if jsonErr := json.Unmarshal([]byte(val), &cached); jsonErr == nil {
			if cached.IsActive && time.Now().Before(cached.ExpiresAt) {
				// Valid entry already exists — return it as-is.
				return &pb.ShortenURLResponse{
					ShortCode: code,
					ExpiresAt: timestamppb.New(cached.ExpiresAt),
				}, nil
			}
		}
	}

	// 2. Upsert into MySQL — reactivate if soft-deleted or expired; insert otherwise.
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO urls (short_code, long_url, expires_at, is_active)
		VALUES (?, ?, ?, true)
		ON DUPLICATE KEY UPDATE
			is_active  = true,
			expires_at = VALUES(expires_at)
	`, code, req.Url, expiresAt)
	if err != nil {
		log.Println("DB upsert error:", err)
		return nil, status.Error(codes.Internal, "failed to persist URL")
	}

	// 3. Write-through cache.
	cached := CachedURL{
		LongURL:   req.Url,
		IsActive:  true,
		ExpiresAt: expiresAt,
	}
	payload, _ := json.Marshal(cached)
	if setErr := s.RDB.Set(
		ctx,
		code,
		payload,
		time.Duration(expiryMinutes)*time.Minute,
	).Err(); setErr != nil {
		log.Println("Redis write failed:", setErr)
	}

	return &pb.ShortenURLResponse{
		ShortCode: code,
		ExpiresAt: timestamppb.New(expiresAt),
	}, nil
}

// ResolveURL looks up the long URL for a given short code.
//
// Flow:
//  1. Try Redis — honour is_active and ExpiresAt from the cached JSON payload.
//  2. Fallback to MySQL — re-populate Redis with the remaining TTL on cache miss.
func (s *URLShortenerServer) ResolveURL(
	ctx context.Context,
	req *pb.ResolveURLRequest,
) (*pb.ResolveURLResponse, error) {

	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	// 1. Redis lookup.
	val, err := s.RDB.Get(ctx, req.Code).Result()
	if err == nil {
		var cached CachedURL
		if jsonErr := json.Unmarshal([]byte(val), &cached); jsonErr == nil {
			if !cached.IsActive {
				return nil, status.Error(codes.NotFound, "URL has been deleted")
			}
			if time.Now().After(cached.ExpiresAt) {
				return nil, status.Error(codes.NotFound, "link has expired")
			}
			return &pb.ResolveURLResponse{Url: cached.LongURL}, nil
		}
	}

	// 2. MySQL fallback.
	var longURL string
	var expiresAt time.Time
	var isActive bool

	err = s.DB.QueryRowContext(ctx,
		"SELECT long_url, expires_at, is_active FROM urls WHERE short_code = ?",
		req.Code,
	).Scan(&longURL, &expiresAt, &isActive)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "short code not found")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "database query failed")
	}
	if !isActive {
		return nil, status.Error(codes.NotFound, "URL has been deleted")
	}
	if time.Now().After(expiresAt) {
		return nil, status.Error(codes.NotFound, "link has expired")
	}

	// 3. Re-populate Redis with remaining TTL.
	ttl := time.Until(expiresAt)
	if ttl > 0 {
		cached := CachedURL{LongURL: longURL, IsActive: true, ExpiresAt: expiresAt}
		payload, _ := json.Marshal(cached)
		if setErr := s.RDB.Set(ctx, req.Code, payload, ttl).Err(); setErr != nil {
			log.Println("Redis repopulate failed:", setErr)
		}
	}

	return &pb.ResolveURLResponse{Url: longURL}, nil
}

// DeleteURL soft-deletes a short URL and evicts it from Redis.
func (s *URLShortenerServer) DeleteURL(
	ctx context.Context,
	req *pb.DeleteURLRequest,
) (*pb.DeleteURLResponse, error) {

	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	result, err := s.DB.ExecContext(ctx, `
		UPDATE urls
		SET is_active = false
		WHERE short_code = ?
	`, req.Code)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error while deleting URL")
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, status.Error(codes.NotFound, "short code not found")
	}

	// Evict from cache.
	if delErr := s.RDB.Del(ctx, req.Code).Err(); delErr != nil {
		log.Println("Redis eviction failed:", delErr)
	}

	return &pb.DeleteURLResponse{Message: "URL deleted successfully"}, nil
}
