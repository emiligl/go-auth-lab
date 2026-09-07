package main

import (
	"time"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("super-secret-key")

func CreateJWT(user User) (string, error) {

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"user_name":user.Username,
		"role":    user.Role,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(jwtSecret)
}

func getSigningKey(token *jwt.Token) (interface{}, error) {

	fmt.Printf("TOKEN: %+v\n", token)

	fmt.Printf("METHOD: %+v\n", token.Method)
	fmt.Printf("HEADER: %+v\n", token.Header)
	fmt.Printf("CLAIMS: %+v\n", token.Claims)
	fmt.Printf("SIGNATURE: %+v\n", token.Signature)
	fmt.Printf("VALID: %+v\n", token.Valid)

	method := token.Method
	expectedMethod := jwt.SigningMethodHS256

	if method != expectedMethod {
		return nil, fmt.Errorf("unexpected signing method")
	}

	signingKey := jwtSecret

	return signingKey, nil
}

func VerifyJWT(tokenString string) (*jwt.Token, error) {

	token, err := jwt.Parse(
		tokenString,
		getSigningKey,
	)

	if err != nil {
		return nil, err
	}

	return token, nil
}
