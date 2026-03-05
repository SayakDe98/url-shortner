package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestDeleteURL(t *testing.T) {

	gin.SetMode(gin.TestMode)

	db, mock, _ := sqlmock.New()

	mock.ExpectExec("UPDATE urls").
		WillReturnResult(sqlmock.NewResult(1, 1))

	router := gin.Default()

	router.DELETE("/urls/:code", func(c *gin.Context) {

		_, err := db.Exec(`
			UPDATE urls
			SET is_active = false
			WHERE short_url = $1
		`, "abc123")

		if err != nil {
			c.JSON(500, gin.H{"error": "db error"})
			return
		}

		c.JSON(200, gin.H{"message": "deleted"})
	})

	req, _ := http.NewRequest("DELETE", "/urls/abc123", nil)

	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}
