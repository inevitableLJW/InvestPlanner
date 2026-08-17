package httpapi

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"investplanner/server/internal/config"
	"investplanner/server/internal/repository"
	"investplanner/server/internal/service"
)

const sessionCookie = "invest_planner_session"
const userContextKey = "current_user_id"

type API struct {
	cfg config.Config
	app *service.App
}

func New(cfg config.Config, app *service.App) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	api := &API{cfg: cfg, app: app}
	router := gin.New()
	router.Use(gin.Recovery(), requestLogger())
	router.Use(cors.New(cors.Config{
		AllowOriginWithContextFunc: func(c *gin.Context, origin string) bool {
			return originAllowed(c, origin, cfg.WebOrigin)
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Accept"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	v1 := router.Group("/api/v1")
	v1.Use(api.originGuard())
	authLimiter := newWindowLimiter(10, time.Minute)
	v1.POST("/auth/register", authLimiter.middleware(), api.register)
	v1.POST("/auth/login", authLimiter.middleware(), api.login)

	protected := v1.Group("")
	protected.Use(api.authenticate())
	protected.GET("/auth/me", api.me)
	protected.POST("/auth/logout", api.logout)

	protected.GET("/expense-sources", api.listSources)
	protected.POST("/expense-sources", api.saveSource)
	protected.PUT("/expense-sources/:sourceID", api.saveSource)

	protected.GET("/plans", api.listPlans)
	protected.POST("/plans", api.createPlan)
	protected.GET("/plans/:planID", api.getPlan)
	protected.PUT("/plans/:planID", api.updatePlan)
	protected.DELETE("/plans/:planID", api.archivePlan)
	protected.DELETE("/plans/:planID/draft", api.deleteDraftPlan)

	protected.GET("/plans/:planID/months", api.listMonths)
	protected.GET("/plans/:planID/months/previous-sources", api.previousSources)
	protected.GET("/plans/:planID/months/:month", api.getMonth)
	protected.PUT("/plans/:planID/months/:month", api.upsertMonth)
	protected.DELETE("/plans/:planID/months/:month", api.deleteMonth)
	protected.PUT("/plans/:planID/months/:month/actuals", api.updateActuals)
	protected.GET("/plans/:planID/stats", api.planStats)
	return router
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		gin.DefaultWriter.Write([]byte(c.Request.Method + " " + c.FullPath() + " " + time.Since(start).String() + "\n"))
	}
}

func (a *API) originGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		origin := c.GetHeader("Origin")
		if origin != "" && !originAllowed(c, origin, a.cfg.WebOrigin) {
			respondError(c, http.StatusForbidden, "origin_forbidden", "请求来源不被允许", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func originAllowed(c *gin.Context, origin, configuredOrigin string) bool {
	return origin == configuredOrigin || origin == requestOrigin(c)
}

func requestOrigin(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]); forwarded != "" {
		scheme = forwarded
	}
	if scheme != "http" && scheme != "https" {
		return ""
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		return ""
	}
	return (&url.URL{Scheme: scheme, Host: host}).String()
}

func (a *API) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(sessionCookie)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "unauthorized", "请先登录", nil)
			c.Abort()
			return
		}
		user, err := a.app.Authenticate(token)
		if err != nil {
			respondError(c, http.StatusUnauthorized, "unauthorized", "登录已失效，请重新登录", nil)
			c.Abort()
			return
		}
		c.Set(userContextKey, user.ID)
		c.Set("current_username", user.Username)
		c.Next()
	}
}

func currentUserID(c *gin.Context) string {
	value, _ := c.Get(userContextKey)
	id, _ := value.(string)
	return id
}

func respondError(c *gin.Context, status int, code, message string, details any) {
	payload := gin.H{"error": gin.H{"code": code, "message": message}}
	if details != nil {
		payload["error"].(gin.H)["details"] = details
	}
	c.JSON(status, payload)
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		respondError(c, http.StatusNotFound, "not_found", "资源不存在", nil)
	case errors.Is(err, repository.ErrConflict):
		respondError(c, http.StatusConflict, "conflict", "数据已被修改，请刷新后重试", nil)
	case errors.Is(err, service.ErrValidation):
		respondError(c, http.StatusBadRequest, "validation_error", strings.TrimPrefix(err.Error(), service.ErrValidation.Error()+": "), nil)
	case errors.Is(err, service.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized, "invalid_credentials", "用户名或密码错误", nil)
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "服务暂时不可用，请稍后重试", nil)
	}
}

type windowLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]*windowCounter
}

type windowCounter struct {
	count int
	reset time.Time
}

func newWindowLimiter(limit int, window time.Duration) *windowLimiter {
	return &windowLimiter{limit: limit, window: window, entries: map[string]*windowCounter{}}
}

func (l *windowLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()
		l.mu.Lock()
		entry := l.entries[key]
		if entry == nil || now.After(entry.reset) {
			entry = &windowCounter{reset: now.Add(l.window)}
			l.entries[key] = entry
		}
		entry.count++
		allowed := entry.count <= l.limit
		l.mu.Unlock()
		if !allowed {
			respondError(c, http.StatusTooManyRequests, "rate_limited", "请求过于频繁，请稍后再试", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
