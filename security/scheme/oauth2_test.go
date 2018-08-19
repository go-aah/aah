// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package scheme

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/security.v0/acrypto"
	"aahframework.org/security.v0/authc"
	"aahframework.org/test.v0/assert"
	"golang.org/x/oauth2"
)

func TestOAuth2InitializeError(t *testing.T) {
	testcases := []struct {
		label, config, keyname string
		err                    error
	}{
		{
			label: "Client ID missing",
			config: `
			security {
			  auth_schemes {
			    facebook_auth {
			      scheme = "oauth2"
					}
				}
			}
			`,
			keyname: "facebook_auth",
			err:     errors.New("facebook_auth: config 'security.auth_schemes.facebook_auth.client.id' is required"),
		},
		{
			label: "Client Secret missing",
			config: `
			security {
			  auth_schemes {
			    facebook_auth {
			      scheme = "oauth2"
						client {
							id = "client id"
						}
					}
				}
			}
			`,
			keyname: "facebook_auth",
			err:     errors.New("facebook_auth: config 'security.auth_schemes.facebook_auth.client.secret' is required"),
		},
		{
			label: "OAuth2 URLs is missing",
			config: `
			security {
			  auth_schemes {
			    facebook_auth {
			      scheme = "oauth2"
						client {
							id = "client id"
							secret = "client secret"
						}
						provider {
						}
					}
				}
			}
			`,
			keyname: "facebook_auth",
			err: errors.New("facebook_auth: either one is required 'security.auth_schemes.facebook_auth.client.provider.name' " +
				"or (security.auth_schemes.facebook_auth.client.provider.url.auth and " +
				"security.auth_schemes.facebook_auth.client.provider.url.token)"),
		},
		{
			label: "OAuth2 authorizer is missing",
			config: `
			security {
			  auth_schemes {
			    facebook_auth {
			      scheme = "oauth2"
						client {
							id = "client id"
							secret = "client secret"
							provider {
								name = "facebook"
							}
						}
						principal = "security/SubjectPrincipalProvider"
					}
				}
			}
			`,
			keyname: "facebook_auth",
			err: errors.New("facebook_auth: 'security.auth_schemes.facebook_auth.principal' " +
				"and 'security.auth_schemes.facebook_auth.authorizer' are required"),
		},
		{
			label: "OAuth2 correct config 1",
			config: `
			security {
			  auth_schemes {
			    facebook_auth {
			      scheme = "oauth2"
						client {
							id = "client id"
							secret = "client secret"
							provider {
								name = "facebook"
							}
						}
						principal = "security/SubjectPrincipalProvider"
						authorizer = "security/AuthorizationProvider"
					}
				}
			}
			`,
			keyname: "facebook_auth",
		},
		{
			label: "OAuth2 correct config 2",
			config: `
			security {
			  auth_schemes {
			    facebook_auth {
			      scheme = "oauth2"
						client {
							id = "client id"
							secret = "client secret"
							provider {
								url {
				          auth = "https://provider.com/o/oauth2/auth"
				          token = "https://provider.com/o/oauth2/token"
				        }
							}
						}
						principal = "security/SubjectPrincipalProvider"
						authorizer = "security/AuthorizationProvider"
					}
				}
			}
			`,
			keyname: "facebook_auth",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			oauth := new(OAuth2)
			cfg, _ := config.ParseString(tc.config)
			err := oauth.Init(cfg, tc.keyname)
			assert.Equal(t, tc.err, err)
		})
	}

}

func TestOAuth2SignAndVerify(t *testing.T) {
	oauth := &OAuth2{
		signSha: "sha-256",
		signKey: []byte("5a977494319cde3203fbb49711f08ad2"),
	}

	state, stateSigned := oauth.generateStateKey()
	result := oauth.validateStateKey(state, stateSigned)
	assert.True(t, result)

	result = oauth.validateStateKey(state, stateSigned+"y57653")
	assert.False(t, result)

	tstr := fmt.Sprintf("%v", time.Now().UTC().Truncate(time.Minute*20).UnixNano())
	result = oauth.validateStateKey(state[:33]+":"+tstr, stateSigned)
	assert.False(t, result)
}

