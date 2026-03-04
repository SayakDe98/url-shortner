package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func generateShortCode(url string) string {
	hash := sha256.Sum256([]byte(url))
	return base64.URLEncoding.EncodeToString(hash[:])[:6]
}

type CreateRequest struct {
	URL           string `json:"url" binding:"required"`
	ExpiryMinutes int    `json:"expiry_minutes"`
}

func main() {
	var err error

	dsn := "root:password@tcp(127.0.0.1:3306)/urlshortener?parseTime=true"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("DB connection error:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("DB unreachable:", err)
	}

	r := gin.Default()

	// Create short URL
	r.POST("/shorten", func(c *gin.Context) {
		var req CreateRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		// Default expiry = 24 hours
		if req.ExpiryMinutes <= 0 {
			req.ExpiryMinutes = 1440
		}

		expiresAt := time.Now().Add(time.Duration(req.ExpiryMinutes) * time.Minute)
		code := generateShortCode(req.URL)

		// Insert into DB
		_, err := db.Exec(
			"INSERT INTO urls (short_code, long_url, expires_at) VALUES (?, ?, ?)",
			code,
			req.URL,
			expiresAt,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Insert failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"short_code": code,
			"expires_at": expiresAt,
		})
	})

	// Resolve short URL
	r.GET("/resolve/:code", func(c *gin.Context) {
		code := c.Param("code")

		var longURL string
		var expiresAt time.Time

		err := db.QueryRow(
			"SELECT long_url, expires_at FROM urls WHERE short_code = ?",
			code,
		).Scan(&longURL, &expiresAt)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
			return
		}

		// Expiry check
		if time.Now().After(expiresAt) {
			c.JSON(http.StatusGone, gin.H{"error": "Link expired"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"url": longURL,
		})
	})

	r.Run(":8080")
}
