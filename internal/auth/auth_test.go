package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tx-mi/bazgain-places-go/internal/places"
)

const (
	testSecret   = "01234567890123456789012345678901"
	testIssuer   = "bazgain"
	testAudience = "bazgain-mobile"
	testUserID   = "019a3ee8-f95c-7000-8000-000000000001"
	testSession  = "019a3ee8-f95c-7000-8000-000000000002"
)

type userStoreStub struct {
	user *User
	err  error
}

func (s userStoreStub) FindAuthUser(context.Context, string) (*User, error) { return s.user, s.err }

type revocationStub struct {
	revoked bool
	err     error
}

func (s revocationStub) SessionRevoked(context.Context, string) (bool, error) {
	return s.revoked, s.err
}

func signedToken(t *testing.T, method jwt.SigningMethod, mutate func(*Claims)) string {
	t.Helper()
	version := 3
	claims := &Claims{
		SessionID: testSession, TokenVersion: &version, Role: "admin", Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: testUserID, Issuer: testIssuer, Audience: jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	if mutate != nil {
		mutate(claims)
	}
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func validAuthenticator(user *User, revocation revocationStub) *Authenticator {
	return New(testSecret, testIssuer, testAudience, userStoreStub{user: user}, revocation)
}

func assertAppErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	var appErr *places.AppError
	if !errors.As(err, &appErr) || appErr.Code != code {
		t.Fatalf("error = %#v, want code %q", err, code)
	}
}

func TestAuthenticateValidAdminUsesCurrentDatabaseRole(t *testing.T) {
	authenticator := validAuthenticator(&User{ID: testUserID, State: "active", Role: "admin", TokenVersion: 3}, revocationStub{})
	actor, err := authenticator.Authenticate(context.Background(), "Bearer "+signedToken(t, jwt.SigningMethodHS256, nil))
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if actor.UserID != testUserID || actor.SessionID != testSession || actor.Role != "admin" || actor.TokenVersion != 3 {
		t.Fatalf("actor = %#v", actor)
	}
}

func TestAuthenticateRejectsMalformedAuthorization(t *testing.T) {
	authenticator := validAuthenticator(nil, revocationStub{})
	for _, header := range []string{"", "Bearer", "Basic abc", "Bearer a b"} {
		_, err := authenticator.Authenticate(context.Background(), header)
		assertAppErrorCode(t, err, "access_token_invalid")
	}
}

func TestAuthenticateRejectsInvalidJWTClaims(t *testing.T) {
	baseUser := &User{ID: testUserID, State: "active", Role: "admin", TokenVersion: 3}
	cases := map[string]struct {
		method jwt.SigningMethod
		mutate func(*Claims)
	}{
		"wrong algorithm": {method: jwt.SigningMethodHS512},
		"wrong issuer":    {method: jwt.SigningMethodHS256, mutate: func(c *Claims) { c.Issuer = "other" }},
		"wrong audience":  {method: jwt.SigningMethodHS256, mutate: func(c *Claims) { c.Audience = jwt.ClaimStrings{"other"} }},
		"expired":         {method: jwt.SigningMethodHS256, mutate: func(c *Claims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute)) }},
		"wrong type":      {method: jwt.SigningMethodHS256, mutate: func(c *Claims) { c.Type = "refresh" }},
		"missing subject": {method: jwt.SigningMethodHS256, mutate: func(c *Claims) { c.Subject = "" }},
		"invalid subject": {method: jwt.SigningMethodHS256, mutate: func(c *Claims) { c.Subject = "invalid" }},
		"invalid session": {method: jwt.SigningMethodHS256, mutate: func(c *Claims) { c.SessionID = "invalid" }},
		"missing version": {method: jwt.SigningMethodHS256, mutate: func(c *Claims) { c.TokenVersion = nil }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			authenticator := validAuthenticator(baseUser, revocationStub{})
			_, err := authenticator.Authenticate(context.Background(), "Bearer "+signedToken(t, tc.method, tc.mutate))
			assertAppErrorCode(t, err, "access_token_invalid")
		})
	}
}

