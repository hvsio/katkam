package auth

import (
	"fmt"
	"katkam/internal/config"
	repo "katkam/internal/infrastructure/repository"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type Authorizer struct {
	config   config.Auth
	userRepo *repo.UserRepository
}

func NewAuthorizer(config config.Auth, userRepo *repo.UserRepository) *Authorizer {
	return &Authorizer{config: config, userRepo: userRepo}
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

func (a *Authorizer) VerifyJWT(token string) (bool, error) {
	strippedToken := strings.TrimPrefix(token, "Bearer ")
	parsedToken, err := jwt.Parse(strippedToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.config.JwtSecretKey), nil
	})

	if err != nil {
		return false, err
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		return claims["authorized"] == true && claims["exp"].(float64) > float64(time.Now().Unix()), nil
	}

	return false, nil
}

func (a *Authorizer) GetJwtToken(username string) (JwtToken, error) {
	return a.generateJwt(username)
}

func (a *Authorizer) authenticate(username, password string) (bool, error) {
	user, err := a.userRepo.GetByUsername(username)
	if err != nil {
		return false, ErrorUserNotFound
	}

	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	fmt.Println(string(b))
	if err != nil {
		return false, ErrorInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)); err != nil {
		return false, ErrorInvalidCredentials
	}

	return true, nil
}

func (a *Authorizer) generateJwt(username string) (JwtToken, error) {
	secretKey := []byte(a.config.JwtSecretKey)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":        time.Now().Add(time.Duration(a.config.ExpirationTime) * time.Second).Unix(),
		"authorized": true,
		"user":       username,
	})

	tokenString, err := token.SignedString(secretKey)
	return JwtToken(tokenString), err
}
