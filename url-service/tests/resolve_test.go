package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestResolveURL_DBHit(t *testing.T) {

	gin.SetMode(gin.TestMode)

	db, mock, _ := sqlmock.New()

	expires := time.Now().Add(10 * time.Minute)

	rows := sqlmock.NewRows([]string{"long_url", "expires_at", "is_active"}).
		AddRow("https://google.com", expires, true)

	mock.ExpectQuery("SELECT long_url").
		WillReturnRows(rows)

	router := gin.Default()

	router.GET("/resolve/:code", func(c *gin.Context) {

		var longURL string
		var expiresAt time.Time
		var isActive bool

		err := db.QueryRow(
			`SELECT long_url, expires_at, is_active
			 FROM urls
			 WHERE short_code = ?`,
			"abc123",
		).Scan(&longURL, &expiresAt, &isActive)

		if err != nil {
			c.JSON(500, gin.H{"error": "query failed"})
			return
		}

		c.JSON(200, gin.H{"url": longURL})
	})

	req, _ := http.NewRequest("GET", "/resolve/abc123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", w.Code)
	}
}
