package main

import (
    	"crypto/rand"
    	"encoding/base64"
	"time"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var jwtSecret = []byte("super-secret-key")

func GenerateRefreshToken() (string, error) {
    tokenBytes := make([]byte, 32)

    _, err := rand.Read(tokenBytes)
    if err != nil {
        return "", err
    }

    token := base64.RawURLEncoding.EncodeToString(tokenBytes)

    return token, nil
}


func CreateJWT(user User) (string, error) {

	jti := uuid.New().String()

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"user_name":user.Username,
		"role":     user.Role,
		"exp": 	    time.Now().Add(1 * time.Minute).Unix(),
		"jti":	    jti,
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	return token.SignedString(jwtSecret)
}

func getSigningKey(token *jwt.Token) (interface{}, error) {

/*	fmt.Printf("TOKEN: %+v\n", token)

	fmt.Printf("METHOD: %+v\n", token.Method)
	fmt.Printf("HEADER: %+v\n", token.Header)
	fmt.Printf("CLAIMS: %+v\n", token.Claims)
	fmt.Printf("SIGNATURE: %+v\n", token.Signature)
	fmt.Printf("VALID: %+v\n", token.Valid)
*/
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
