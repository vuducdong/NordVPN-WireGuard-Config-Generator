package middleware

import (
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	ArraySize = 16384
	MaxProbes = 4
)

type RateBucket struct {
	IPHash uint64
	State  uint64
}

type AtomicLimiter struct {
	Buckets [ArraySize]RateBucket
}

func hashIP(ip string) uint64 {
	var hash uint64 = 14695981039346656037
	for i := 0; i < len(ip); i++ {
		hash ^= uint64(ip[i])
		hash *= 1099511628211
	}
	return hash
}

func (l *AtomicLimiter) Allow(ip string, limit int, window int64) bool {
	hash := hashIP(ip)
	now := time.Now().Unix()
	baseSlot := int(hash & (ArraySize - 1))

	for i := 0; i < MaxProbes; i++ {
		slot := (baseSlot + i) & (ArraySize - 1)
		b := &l.Buckets[slot]

		currentIP := atomic.LoadUint64(&b.IPHash)
		if currentIP == 0 {
			if atomic.CompareAndSwapUint64(&b.IPHash, 0, hash) {
				expiry := now + window
				state := (uint64(expiry) << 16) | 1
				atomic.StoreUint64(&b.State, state)
				return true
			}
			currentIP = atomic.LoadUint64(&b.IPHash)
		}

		if currentIP != hash {
			state := atomic.LoadUint64(&b.State)
			expiry := int64(state >> 16)
			if now > expiry {
				if atomic.CompareAndSwapUint64(&b.IPHash, currentIP, hash) {
					expiry := now + window
					state := (uint64(expiry) << 16) | 1
					atomic.StoreUint64(&b.State, state)
					return true
				}
			}
			continue
		}

		for {
			state := atomic.LoadUint64(&b.State)
			expiry := int64(state >> 16)
			count := uint16(state)

			var nextState uint64
			if now > expiry {
				nextExpiry := now + window
				nextState = (uint64(nextExpiry) << 16) | 1
			} else {
				if count == 65535 {
					return false
				}
				nextState = (uint64(expiry) << 16) | uint64(count+1)
			}

			if atomic.CompareAndSwapUint64(&b.State, state, nextState) {
				if now <= expiry && count >= uint16(limit) {
					return false
				}
				return true
			}
		}
	}

	slot := baseSlot
	b := &l.Buckets[slot]
	currentIP := atomic.LoadUint64(&b.IPHash)
	if atomic.CompareAndSwapUint64(&b.IPHash, currentIP, hash) {
		expiry := now + window
		state := (uint64(expiry) << 16) | 1
		atomic.StoreUint64(&b.State, state)
		return true
	}

	return true
}

func NewLimiter(limit int, window time.Duration, errMsg string) fiber.Handler {
	l := &AtomicLimiter{}
	windowSeconds := int64(window.Seconds())
	if windowSeconds <= 0 {
		windowSeconds = 1
	}
	return func(c fiber.Ctx) error {
		ip := c.IP()
		if !l.Allow(ip, limit, windowSeconds) {
			return c.Status(429).JSON(fiber.Map{"error": errMsg})
		}
		return c.Next()
	}
}
