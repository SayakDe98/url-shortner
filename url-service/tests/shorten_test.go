package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

type CreateRequest struct {
	URL           string `json:"url"`
	ExpiryMinutes int    `json:"expiry_minutes"`
}

func TestCreateShortURL_Success(t *testing.T) {

	gin.SetMode(gin.TestMode)

	// mock DB
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock")
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO urls").
		WillReturnResult(sqlmock.NewResult(1, 1))

	router := gin.Default()

	router.POST("/shorten", func(c *gin.Context) {

		var req CreateRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		if req.ExpiryMinutes <= 0 {
			req.ExpiryMinutes = 1440
		}

		expiresAt := time.Now().Add(time.Duration(req.ExpiryMinutes) * time.Minute)

		code := "test123"

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

	body := CreateRequest{
		URL:           "https://google.com",
		ExpiryMinutes: 60,
	}

	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(
		"POST",
		"/shorten",
		bytes.NewBuffer(jsonBody),
	)

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", w.Code)
	}
}
