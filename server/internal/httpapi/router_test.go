package httpapi

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"investplanner/server/internal/config"
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

func TestSessionCookieSecureMode(t *testing.T) {
	tests := []struct {
		name           string
		mode           config.CookieSecureMode
		forwardedProto string
		directTLS      bool
		wantSecure     bool
	}{
		{name: "auto over HTTP", mode: config.CookieSecureAuto, forwardedProto: "http", wantSecure: false},
		{name: "auto over HTTPS proxy", mode: config.CookieSecureAuto, forwardedProto: "https", wantSecure: true},
		{name: "auto over direct TLS", mode: config.CookieSecureAuto, directTLS: true, wantSecure: true},
		{name: "auto accepts uppercase forwarded HTTPS", mode: config.CookieSecureAuto, forwardedProto: "HTTPS", wantSecure: true},
		{name: "forced secure over HTTP", mode: config.CookieSecureAlways, forwardedProto: "http", wantSecure: true},
		{name: "forced insecure over HTTPS", mode: config.CookieSecureNever, forwardedProto: "https", wantSecure: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "http://api:8080/api/v1/auth/login", nil)
			ctx.Request.Header.Set("X-Forwarded-Proto", tt.forwardedProto)
			if tt.directTLS {
				ctx.Request.TLS = &tls.ConnectionState{}
			}
			api := &API{cfg: config.Config{CookieSecureMode: tt.mode, SessionTTL: time.Hour}}

			api.setSessionCookie(ctx, "test-token")
			cookies := recorder.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("cookies = %d, want 1", len(cookies))
			}
			if cookies[0].Secure != tt.wantSecure || cookies[0].MaxAge != 3600 || cookies[0].SameSite != http.SameSiteLaxMode || !cookies[0].HttpOnly {
				t.Fatalf("unexpected cookie: %#v", cookies[0])
			}
		})
	}
}
