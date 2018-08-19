// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/security.v0/acrypto"
	"aahframework.org/security.v0/authc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/amazon"
	"golang.org/x/oauth2/bitbucket"
	"golang.org/x/oauth2/cern"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/fitbit"
	"golang.org/x/oauth2/foursquare"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/heroku"
	"golang.org/x/oauth2/hipchat"
	"golang.org/x/oauth2/kakao"
	"golang.org/x/oauth2/linkedin"
	"golang.org/x/oauth2/mailchimp"
	"golang.org/x/oauth2/mailru"
	"golang.org/x/oauth2/mediamath"
	"golang.org/x/oauth2/microsoft"
	"golang.org/x/oauth2/odnoklassniki"
	"golang.org/x/oauth2/paypal"
	"golang.org/x/oauth2/slack"
	"golang.org/x/oauth2/spotify"
	"golang.org/x/oauth2/twitch"
	"golang.org/x/oauth2/uber"
	"golang.org/x/oauth2/vk"
	"golang.org/x/oauth2/yahoo"
	"golang.org/x/oauth2/yandex"
)

var _ Schemer = (*OAuth2)(nil)

// OAuth2 Errors
var (
	ErrOAuth2MissingStateOrCode = errors.New("oauth2: callback missing state or code")
	ErrOAuth2InvalidState       = errors.New("oauth2: invalid state")
	ErrOAuth2Exchange           = errors.New("oauth2: exchange failed, unable to get token")
	ErrOAuth2TokenIsValid       = errors.New("oauth2: token is vaild")
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// OAuth2 Auth Scheme
//______________________________________________________________________________

type OAuth2 struct {
	BaseAuth
	LoginURL    string
	RedirectURL string
	SuccessURL  string

	redirectUpdated bool
	signSha         string
	signKey         []byte
	appCfg          *config.Config
	oauthCfg        *oauth2.Config
}

// Init method initialize the OAuth2 auth scheme during an application start.
func (o *OAuth2) Init(appCfg *config.Config, keyName string) error {
	o.AppConfig = appCfg
	o.KeyName = keyName
	o.KeyPrefix = "security.auth_schemes." + o.KeyName
	o.Name, _ = o.AppConfig.String(o.ConfigKey("scheme"))

	clientId, found := o.AppConfig.String(o.ConfigKey("client.id"))
	if !found {
		return o.ConfigError("client.id")
	}

	clientSecret, found := o.AppConfig.String(o.ConfigKey("client.secret"))
	if !found {
		return o.ConfigError("client.secret")
	}

	o.signSha = "sha-256"
	o.signKey = []byte(o.AppConfig.StringDefault(o.ConfigKey("client.sign_key"), ess.SecureRandomString(32)))

	o.oauthCfg = &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
	}

	o.oauthCfg.Scopes, _ = o.AppConfig.StringList(o.ConfigKey("client.scopes"))
	provider := o.AppConfig.StringDefault(o.ConfigKey("client.provider.name"), "nil")
	endpoint := inferEndpoint(provider)
	if ess.IsStrEmpty(endpoint.AuthURL) && ess.IsStrEmpty(endpoint.TokenURL) {
		authURL := o.AppConfig.StringDefault(o.ConfigKey("client.provider.url.auth"), "")
		tokenURL := o.AppConfig.StringDefault(o.ConfigKey("client.provider.url.token"), "")
		if ess.IsStrEmpty(authURL) || ess.IsStrEmpty(tokenURL) {
			return fmt.Errorf("%s: either one is required '%s' or (%s and %s)",
				o.KeyName, o.ConfigKey("client.provider.name"),
				o.ConfigKey("client.provider.url.auth"),
				o.ConfigKey("client.provider.url.token"))
		}
		o.oauthCfg.Endpoint = oauth2.Endpoint{AuthURL: authURL, TokenURL: tokenURL}
	} else {
		o.oauthCfg.Endpoint = endpoint
	}

	principal := o.AppConfig.StringDefault(o.ConfigKey("principal"), "")
	authorizer := o.AppConfig.StringDefault(o.ConfigKey("authorizer"), "")
	if ess.IsStrEmpty(principal) || ess.IsStrEmpty(authorizer) {
		return fmt.Errorf("%s: '%s' and '%s' are required", o.KeyName, o.ConfigKey("principal"), o.ConfigKey("authorizer"))
	}

	o.LoginURL = o.AppConfig.StringDefault(o.ConfigKey("url.login"), createDefaultURL(keyName, "login"))
	o.RedirectURL = o.AppConfig.StringDefault(o.ConfigKey("url.redirect"), createDefaultURL(keyName, "callback"))
	o.SuccessURL = o.AppConfig.StringDefault(o.ConfigKey("url.success"), "/")
	o.oauthCfg.RedirectURL = o.RedirectURL

	return nil
}

// Config method returns OAuth2 config instance.
func (o *OAuth2) Config() *oauth2.Config {
	return o.oauthCfg
}

// Client method returns Go HTTP client configured with given OAuth2 Token.
func (o *OAuth2) Client(token *oauth2.Token) *http.Client {
	return o.oauthCfg.Client(context.Background(), token)
}

