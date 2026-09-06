package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

var ctx = context.Background()

func newRedisSessionID() string {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}

func createRedisSession(user User) (string, error) {
	sessionID := newRedisSessionID()

	data, err := json.Marshal(user)
	if err != nil {
		return "", err
	}

	err = rdb.Set(
		ctx,
		"sess:"+sessionID,
		data,
		15*time.Minute,
	).Err()

	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func getRedisSession(sessionID string) (User, bool) {
	data, err := rdb.Get(
		ctx,
		"sess:"+sessionID,
	).Result()

	if err != nil {
		return User{}, false
	}

	var user User

	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return User{}, false
	}

	return user, true
}
