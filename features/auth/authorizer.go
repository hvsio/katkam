package auth

import (
	"katkam/config"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type Authorizer struct {
	config config.AuthConfig
	db     any
}

func NewAuthorizer(config config.AuthConfig, db any) *Authorizer {
	return &Authorizer{config: config, db: db}
}

func (a *Authorizer) AuthorizeUser(username, password string) (bool, error) {
	ok, err := a.authenticate(username, password)
	if err != nil {
		return false, err
	}

	if !ok {
		return false, ErrorInvalidCredentials
	}

	return true, err
}

func (a *Authorizer) GetJwtToken(username string) (JwtToken, error) {
	return a.generateJwt(username)
}

func (a *Authorizer) authenticate(username, password string) (bool, error) {
	var storedHashedPassword = "a"
	//TODO: make a mocked db with auth users
	// if err := a.verifyPassword(storedHashedPassword, password); err != nil {
	// 	return false, ErrorInvalidCredentials
	// }
	if storedHashedPassword != password {
		return false, ErrorInvalidCredentials
	}
	return true, nil
}

func (a *Authorizer) verifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (a *Authorizer) generateJwt(username string) (JwtToken, error) {
	secretKey := []byte(a.config.SecretKey)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":        time.Now().Add(time.Duration(10) * time.Second).Unix(),
		"authorized": true,
		"user":       username,
	})

	tokenString, err := token.SignedString(secretKey)
	return JwtToken(tokenString), err
}

func (a *Authorizer) VerifyJWT(token string) (bool, error) {
	strippedToken := strings.TrimPrefix(token, "Bearer ")
	parsedToken, err := jwt.Parse(strippedToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.SecretKey), nil
	})

	if err != nil {
		return false, err
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		return claims["authorized"] == true && claims["exp"].(float64) > float64(time.Now().Unix()), nil
	}

	return false, nil
}
