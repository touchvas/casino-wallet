package wallet

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func GenerateToken(redisConn *redis.Client, profileID string, ctx context.Context) string {

	token := uuid.New().String()
	SetRedisKeyWithExpiry(redisConn, token, profileID, 60*60*5, ctx)

	sessionKeys := fmt.Sprintf("session:%s", profileID)
	SetRedisKeyWithExpiry(redisConn, sessionKeys, token, 60*60*5, ctx)

	return token
}

func GetSessionID(redisConn *redis.Client, profileID string, ctx context.Context) string {

	sessionKeys := fmt.Sprintf("session:%s", profileID)
	profile, _ := GetRedisKey(redisConn, sessionKeys, ctx)
	return profile
}

func GetProfileIDFromtoken(redisConn *redis.Client, token string, ctx context.Context) string {

	profile, _ := GetRedisKey(redisConn, token, ctx)
	return profile
}
