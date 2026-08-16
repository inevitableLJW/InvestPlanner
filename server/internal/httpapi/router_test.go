package httpapi

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestOrigin(t *testing.T) {
	tests := []struct {
		name           string
		host           string
		forwardedProto string
		want           string
	}{
		{name: "node IP with published port", host: "192.168.1.10:3000", forwardedProto: "http", want: "http://192.168.1.10:3000"},
		{name: "HTTPS reverse proxy", host: "invest.example.com", forwardedProto: "https", want: "https://invest.example.com"},
		{name: "first forwarded protocol", host: "invest.example.com", forwardedProto: "https, http", want: "https://invest.example.com"},
		{name: "reject invalid protocol", host: "invest.example.com", forwardedProto: "javascript", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest("POST", "http://api:8080/api/v1/auth/login", nil)
			ctx.Request.Host = tt.host
			if tt.forwardedProto != "" {
				ctx.Request.Header.Set("X-Forwarded-Proto", tt.forwardedProto)
			}
			if got := requestOrigin(ctx); got != tt.want {
				t.Fatalf("requestOrigin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOriginAllowed(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("POST", "http://api:8080/api/v1/auth/login", nil)
	ctx.Request.Host = "192.168.1.10:3000"
	ctx.Request.Header.Set("X-Forwarded-Proto", "http")

	if !originAllowed(ctx, "http://192.168.1.10:3000", "http://localhost:5173") {
		t.Fatal("expected the actual Node IP origin to be allowed")
	}
	if !originAllowed(ctx, "http://localhost:5173", "http://localhost:5173") {
		t.Fatal("expected the configured development origin to be allowed")
	}
	if originAllowed(ctx, "http://attacker.example", "http://localhost:5173") {
		t.Fatal("expected an unrelated origin to be rejected")
	}
}
