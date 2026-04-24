package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/synctv-org/synctv/internal/provider"
	pluginshared "github.com/synctv-org/synctv/internal/provider/plugins"
	"golang.org/x/oauth2"
)

const (
	authentikProviderName = "authentik"
	authURLPath           = "/application/o/authorize/"
	tokenURLPath          = "/application/o/token/"
	userInfoURLPath       = "/application/o/userinfo/"
)

var defaultScopes = []string{"openid", "profile", "email"}

// Build:
//
//	CGO_ENABLED=0 go build -o authentik ./internal/provider/plugins/authentik
//
// Config:
//
//	oauth2_plugins:
//	  - plugin_file: plugins/oauth2/authentik
//	    args: ["https://authentik.example.com"]
type authentikProvider struct {
	config      oauth2.Config
	userInfoURL string
}

func newAuthentikProvider(rawBaseURL string, scopes []string) (*authentikProvider, error) {
	baseURL, err := normalizeBaseURL(rawBaseURL)
	if err != nil {
		return nil, err
	}

	if len(scopes) == 0 {
		scopes = append([]string(nil), defaultScopes...)
	}

	return &authentikProvider{
		config: oauth2.Config{
			Scopes: scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  baseURL + authURLPath,
				TokenURL: baseURL + tokenURLPath,
			},
		},
		userInfoURL: baseURL + userInfoURLPath,
	}, nil
}

func normalizeBaseURL(rawBaseURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rawBaseURL))
	if err != nil {
		return "", fmt.Errorf("parse authentik url: %w", err)
	}

	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid authentik url %q", rawBaseURL)
	}

	u.RawQuery = ""
	u.Fragment = ""

	return strings.TrimRight(u.String(), "/"), nil
}

func (p *authentikProvider) Init(opt provider.Oauth2Option) {
	p.config.ClientID = opt.ClientID
	p.config.ClientSecret = opt.ClientSecret
	p.config.RedirectURL = opt.RedirectURL
}

func (p *authentikProvider) Provider() provider.OAuth2Provider {
	return authentikProviderName
}

func (p *authentikProvider) NewAuthURL(_ context.Context, state string) (string, error) {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

func (p *authentikProvider) GetUserInfo(ctx context.Context, code string) (*provider.UserInfo, error) {
	tk, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	client := p.config.Client(ctx, tk)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("userinfo request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var ui authentikUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&ui); err != nil {
		return nil, err
	}

	return &provider.UserInfo{
		ProviderUserID: ui.Sub,
		Username:       ui.username(),
	}, nil
}

type authentikUserInfo struct {
	Sub               string `json:"sub"`
	PreferredUsername string `json:"preferred_username"`
	Nickname          string `json:"nickname"`
	Name              string `json:"name"`
	Email             string `json:"email"`
}

func (u authentikUserInfo) username() string {
	for _, candidate := range []string{
		u.PreferredUsername,
		u.Nickname,
		u.Name,
		u.Email,
	} {
		if candidate != "" {
			return candidate
		}
	}

	return ""
}

func parseScopes(rawScopes []string) []string {
	if len(rawScopes) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(rawScopes))
	scopes := make([]string, 0, len(rawScopes))

	for _, rawScope := range rawScopes {
		for _, scope := range strings.FieldsFunc(rawScope, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\n'
		}) {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				continue
			}

			if _, ok := seen[scope]; ok {
				continue
			}

			seen[scope] = struct{}{}
			scopes = append(scopes, scope)
		}
	}

	return scopes
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: authentik <authentik-base-url> [scope ...]")
	}

	p, err := newAuthentikProvider(os.Args[1], parseScopes(os.Args[2:]))
	if err != nil {
		log.Fatalf("init authentik provider: %v", err)
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: pluginshared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"Provider": &pluginshared.ProviderPlugin{Impl: p},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
