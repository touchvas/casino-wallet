package wallet

import (
	"context"
	"fmt"
	"os"

	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func getKey(key string) string {

	if len(os.Getenv("REDIS_KEY_PREFIX")) > 0 {

		return fmt.Sprintf("%s:%s", os.Getenv("REDIS_KEY_PREFIX"), key)
	}

	return key

}

func GetRedisKey(conn *redis.Client, key string, ctx context.Context) (string, error) {

	var data string
	data, err := conn.Get(ctx, getKey(key)).Result()
	if err != nil {

		return data, fmt.Errorf("error getting key %s: %v", key, err)
	}

	return data, err

}

func SetRedisKey(conn *redis.Client, key string, value string, ctx context.Context) error {

	_, err := conn.Set(ctx, getKey(key), value, time.Second*time.Duration(0)).Result()
	if err != nil {

		v := string(value)

		if len(v) > 15 {

			v = v[0:12] + "..."
		}

		return fmt.Errorf("error setting key %s to %s: %v", key, v, err)
	}
	return err
}

func SetRedisKeyWithExpiry(conn *redis.Client, key string, value string, seconds int, ctx context.Context) error {

	_, err := conn.Set(ctx, getKey(key), value, time.Second*time.Duration(seconds)).Result()
	if err != nil {

		v := string(value)

		if len(v) > 15 {

			v = v[0:12] + "..."
		}

		log.Printf("error saving redisKey %s error %s", key, err.Error())
		return fmt.Errorf("error setting key %s to %s: %v", key, v, err)
	}

	return err
}

func IncRedisKey(conn *redis.Client, key string, ctx context.Context) (int64, error) {

	var data int64
	data, err := conn.Incr(ctx, getKey(key)).Result()

	if err != nil {

		return data, fmt.Errorf("error getting key %s: %v", key, err)
	}

	return data, err
}
