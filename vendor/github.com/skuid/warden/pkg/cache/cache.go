package cache

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

var (
	pool *redis.Pool
)

type CacheApi struct {
	conn redis.Conn
}

// New returns an API to the underlying cache implementation for a given connection
func New(c redis.Conn) CacheApi {
	return CacheApi{
		conn: c,
	}
}

func newPool(address string, maxConnections int) *redis.Pool {
	return &redis.Pool{
		MaxActive:   maxConnections,
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", address)
		},
	}
}

// GetConnection returns a connection from the Redis pool
// which should have been created via Start on server startup
func GetConnection() redis.Conn {
	return pool.Get()
}

// Start should be called once at server startup to initialize a pool
// for connections to Redis.
func Start(redisAddress string, maxConnections int) {
	pool = newPool(redisAddress, maxConnections)
}

// Conn returns the Redis connection associated with this CacheApi instance
func (api CacheApi) Conn() redis.Conn {
	return api.conn
}

// Get returns the value of a single string key in cache
func (api CacheApi) Get(key string) (string, error) {

	value, err := api.Conn().Do("GET", key)

	if err != nil {
		return "", err
	}

	value, err = redis.String(value, nil)

	if err != nil {
		return "", err
	}

	return value.(string), nil

}

// GetMap returns an object of all values stored in a cache value that is a hash map
func (api CacheApi) GetMap(key string) (map[string]string, error) {

	value, err := api.Conn().Do("HGETALL", key)

	if err != nil {
		return nil, err
	}

	value, err = redis.StringMap(value, nil)

	if err != nil {
		return nil, err
	}

	return value.(map[string]string), nil

}

// Set populates the value of a single string key in cache,
// and sets an expiration for the cache key (in seconds).
func (api CacheApi) Set(key string, value string, expirationSeconds int) (interface{}, error) {
	return api.Conn().Do("SET", key, value, expirationSeconds)
}

// SetMap takes a map of key-value pairs and populates this in a hash-map cache value,
// and sets an expiration for the cache key (in seconds).
func (api CacheApi) SetMap(key string, obj map[string]string, expirationSeconds int) (interface{}, error) {

	conn := api.Conn()

	conn.Send("MULTI")

	for field, value := range obj {
		conn.Send("HSET", key, field, value)
	}

	conn.Send("EXPIRE", key, expirationSeconds)

	return conn.Do("EXEC")

}
