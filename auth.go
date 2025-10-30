package goplugify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AuthHttpRouter struct {
	router HttpRouter
	auth   Authenticator
}

func (a *AuthHttpRouter) Add(method, path string, handler func(c HttpContext)) {
	a.router.Add(method, path, WithAuthMiddleware(handler, a.auth))
}

func WithAuthHttpRouter(router HttpRouter, auth Authenticator) HttpRouter {
	return &AuthHttpRouter{
		router: router,
		auth:   auth,
	}
}

type Authenticator interface {
	Auth(c HttpContext) error
}

func WithAuthMiddleware(handler func(c HttpContext), auth Authenticator) func(c HttpContext) {
	return func(c HttpContext) {
		if err := auth.Auth(c); err != nil {
			ErrorRet(c, fmt.Errorf("authentication failed: %v", err))
			return
		}
		handler(c)
	}
}

type NoAuth struct{}

func (a NoAuth) Auth(c HttpContext) error {
	return nil
}

type HMACAuth struct {
	AppID     string
	AppSecret string
}

func NewHMACAuth(appID, appSecret string) *HMACAuth {
	return &HMACAuth{
		AppID:     appID,
		AppSecret: appSecret,
	}
}

func (a *HMACAuth) Auth(c HttpContext) error {
	signature := c.GetHeader("X-Go-Plugify-Signature")
	if signature == "" {
		return errors.New("missing signature header")
	}

	params := GetHMACAuthSignParamsFromContext(c, a.AppSecret)
	if params.AppID != a.AppID {
		return errors.New("invalid appid")
	}

	return params.VerifySignature(signature)
}

type HMACAuthSignParams struct {
	AppID       string
	AppSecret   string
	Timestamp   string
	Nonce       string
	ContentHash string
}

func GetHMACAuthSignParamsFromContext(c HttpContext, appSecret string) HMACAuthSignParams {
	appid := c.GetHeader("X-Go-Plugify-Appid")
	timestamp := c.GetHeader("X-Go-Plugify-Timestamp")
	nonce := c.GetHeader("X-Go-Plugify-Nonce")
	ContentHash := c.GetHeader("X-Go-Plugify-Content-Hash")

	return HMACAuthSignParams{
		AppID:       appid,
		AppSecret:   appSecret,
		Timestamp:   timestamp,
		Nonce:       nonce,
		ContentHash: ContentHash,
	}
}

func (params *HMACAuthSignParams) GenerateSignature() string {
	canonical := params.buildCanonicalString()
	mac := hmac.New(sha256.New, []byte(params.AppSecret))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

func (params *HMACAuthSignParams) VerifySignature(providedSign string) error {
	ts, err := strconv.ParseInt(params.Timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid timestamp")
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return errors.New("timestamp expired")
	}

	expected := params.GenerateSignature()
	if !hmac.Equal([]byte(expected), []byte(providedSign)) {
		return errors.New("invalid signature")
	}

	return nil
}

func (params *HMACAuthSignParams) buildCanonicalString() string {
	kv := map[string]string{
		"appid":     params.AppID,
		"timestamp": params.Timestamp,
		"nonce":     params.Nonce,
	}

	if params.ContentHash != "" {
		kv["content_hash"] = params.ContentHash
	}

	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%s=%s&", k, kv[k]))
	}

	return strings.TrimRight(sb.String(), "&")
}
