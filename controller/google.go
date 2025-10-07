package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"one-api/common"
	"one-api/model"
	"one-api/setting/system_setting"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type GoogleOAuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	IdToken     string `json:"id_token"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
	ErrorURI    string `json:"error_uri"`
}

type GoogleUser struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

func getGoogleRedirectURI(c *gin.Context) string {
	base := strings.TrimSuffix(system_setting.ServerAddress, "/")
	if base == "" {
		scheme := "http"
		if forwardedProto := c.Request.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
			parts := strings.Split(forwardedProto, ",")
			if len(parts) > 0 {
				scheme = strings.TrimSpace(parts[0])
			}
		} else if c.Request.TLS != nil {
			scheme = "https"
		}
		base = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	}
	return base + "/oauth/google"
}

func getGoogleUserInfoByCode(c *gin.Context, code string) (*GoogleUser, error) {
	if code == "" {
		return nil, errors.New("无效的参数")
	}

	redirectURI := getGoogleRedirectURI(c)

	values := url.Values{}
	values.Set("code", code)
	values.Set("client_id", common.GoogleClientId)
	values.Set("client_secret", common.GoogleClientSecret)
	values.Set("redirect_uri", redirectURI)
	values.Set("grant_type", "authorization_code")

	req, err := http.NewRequest(
		"POST",
		"https://oauth2.googleapis.com/token",
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		common.SysLog(err.Error())
		return nil, errors.New("无法连接至 Google 服务器，请稍后重试！")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token endpoint returned status %d", res.StatusCode)
	}
	var tokenResponse GoogleOAuthResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenResponse); err != nil {
		return nil, err
	}

	if tokenResponse.AccessToken == "" {
		errMsg := tokenResponse.ErrorDesc
		if errMsg == "" {
			errMsg = tokenResponse.Error
		}
		if errMsg == "" {
			errMsg = "请检查配置"
		}
		if tokenResponse.ErrorURI != "" {
			errMsg = errMsg + " (" + tokenResponse.ErrorURI + ")"
		}
		return nil, errors.New("获取 Google 访问令牌失败：" + errMsg)
	}

	req, err = http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenResponse.AccessToken))
	req.Header.Set("Accept", "application/json")

	res2, err := client.Do(req)
	if err != nil {
		common.SysLog(err.Error())
		return nil, errors.New("无法连接至 Google 服务器，请稍后重试！")
	}
	defer res2.Body.Close()

	var googleUser GoogleUser
	if err := json.NewDecoder(res2.Body).Decode(&googleUser); err != nil {
		return nil, err
	}

	if googleUser.Sub == "" {
		return nil, errors.New("返回值非法，用户字段为空，请稍后重试！")
	}
	if googleUser.Email != "" && !googleUser.EmailVerified {
		return nil, errors.New("google 账户邮箱未验证,请先验证邮箱")
	}

	return &googleUser, nil
}

func GoogleOAuth(c *gin.Context) {
	session := sessions.Default(c)
	state := c.Query("state")
	savedState, ok := session.Get("oauth_state").(string)
	if state == "" || !ok || state != savedState {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "state is empty or not same",
		})
		return
	}

	if session.Get("username") != nil {
		GoogleBind(c)
		return
	}

	if !common.GoogleOAuthEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未开启通过 Google 登录以及注册",
		})
		return
	}

	code := c.Query("code")
	googleUser, err := getGoogleUserInfoByCode(c, code)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	user := model.User{GoogleId: googleUser.Sub}

	if model.IsGoogleIdAlreadyTaken(user.GoogleId) {
		if err := user.FillUserByGoogleId(); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		if googleUser.Email != "" && user.GoogleEmail != googleUser.Email {
			user.GoogleEmail = googleUser.Email
			_ = user.Update(false)
		}
		if user.Id == 0 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "用户已注销",
			})
			return
		}
	} else {
		if !common.RegisterEnabled {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "管理员关闭了新用户注册",
			})
			return
		}

		// Generate username from email prefix, fallback to random string
		username := "google_user"
		if googleUser.Email != "" {
			emailPrefix := strings.Split(googleUser.Email, "@")[0]
			username = emailPrefix
		}

		// Check for username conflicts and append suffix if needed
		originalUsername := username
		suffix := 1
		for model.IsUsernameAlreadyTaken(username) {
			username = fmt.Sprintf("%s_%d", originalUsername, suffix)
			suffix++
		}
		user.Username = username

		if googleUser.Name != "" {
			user.DisplayName = googleUser.Name
		} else {
			user.DisplayName = "Google User"
		}
		user.Email = googleUser.Email
		user.GoogleEmail = googleUser.Email
		user.Role = common.RoleCommonUser
		user.Status = common.UserStatusEnabled

		inviterId := 0
		if aff := session.Get("aff"); aff != nil {
			if affCode, ok := aff.(string); ok {
				inviterId, _ = model.GetUserIdByAffCode(affCode)
			}
		}

		if err := user.Insert(inviterId); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	if user.Status != common.UserStatusEnabled {
		c.JSON(http.StatusOK, gin.H{
			"message": "用户已被封禁",
			"success": false,
		})
		return
	}

	setupLogin(&user, c)
}

func GoogleBind(c *gin.Context) {
	if !common.GoogleOAuthEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "管理员未开启通过 Google 登录以及注册",
		})
		return
	}

	code := c.Query("code")
	googleUser, err := getGoogleUserInfoByCode(c, code)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	user := model.User{GoogleId: googleUser.Sub}
	if model.IsGoogleIdAlreadyTaken(user.GoogleId) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该 Google 账户已被绑定",
		})
		return
	}

	session := sessions.Default(c)
	id, ok := session.Get("id").(int)
	if !ok || id == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未登录或会话已过期",
		})
		return
	}
	user.Id = id

	if err := user.FillUserById(); err != nil {
		common.ApiError(c, err)
		return
	}

	user.GoogleId = googleUser.Sub
	if googleUser.Email != "" {
		user.GoogleEmail = googleUser.Email
	}
	if err := user.Update(false); err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "bind",
	})
}
