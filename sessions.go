package wallet

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

func GenerateToken(redisConn *redis.Client, profileID string) string {

	token := uuid.New().String()
	SetRedisKeyWithExpiry(redisConn, token, profileID, 60*60*5)

	sessionKeys := fmt.Sprintf("session:%s", profileID)
	SetRedisKeyWithExpiry(redisConn, sessionKeys, token, 60*60*5)

	return token
}

func GetSessionID(redisConn *redis.Client, profileID string) string {

	sessionKeys := fmt.Sprintf("session:%s", profileID)
	profile, _ := GetRedisKey(redisConn, sessionKeys)
	return profile
}

func GetProfileIDFromtoken(redisConn *redis.Client, token string) string {

	profile, _ := GetRedisKey(redisConn, token)
	return profile
}