// RefreshAccessToken method returns new OAuth2 token if given token was expried
// otherwise returns error `scheme.ErrOAuth2TokenIsValid`.
func (o *OAuth2) RefreshAccessToken(token *oauth2.Token) (*oauth2.Token, error) {
	tsrc := o.oauthCfg.TokenSource(context.Background(), token)
	tn, err := tsrc.Token()
	if err != nil {
		return nil, err
	}

	// if its same access token then given token is stil vaild
	if tn.AccessToken == token.AccessToken {
		return nil, ErrOAuth2TokenIsValid
	}

	return tn, nil
}

// ProviderAuthURL method returns aah generated state value and OAuth2 login URL.
func (o *OAuth2) ProviderAuthURL(r *ahttp.Request) (string, string) {
	if !o.redirectUpdated {
		if !strings.HasPrefix(o.RedirectURL, "http") { // this is for possiblity check
			base, _ := url.Parse(r.Scheme + "://" + r.Host)
			u, _ := url.Parse(o.RedirectURL)
			o.oauthCfg.RedirectURL = base.ResolveReference(u).String()
			o.redirectUpdated = true
		}
	}

	state, signedState := o.generateStateKey()
	authURL := o.oauthCfg.AuthCodeURL(signedState)
	return state, authURL
}

// ValidateCallback method validates the incoming OAuth2 provider redirect request
// and gets Access token from OAuth2 provider.
func (o *OAuth2) ValidateCallback(state string, r *ahttp.Request) (*oauth2.Token, error) {
	callbackState, code := r.FormValue("state"), r.FormValue("code")
	if ess.IsStrEmpty(callbackState) || ess.IsStrEmpty(code) {
		return nil, ErrOAuth2MissingStateOrCode
	}

	// Validate state value
	if !o.validateStateKey(state, callbackState) {
		return nil, ErrOAuth2InvalidState
	}

	// Now get the access token
	token, err := o.oauthCfg.Exchange(context.TODO(), code)
	if err != nil {
		return nil, ErrOAuth2Exchange
	}

	return token, nil
}

// Principal method calls the registered interface `SubjectPrincipalProvider`
// to obtain Subject principals.
func (o *OAuth2) Principal(keyName string, v ess.Valuer) ([]*authc.Principal, error) {
	if o.principalProvider == nil {
		return nil, fmt.Errorf("%s: '%s.provider.principal' not configured properly", o.Scheme(), o.KeyPrefix)
	}
	return o.principalProvider.Principal(keyName, v)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// OAuth2 Unexported methods
//______________________________________________________________________________

func (o *OAuth2) generateStateKey() (string, string) {
	state := ess.SecureRandomString(32) + ":" + fmt.Sprintf("%v", time.Now().UTC().UnixNano())
	return state, base64.RawURLEncoding.EncodeToString(acrypto.Sign(o.signKey, []byte(state), o.signSha))
}

func (o *OAuth2) validateStateKey(state, signedState string) bool {
	b, err := base64.RawURLEncoding.DecodeString(signedState)
	if err != nil {
		return false
	}

	if !acrypto.Verify(o.signKey, []byte(state), b, o.signSha) {
		return false
	}

	// check duration, aah state key is only valid for 10 minutes
	utcNano, _ := strconv.ParseInt(state[33:], 10, 64)
	min := time.Now().UTC().Sub(time.Unix(0, utcNano).UTC()).Minutes()
	return int(min) <= 10
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package Unexported methods
//______________________________________________________________________________

func createDefaultURL(keyName, suffix string) string {
	keyName = strings.Replace(keyName, "_", "-", -1)
	return "/" + path.Join(keyName, suffix)
}

// just alphabet order, no preferences
func inferEndpoint(provider string) oauth2.Endpoint {
	switch provider {
	case "amazon":
		return amazon.Endpoint
	case "bitbucket":
		return bitbucket.Endpoint
	case "cern":
		return cern.Endpoint
	case "facebook":
		return facebook.Endpoint
	case "fitbit":
		return fitbit.Endpoint
	case "foursquare":
		return foursquare.Endpoint
	case "github":
		return github.Endpoint
	case "gitlab":
		return gitlab.Endpoint
	case "google":
		return google.Endpoint
	case "heroku":
		return heroku.Endpoint
	case "hipchat":
		return hipchat.Endpoint
	case "kakao":
		return kakao.Endpoint
	case "linkedin":
		return linkedin.Endpoint
	case "mailchimp":
		return mailchimp.Endpoint
	case "mailru":
		return mailru.Endpoint
	case "mediamath":
		return mediamath.Endpoint
	case "microsoft":
		return microsoft.LiveConnectEndpoint
	case "odnoklassniki":
		return odnoklassniki.Endpoint
	case "paypal":
		return paypal.Endpoint
	case "slack":
		return slack.Endpoint
	case "spotify":
		return spotify.Endpoint
	case "twitch":
		return twitch.Endpoint
	case "uber":
		return uber.Endpoint
	case "vk":
		return vk.Endpoint
	case "yahoo":
		return yahoo.Endpoint
	case "yandex":
		return yandex.Endpoint
	}

	// handling AzureADEndpoint
	if strings.HasPrefix(provider, "azure") {
		return microsoft.AzureADEndpoint(provider[6:])
	}

	return oauth2.Endpoint{}
}
