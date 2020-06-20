package user

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"

	// database driver
	_ "github.com/mattn/go-sqlite3"

	"github.com/shaardie/mondane/user/proto"
)

// customClaims are JWT claims inclusive a user
type customClaims struct {
	User *proto.User
	jwt.StandardClaims
}

// authable interface
type authable interface {
	// decode a token and return the included custom claim
	decode(token string) (*customClaims, error)

	// encode a user into a token
	encode(user *proto.User) (string, error)
}

// tokenService fullfill the authable interface
type tokenService struct {
	db  repository
	key []byte
}

// decode a token and return the included custom claim
func (t *tokenService) decode(token string) (*customClaims, error) {
	// Parse the token
	tokenType, err := jwt.ParseWithClaims(token, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		return t.key, nil
	})

	// token type not found
	if tokenType == nil {
		return &customClaims{}, fmt.Errorf("Invalid token type")
	}

	// Validate the token and return the custom claims
	if claims, ok := tokenType.Claims.(*customClaims); ok && tokenType.Valid {
		return claims, nil
	}
	return nil, err
}

// encode a user into a token
func (t *tokenService) encode(user *proto.User) (string, error) {
	// Create the Claims
	claims := customClaims{
		user,
		jwt.StandardClaims{
			Issuer: "mondane.service.user",
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token and return
	return token.SignedString(t.key)
}
