package handler

import (
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	auditlog "github.com/teamhanko/hanko/backend/audit_log"
	"github.com/teamhanko/hanko/backend/config"
	"github.com/teamhanko/hanko/backend/crypto/jwk"
	"github.com/teamhanko/hanko/backend/dto"
	"github.com/teamhanko/hanko/backend/session"
	"github.com/teamhanko/hanko/backend/test"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestThirdPartySuite(t *testing.T) {
	suite.Run(t, new(thirdPartySuite))
}

type thirdPartySuite struct {
	test.Suite
}

func (s *thirdPartySuite) setUpContext(request *http.Request) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	e.Validator = dto.NewCustomValidator()
	rec := httptest.NewRecorder()
	c := e.NewContext(request, rec)
	return c, rec
}

func (s *thirdPartySuite) setUpHandler(cfg *config.Config) *ThirdPartyHandler {
	auditLogger := auditlog.NewLogger(s.Storage, cfg.AuditLog)

	jwkMngr, err := jwk.NewDefaultManager(cfg.Secrets.Keys, s.Storage.GetJwkPersister())
	s.Require().NoError(err)

	sessionMngr, err := session.NewManager(jwkMngr, cfg.Session)
	s.Require().NoError(err)

	handler := NewThirdPartyHandler(cfg, s.Storage, sessionMngr, auditLogger)
	return handler
}

func (s *thirdPartySuite) setUpConfig(enabledProviders []string, allowedRedirectURLs []string) *config.Config {
	cfg := &config.Config{
		ThirdParty: config.ThirdParty{
			Providers: config.ThirdPartyProviders{
				Google: config.ThirdPartyProvider{
					Enabled:  false,
					ClientID: "fakeClientID",
					Secret:   "fakeClientSecret",
				}, GitHub: config.ThirdPartyProvider{
					Enabled:  false,
					ClientID: "fakeClientID",
					Secret:   "fakeClientSecret",
				}},
			ErrorRedirectURL:    "https://error.test.example",
			RedirectURL:         "https://api.test.example/callback",
			AllowedRedirectURLS: allowedRedirectURLs,
		},
		Secrets: config.Secrets{
			Keys: []string{"thirty-two-byte-long-test-secret"},
		},
		AuditLog: config.AuditLog{
			Storage: config.AuditLogStorage{Enabled: true},
		},
		Emails: config.Emails{
			MaxNumOfAddresses: 5,
		},
	}

	for _, provider := range enabledProviders {
		switch provider {
		case "google":
			cfg.ThirdParty.Providers.Google.Enabled = true
		case "github":
			cfg.ThirdParty.Providers.GitHub.Enabled = true
		}
	}

	err := cfg.PostProcess()
	s.Require().NoError(err)

	return cfg
}

func (s *thirdPartySuite) assertLocationHeaderHasToken(rec *httptest.ResponseRecorder) {
	location, err := url.Parse(rec.Header().Get("Location"))
	s.NoError(err)
	s.True(location.Query().Has(HankoTokenQuery))
	s.NotEmpty(location.Query().Get(HankoTokenQuery))
}

func (s *thirdPartySuite) assertStateCookieRemoved(rec *httptest.ResponseRecorder) {
	cookies := rec.Result().Cookies()
	s.Len(cookies, 1)
	s.Equal(HankoThirdpartyStateCookie, cookies[0].Name)
	s.Equal(-1, cookies[0].MaxAge)
}
