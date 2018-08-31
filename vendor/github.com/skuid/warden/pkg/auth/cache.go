package auth

import (
	"time"
)

type userInfoCache struct {
	expiry   int64
	userInfo UserInfo
}

var authCache = make(map[string]*userInfoCache)

/*
CacheSet will take a token (key) and save off the userinfo for that for a
specified duration.
*/
func CacheSet(token string, userInfo UserInfo, d time.Duration) {
	authCache[token] = &userInfoCache{
		expiry:   time.Now().Add(d).UnixNano(),
		userInfo: userInfo,
	}
}

/*
CacheGet will get a cached UserInfo object for that token, and return nil if
there is nothing set there already or if it has expired
*/
func CacheGet(token string) UserInfo {
	item := authCache[token]
	if item != nil {
		ttl := item.expiry - time.Now().UnixNano()
		if ttl < 0 {
			return nil
		}
		return item.userInfo
	}
	return nil
}
