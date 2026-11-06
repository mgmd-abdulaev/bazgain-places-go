package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mgmd-abdulaev/bazgain-places-go/internal/places"
)

type Claims struct {
	SessionID    string `json:"sid"`
	TokenVersion *int   `json:"ver"`
	Role         string `json:"role"`
	Type         string `json:"typ"`
	jwt.RegisteredClaims
}

func (c Claims) Validate() error {
	if c.Type != "access" {
		return fmt.Errorf("unexpected token type")
	}
	if c.Subject == "" || c.SessionID == "" || c.TokenVersion == nil {
		return fmt.Errorf("missing subject or session")
	}
	if _, err := uuid.Parse(c.Subject); err != nil {
		return fmt.Errorf("invalid subject")
	}
	if _, err := uuid.Parse(c.SessionID); err != nil {
		return fmt.Errorf("invalid session")
	}
	return nil
}

type User struct {
	ID, State, Role string
	TokenVersion    int
}
type UserStore interface {
	FindAuthUser(context.Context, string) (*User, error)
}
type RevocationStore interface {
	SessionRevoked(context.Context, string) (bool, error)
}

type Authenticator struct {
	secret           []byte
	issuer, audience string
	users            UserStore
	revocations      RevocationStore
}

func New(secret, issuer, audience string, users UserStore, revocations RevocationStore) *Authenticator {
	return &Authenticator{secret: []byte(secret), issuer: issuer, audience: audience, users: users, revocations: revocations}
}

func (a *Authenticator) Authenticate(ctx context.Context, header string) (places.Actor, error) {
	if len(header) > 8192 {
		return places.Actor{}, places.Unauthorized()
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return places.Actor{}, places.Unauthorized()
	}
	claims := &Claims{}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(a.issuer), jwt.WithAudience(a.audience), jwt.WithExpirationRequired())
	token, err := parser.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (any, error) { return a.secret, nil })
	if err != nil || !token.Valid || claims.Validate() != nil {
		return places.Actor{}, places.Unauthorized()
	}
	revoked, err := a.revocations.SessionRevoked(ctx, claims.SessionID)
	if err != nil {
		return places.Actor{}, places.Internal(err)
	}
	if revoked {
		return places.Actor{}, places.Unauthorized()
	}
	user, err := a.users.FindAuthUser(ctx, claims.Subject)
	if err != nil {
		return places.Actor{}, places.Internal(err)
	}
	if user == nil || user.State != "active" || claims.TokenVersion == nil || user.TokenVersion != *claims.TokenVersion {
		return places.Actor{}, places.Unauthorized()
	}
	if user.Role != "admin" {
		return places.Actor{}, places.Forbidden()
	}
	return places.Actor{UserID: user.ID, SessionID: claims.SessionID, Role: user.Role, TokenVersion: user.TokenVersion}, nil
}
