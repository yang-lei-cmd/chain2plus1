// Package middleware 中间件单元测试
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linqi/chain2plus1/pkg/logger"
)

func init() {
	logger.InitLogger("error")
}

func performRequest(r http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := performRequest(r, "GET", "/test")
	resp := w.Result()

	tests := []struct {
		header string
		value  string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}
	for _, tc := range tests {
		got := resp.Header.Get(tc.header)
		if got != tc.value {
			t.Errorf("header %s: expected '%s', got '%s'", tc.header, tc.value, got)
		}
	}
}

func TestRateLimit_UnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimit(5, 60*time.Second)) // 5 requests per 60 seconds
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 3 requests should all succeed
	for i := 0; i < 3; i++ {
		w := performRequest(r, "GET", "/test")
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestAuthRateLimit_BlockAfter5(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Reset the auth limiter for clean test
	authLimiter = NewRateLimiter()

	r := gin.New()
	r.POST("/login", AuthRateLimit(), func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 5 requests should succeed
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/login", nil)
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d (body: %s)", i+1, w.Code, w.Body.String())
		}
	}

	// 6th request should be blocked with 429
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/login", nil)
	r.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Errorf("6th request: expected 429, got %d", w.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := performRequest(r, "GET", "/test")
	resp := w.Result()

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS Allow-Origin: expected '*', got '%s'", resp.Header.Get("Access-Control-Allow-Origin"))
	}

	// OPTIONS preflight
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	r.ServeHTTP(w2, req)
	if w2.Code != 204 {
		t.Errorf("OPTIONS: expected 204, got %d", w2.Code)
	}
}

func TestGlobalErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(GlobalErrorHandler())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := performRequest(r, "GET", "/panic")
	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
