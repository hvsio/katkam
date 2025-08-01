package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type JwtToken string

var (
	ErrorInvalidCredentials = errors.New("Invalid credentials")
)

type Authorizer struct {
	db any
}

func NewAuthorizer(db any) *Authorizer {
	return &Authorizer{db: db}
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

func (a *Authorizer) GetJwtToken(username, password string) (JwtToken, error) {
	return a.generateJwt()
}

func (a *Authorizer) authenticate(username, password string) (bool, error) {
	var storedHashedPassword string
	//TODO: make a mocked db with auth users
	if err := a.verifyPassword(storedHashedPassword, password); err != nil {
		return false, ErrorInvalidCredentials
	}
	return true, nil
}

func (a *Authorizer) verifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (a *Authorizer) generateJwt() (JwtToken, error) {
	sampleSecretKey := ""
	token := jwt.New(jwt.SigningMethodEdDSA)
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(10 * time.Minute) //TODO: timeout
	claims["authorized"] = true
	claims["user"] = "username"

	tokenString, err := token.SignedString(sampleSecretKey)
	if err != nil {
		return "", err
	}

	return JwtToken(tokenString), nil
}

func verifyJWT(endpointHandler func(writer http.ResponseWriter, request *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header["Token"] != nil {
			token, err := jwt.Parse(request.Header["Token"][0], func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte("secret"), nil
			})

			if err != nil {
				writer.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintf(writer, "Error: %v", err)
				return
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				fmt.Fprintf(writer, "User: %v", claims["user"])
				endpointHandler(writer, request)
			}
		}
	})
}