func TestOAuth2LifeCycle(t *testing.T) {
	cfg, _ := config.ParseString(`
security {
	auth_schemes {
		local_auth {
			scheme = "oauth2"
			client {
				id = "clientid"
				secret = "clientsecret"
				sign_key = "5a977494319cde3203fbb49711f08ad2"
				provider {
					url {
						auth = "http://localhost/auth/login"
						token = "http://localhost/auth/token"
					}
				}
			}
			principal = "security/SubjectPrincipalProvider"
			authorizer = "security/AuthorizationProvider"
		}
	}
}`)

	oauth := new(OAuth2)
	err := oauth.Init(cfg, "local_auth")
	assert.Nil(t, err)

	ts := createOAuth2TestServer()
	defer ts.Close()

	t.Logf("Local dummy mock OAuth2 server: %s", ts.URL)
	oauth.Config().Endpoint = oauth2.Endpoint{
		AuthURL:  ts.URL + "/auth/login",
		TokenURL: ts.URL + "/auth/token",
	}

	r := ahttp.AcquireRequest(httptest.NewRequest("GET", ts.URL+"/login", nil))
	state, authURL := oauth.ProviderAuthURL(r)

	// Validate AuthURL
	t.Log("Validate AuthURL")
	u, err := url.Parse(authURL)
	assert.Nil(t, err)
	assert.NotNil(t, u)
	assert.Equal(t, oauth.Config().Endpoint.AuthURL, u.String()[:len(oauth.Config().Endpoint.AuthURL)])

	// Validate State value
	t.Log("Validate State value")
	urlState := u.Query().Get("state")
	counterState := base64.RawURLEncoding.EncodeToString(acrypto.Sign(oauth.signKey, []byte(state), oauth.signSha))
	assert.Equal(t, urlState, counterState)

	code := ess.SecureRandomString(32)
	testcases := []struct {
		label, authurl string
		token          *oauth2.Token
		err            error
	}{
		{
			label:   "Validate OAuth2 callback exchange error",
			authurl: authURL + "&code=" + code + ":senderror",
			err:     ErrOAuth2Exchange,
		},
		{
			label:   "Validate OAuth2 callback missing code or state",
			authurl: authURL,
			err:     ErrOAuth2MissingStateOrCode,
		},
		{
			label:   "Validate OAuth2 callback invalid state",
			authurl: authURL + "34FJEvsa&code=" + code,
			err:     ErrOAuth2InvalidState,
		},
		{
			label:   "Validate OAuth2 callback",
			authurl: authURL + "&code=" + code,
			token: &oauth2.Token{
				AccessToken:  "EAACmZAkEPRWwBABp3pPRSAww7i4NSIbGHjwmGpR0tuqN29ZCXA2",
				RefreshToken: "SzGotMzeKoIlCLVrZApwEfo4zNA10mcsWMeViZAy2y7legE6aEZD",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.label, func(t *testing.T) {
			callbackReq := httptest.NewRequest("GET", tc.authurl, nil)
			token, err := oauth.ValidateCallback(state, ahttp.AcquireRequest(callbackReq))
			assert.Equal(t, tc.err, err)
			if tc.token == nil {
				assert.Equal(t, tc.token, token)
			} else {
				assert.Equal(t, tc.token.AccessToken, token.AccessToken)
				assert.Equal(t, tc.token.RefreshToken, token.RefreshToken)
			}
		})
	}

	result, err := oauth.Principal("local_auth", nil)
	assert.Equal(t, errors.New("oauth2: 'security.auth_schemes.local_auth.provider.principal' not configured properly"), err)

	oauth.SetPrincipalProvider(&principalprovider{})
	result, err = oauth.Principal("local_auth", nil)
	assert.Nil(t, err)
	assert.True(t, result[0].IsPrimary)
	assert.Equal(t, "Email", result[0].Claim)
	assert.Equal(t, "test@test.com", result[0].Value)
}

func TestOAuth2InferEndpoint(t *testing.T) {
	for _, ep := range []string{
		"amazon", "bitbucket", "cern", "facebook", "fitbit",
		"foursquare", "github", "gitlab", "google", "heroku",
		"hipchat", "kakao", "linkedin", "mailchimp", "mailru",
		"mediamath", "microsoft", "odnoklassniki", "paypal",
		"slack", "spotify", "twitch", "uber", "vk", "yahoo",
		"yandex",
	} {
		t.Run(ep+" endpoint url test", func(t *testing.T) {
			endpoint := inferEndpoint(ep)
			assert.True(t, len(endpoint.AuthURL) > 0)
			assert.True(t, len(endpoint.TokenURL) > 0)
		})
	}

	endpoint := inferEndpoint("nil")
	assert.True(t, len(endpoint.AuthURL) == 0)
	assert.True(t, len(endpoint.TokenURL) == 0)

	// Azure
	for _, ep := range []string{
		"azure.jeeva", "azure.example",
	} {
		t.Run(ep+" endpoint url test", func(t *testing.T) {
			endpoint := inferEndpoint(ep)
			tenantName := ep[6:]
			assert.Equal(t,
				fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantName),
				endpoint.AuthURL)
			assert.Equal(t,
				fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantName),
				endpoint.TokenURL)
		})
	}
}

func createOAuth2TestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/auth/token" {
			_ = r.ParseForm()
			if strings.HasSuffix(r.FormValue("code"), ":senderror") {
				return
			}

			// test token
			w.Header().Set("Content-Type", "application/json")
			token := &oauth2.Token{
				TokenType:    "bearer",
				AccessToken:  "EAACmZAkEPRWwBABp3pPRSAww7i4NSIbGHjwmGpR0tuqN29ZCXA2",
				RefreshToken: "SzGotMzeKoIlCLVrZApwEfo4zNA10mcsWMeViZAy2y7legE6aEZD",
				Expiry:       time.Now().UTC().AddDate(0, 0, 60),
			}
			_ = json.NewEncoder(w).Encode(token)
			return
		}
	}))
}

type principalprovider struct{}

func (p *principalprovider) Init(_ *config.Config) error { return nil }
func (p *principalprovider) Principal(keyName string, v ess.Valuer) ([]*authc.Principal, error) {
	principals := make([]*authc.Principal, 0)
	principals = append(principals, &authc.Principal{Realm: "Local", IsPrimary: true, Claim: "Email", Value: "test@test.com"})
	return principals, nil
}
