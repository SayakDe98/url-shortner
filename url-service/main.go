package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"urlshortener/middleware"
	"urlshortener/migration"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

var (
	db  *sql.DB
	rdb *redis.Client
	ctx = context.Background()
)

func generateShortCode(url string) string {
	hash := sha256.Sum256([]byte(url))
	return base64.URLEncoding.EncodeToString(hash[:])[:6]
}

type CreateRequest struct {
	URL           string `json:"url" binding:"required"`
	ExpiryMinutes int    `json:"expiry_minutes"`
}

type CachedURL struct {
	LongURL   string    `json:"long_url"`
	IsActive  bool      `json:"is_active"`
	ExpiresAt time.Time `json:"expires_at"`
}

func main() {
	var err error

	// ---- MySQL ----
	dsn := "root:@tcp(127.0.0.1:3306)/urlshortener?parseTime=true&loc=Asia%2FKolkata"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("DB unreachable:", err)
	}

	// ---- Redis ----
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis unreachable:", err)
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.Use(middleware.RequestLogger()) // added logger middleware

	// Create short URL
	r.POST("/shorten", func(c *gin.Context) {
		var req CreateRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}
		if req.ExpiryMinutes <= 0 {
			req.ExpiryMinutes = 1440
		}
		loc, err := time.LoadLocation("Asia/Kolkata")
		if err != nil {
			// fallback or hard fail
			loc = time.FixedZone("IST", 5*60*60+30*60) // UTC+5:30 manually
		}
		expiresAt := time.Now().In(loc).Add(time.Duration(req.ExpiryMinutes) * time.Minute)
		code := generateShortCode(req.URL)

		// Insert into MySQL
		_, err = db.Exec(
			"INSERT INTO urls (short_code, long_url, expires_at) VALUES (?, ?, ?)",
			code,
			req.URL,
			expiresAt,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Insert failed"})
			return
		}
		// Write-through cache (TTL = expiry)
		err = rdb.Set(
			ctx,
			code,
			req.URL,
			time.Duration(req.ExpiryMinutes)*time.Minute,
		).Err()
		if err != nil {
			log.Println("Redis write failed:", err)
		}
		c.JSON(http.StatusOK, gin.H{
			"short_code": code,
			"expires_at": expiresAt,
		})
	})

	// Resolve short URL
	r.GET("/resolve/:code", func(c *gin.Context) {
		code := c.Param("code")

		// 1️⃣ Try Redis first
		val, err := rdb.Get(ctx, code).Result()
		if err == nil {
			var cached CachedURL
			err := json.Unmarshal([]byte(val), &cached)
			if err == nil {

				if !cached.IsActive {
					c.JSON(http.StatusGone, gin.H{"error": "Url deleted"})
					return
				}

				if time.Now().After(cached.ExpiresAt) {
					c.JSON(http.StatusGone, gin.H{"error": "Link expired"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"url": cached.LongURL})
				return
			}
		}

		// 2️⃣ Fallback to MySQL
		var longURL string
		var expiresAt time.Time
		var isActive bool

		err = db.QueryRow(
			"SELECT long_url, expires_at, is_active FROM urls WHERE short_code = ?",
			code,
		).Scan(&longURL, &expiresAt, &isActive)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
			return
		}

		if !isActive {
			c.JSON(http.StatusGone, gin.H{"error": "Url deleted"})
			return
		}
		// Expiry check
		if time.Now().After(expiresAt) {
			c.JSON(http.StatusGone, gin.H{"error": "Link expired"})
			return
		}

		// 3️⃣ Re-populate Redis with remaining TTL
		ttl := time.Until(expiresAt)
		if ttl > 0 {
			rdb.Set(ctx, code, longURL, ttl)
		}

		c.JSON(http.StatusOK, gin.H{"url": longURL})
	})

	// soft delete url
	r.DELETE("/urls/:code", func(c *gin.Context) {
		code := c.Param("code")
		// delete from database
		result, err := db.Exec(`
			UPDATE urls
			SET is_active = false
			WHERE short_code = ?
		`, code)

		if err != nil {
			c.JSON(500, gin.H{"error": "Database error for removing url"})
			return
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(404, gin.H{"error": "Url not found"})
			return
		}

		// remove from cache
		err = rdb.Del(ctx, code).Err()

		if err != nil {
			log.Println("Failed to clear cache")
		}

		c.JSON(http.StatusOK, gin.H{"message": "Url deleted successfully"})
	})
	migration.RunMigrations(db)
	r.Run(":8080")
}
