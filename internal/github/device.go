// Package github implements just enough of GitHub's OAuth Device Authorization
// Flow (RFC 8628) to identify a user from a TUI/SSH session, plus a single
// /user call to resolve the access token to a login.
//
// The flow:
//
//  1. Start posts to /login/device/code, returning a user_code (shown to the
//     user) + device_code (server-side handle) + poll interval + expiry.
//  2. Poll posts to /login/oauth/access_token until GitHub returns either a
//     token or a terminal error. The "authorization_pending" and "slow_down"
//     responses are handled internally.
//  3. UserLogin GETs /user with the token and returns the "login" field.
//
// Tokens are never persisted; we discard them after step 3.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	deviceCodeURL  = "https://github.com/login/device/code"
	tokenURL       = "https://github.com/login/oauth/access_token"
	userURL        = "https://api.github.com/user"
	deviceGrantTyp = "urn:ietf:params:oauth:grant-type:device_code"

	defaultPollInterval = 5 * time.Second
	maxPollInterval     = 30 * time.Second
)

// DeviceFlow is a configured client. ClientID is the public OAuth app id;
// HTTP defaults to http.DefaultClient if nil.
type DeviceFlow struct {
	ClientID string
	Scope    string // optional; empty is fine for read-only login
	HTTP     *http.Client
}

// StartResponse holds the user-visible bits the lobby renders plus the
// internal device_code the poll loop needs.
type StartResponse struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	Interval        time.Duration
	ExpiresIn       time.Duration
}

// Start requests a device code pair from GitHub.
func (d *DeviceFlow) Start(ctx context.Context) (*StartResponse, error) {
	if d.ClientID == "" {
		return nil, errors.New("github: empty client id")
	}
	form := url.Values{}
	form.Set("client_id", d.ClientID)
	if d.Scope != "" {
		form.Set("scope", d.Scope)
	}

	var raw struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Error           string `json:"error"`
		ErrorDesc       string `json:"error_description"`
	}
	if err := d.postForm(ctx, deviceCodeURL, form, &raw); err != nil {
		return nil, err
	}
	if raw.Error != "" {
		return nil, fmt.Errorf("github: %s: %s", raw.Error, raw.ErrorDesc)
	}
	if raw.DeviceCode == "" || raw.UserCode == "" {
		return nil, errors.New("github: empty device or user code")
	}
	interval := time.Duration(raw.Interval) * time.Second
	if interval <= 0 {
		interval = defaultPollInterval
	}
	expires := time.Duration(raw.ExpiresIn) * time.Second
	if expires <= 0 {
		expires = 15 * time.Minute
	}
	return &StartResponse{
		DeviceCode:      raw.DeviceCode,
		UserCode:        raw.UserCode,
		VerificationURI: raw.VerificationURI,
		Interval:        interval,
		ExpiresIn:       expires,
	}, nil
}

// Poll repeatedly hits the token endpoint until the user authorizes (returns
// the access token), denies, or the code expires. The ctx deadline trumps
// GitHub's expiry; cancel ctx to abort early.
func (d *DeviceFlow) Poll(ctx context.Context, deviceCode string, interval time.Duration) (string, error) {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	form := url.Values{}
	form.Set("client_id", d.ClientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", deviceGrantTyp)

	for {
		// Wait first, since the spec says don't poll faster than `interval` and
		// the user needs time to authorize anyway.
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}

		var raw struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			Scope       string `json:"scope"`
			Error       string `json:"error"`
			ErrorDesc   string `json:"error_description"`
		}
		if err := d.postForm(ctx, tokenURL, form, &raw); err != nil {
			return "", err
		}
		if raw.AccessToken != "" {
			return raw.AccessToken, nil
		}
		switch raw.Error {
		case "authorization_pending":
			// keep polling at current interval
		case "slow_down":
			interval += 5 * time.Second
			if interval > maxPollInterval {
				interval = maxPollInterval
			}
		case "expired_token":
			return "", errors.New("github: code expired before authorization")
		case "access_denied":
			return "", errors.New("github: authorization denied")
		case "":
			return "", errors.New("github: empty response from token endpoint")
		default:
			return "", fmt.Errorf("github: %s: %s", raw.Error, raw.ErrorDesc)
		}
	}
}

// UserLogin returns the "login" field of the authenticated user.
func (d *DeviceFlow) UserLogin(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := d.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("github /user: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var u struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", err
	}
	if u.Login == "" {
		return "", errors.New("github: empty login in /user response")
	}
	return u.Login, nil
}

// postForm posts application/x-www-form-urlencoded, expecting JSON back.
func (d *DeviceFlow) postForm(ctx context.Context, endpoint string, form url.Values, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := d.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("github %s: %s: %s", endpoint, resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

func (d *DeviceFlow) client() *http.Client {
	if d.HTTP != nil {
		return d.HTTP
	}
	return http.DefaultClient
}
