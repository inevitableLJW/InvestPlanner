package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"investplanner/server/internal/repository"
)

type authRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (a *API) register(c *gin.Context) {
	var request authRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "请填写有效用户名和密码", nil)
		return
	}
	result, err := a.app.Register(request.Username, request.Password)
	if err != nil {
		if err == repository.ErrConflict {
			respondError(c, http.StatusConflict, "username_exists", "该用户名已被使用", nil)
			return
		}
		handleError(c, err)
		return
	}
	a.setSessionCookie(c, result.Token)
	c.JSON(http.StatusCreated, userResponse(result.User.ID, result.User.Username))
}

func (a *API) login(c *gin.Context) {
	var request authRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", "请填写用户名和密码", nil)
		return
	}
	result, err := a.app.Login(request.Username, request.Password)
	if err != nil {
		handleError(c, err)
		return
	}
	a.setSessionCookie(c, result.Token)
	c.JSON(http.StatusOK, userResponse(result.User.ID, result.User.Username))
}

func (a *API) me(c *gin.Context) {
	username, _ := c.Get("current_username")
	c.JSON(http.StatusOK, userResponse(currentUserID(c), username.(string)))
}

func (a *API) logout(c *gin.Context) {
	token, _ := c.Cookie(sessionCookie)
	if err := a.app.Logout(token); err != nil {
		handleError(c, err)
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, "", -1, "/", "", a.cfg.CookieSecure, true)
	c.Status(http.StatusNoContent)
}

func (a *API) setSessionCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteLaxMode)
	maxAge := int(a.cfg.SessionTTL / time.Second)
	c.SetCookie(sessionCookie, token, maxAge, "/", "", a.cfg.CookieSecure, true)
}

func userResponse(id, username string) gin.H {
	return gin.H{"id": id, "username": username}
}
