package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestValidateWorkerName(t *testing.T) {
	tests := []struct {
		name       string
		workerName string
		wantStatus int
	}{
		{"valid simple", "worker1", http.StatusOK},
		{"valid with dash", "my-worker", http.StatusOK},
		{"valid with underscore", "my_worker", http.StatusOK},
		{"too short", "ab", http.StatusBadRequest},
		{"invalid chars", "worker@123", http.StatusBadRequest},
		{"empty", "", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Params = gin.Params{{Key: "worker_name", Value: tt.workerName}}

			middleware := ValidateWorkerNameParam()
			middleware(c)

			if tt.wantStatus == http.StatusOK {
				assert.False(t, c.IsAborted())
			} else {
				assert.True(t, c.IsAborted())
				assert.Equal(t, tt.wantStatus, w.Code)
			}
		})
	}
}

func TestValidateTaskID(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		wantStatus int
	}{
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", http.StatusOK},
		{"valid short", "task123", http.StatusOK},
		{"too long", strings.Repeat("a", 129), http.StatusBadRequest},
		{"empty", "", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Params = gin.Params{{Key: "task_id", Value: tt.taskID}}

			middleware := ValidateTaskIDParam()
			middleware(c)

			if tt.wantStatus == http.StatusOK {
				assert.False(t, c.IsAborted())
			} else {
				assert.True(t, c.IsAborted())
				assert.Equal(t, tt.wantStatus, w.Code)
			}
		})
	}
}

func TestPayloadSizeLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 测试正常大小的请求
	t.Run("normal size", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(PayloadSizeLimit(1024))
		r.POST("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		body := bytes.NewBufferString("test")
		c.Request = httptest.NewRequest("POST", "/test", body)
		r.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 测试超大请求
	t.Run("oversized", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(PayloadSizeLimit(10))
		r.POST("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		body := bytes.NewBufferString(strings.Repeat("a", 20))
		c.Request = httptest.NewRequest("POST", "/test", body)
		c.Request.ContentLength = 20
		r.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("generate request id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(RequestIDMiddleware())
		r.GET("/test", func(c *gin.Context) {
			requestID, exists := c.Get("request_id")
			assert.True(t, exists)
			assert.NotEmpty(t, requestID)
			c.String(http.StatusOK, "ok")
		})

		c.Request = httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	})

	t.Run("use existing request id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)
		r.Use(RequestIDMiddleware())
		r.GET("/test", func(c *gin.Context) {
			requestID, _ := c.Get("request_id")
			assert.Equal(t, "test-123", requestID)
			c.String(http.StatusOK, "ok")
		})

		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-Request-ID", "test-123")
		r.ServeHTTP(w, c.Request)

		assert.Equal(t, "test-123", w.Header().Get("X-Request-ID"))
	})
}