func TestAuthenticateRejectsInvalidSignature(t *testing.T) {
	version := 3
	claims := &Claims{
		SessionID: testSession, TokenVersion: &version, Role: "admin", Type: "access",
		RegisteredClaims: jwt.RegisteredClaims{Subject: testUserID, Issuer: testIssuer, Audience: jwt.ClaimStrings{testAudience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("different-secret-with-enough-entropy"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = validAuthenticator(&User{ID: testUserID, State: "active", Role: "admin", TokenVersion: 3}, revocationStub{}).Authenticate(context.Background(), "Bearer "+token)
	assertAppErrorCode(t, err, "access_token_invalid")
}

func TestAuthenticateRejectsSignedTokenWithoutVersionClaim(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": testUserID, "sid": testSession, "role": "admin", "typ": "access",
		"iss": testIssuer, "aud": testAudience, "exp": time.Now().Add(time.Hour).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	_, err = validAuthenticator(&User{ID: testUserID, State: "active", Role: "admin", TokenVersion: 0}, revocationStub{}).Authenticate(context.Background(), "Bearer "+token)
	assertAppErrorCode(t, err, "access_token_invalid")
}

func TestAuthenticateRejectsOversizedAuthorizationHeader(t *testing.T) {
	_, err := validAuthenticator(nil, revocationStub{}).Authenticate(context.Background(), "Bearer "+strings.Repeat("a", 8193))
	assertAppErrorCode(t, err, "access_token_invalid")
}

func TestAuthenticateChecksRevocationAndDependencyFailure(t *testing.T) {
	user := &User{ID: testUserID, State: "active", Role: "admin", TokenVersion: 3}
	token := "Bearer " + signedToken(t, jwt.SigningMethodHS256, nil)
	_, err := validAuthenticator(user, revocationStub{revoked: true}).Authenticate(context.Background(), token)
	assertAppErrorCode(t, err, "access_token_invalid")

	_, err = validAuthenticator(user, revocationStub{err: errors.New("redis down")}).Authenticate(context.Background(), token)
	assertAppErrorCode(t, err, "internal_error")
}

func TestAuthenticateChecksDatabaseStateVersionAndRole(t *testing.T) {
	token := "Bearer " + signedToken(t, jwt.SigningMethodHS256, nil)
	cases := map[string]struct {
		user *User
		code string
	}{
		"missing":            {user: nil, code: "access_token_invalid"},
		"blocked":            {user: &User{ID: testUserID, State: "blocked", Role: "admin", TokenVersion: 3}, code: "access_token_invalid"},
		"version mismatch":   {user: &User{ID: testUserID, State: "active", Role: "admin", TokenVersion: 4}, code: "access_token_invalid"},
		"database role user": {user: &User{ID: testUserID, State: "active", Role: "user", TokenVersion: 3}, code: "forbidden"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := validAuthenticator(tc.user, revocationStub{}).Authenticate(context.Background(), token)
			assertAppErrorCode(t, err, tc.code)
		})
	}
}

func TestAuthenticateFailsClosedOnUserStoreFailureAndIgnoresStaleRoleClaim(t *testing.T) {
	token := "Bearer " + signedToken(t, jwt.SigningMethodHS256, func(claims *Claims) { claims.Role = "user" })
	authenticator := New(testSecret, testIssuer, testAudience, userStoreStub{err: errors.New("database down")}, revocationStub{})
	_, err := authenticator.Authenticate(context.Background(), token)
	assertAppErrorCode(t, err, "internal_error")

	authenticator = validAuthenticator(&User{ID: testUserID, State: "active", Role: "admin", TokenVersion: 3}, revocationStub{})
	actor, err := authenticator.Authenticate(context.Background(), token)
	if err != nil || actor.Role != "admin" {
		t.Fatalf("current database role must win: actor=%#v err=%v", actor, err)
	}
}
