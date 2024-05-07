package gosf

import (
	"time"

	"github.com/patrickmn/go-cache"
)

type Cache struct {
	*cache.Cache // 嵌入 cache.Cache
}

func NewCache(defaultExpiration, cleanupInterval time.Duration) *Cache {
	// 创建一个新的 cache.Cache 实例
	return &Cache{
		Cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

// Lock 缓存锁
func (p *Cache) Lock(key string, interval time.Duration, setLock bool) bool {
	key = "lock_" + key
	lock, found := p.Get(key)
	if found && lock == "1" {
		return true
	} else {
		if setLock {
			p.Set(key, "1", interval)
		}
		return false
	}
}

// UnLock 解锁
func (p *Cache) UnLock(key string) {
	key = "lock_" + key
	p.Set(key, "0", time.Duration(0)*time.Second)
}
