package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

// VIOLATION: state outside State. Owner: t0-plredis; moves to PackageDBMng in phase 2.
var rdb *redis.Client

// VIOLATION: init chaining. Init 2 of the package loader tools: the redis
// client for localhost:6379, called from Go package init.
func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func Add(key string, value interface{}) error {
	err := rdb.Set(ctx, key, value, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

func Get(key string) (interface{}, error) {
	ret, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return ret, err
	}
	return ret, nil
}
