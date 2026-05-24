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
	entry uint64
}

type AtomicLimiter struct {
	Buckets [ArraySize]RateBucket
}

func hashIP(ip string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(ip); i++ {
		h ^= uint64(ip[i])
		h *= 1099511628211
	}
	return h
}

func identityKey(h uint64) uint32 {
	k := uint32(h >> 32)
	if k == 0 {
		k = 1
	}
	return k
}

func packBucket(id uint32, exp16 uint16, count uint16) uint64 {
	return uint64(id)<<32 | uint64(exp16)<<16 | uint64(count)
}

func isExpired(exp16 uint16, nowSec int64) bool {
	return uint16(nowSec)-exp16 < 32768
}

func (l *AtomicLimiter) Allow(ip string, limit int, windowSec int64) bool {
	h := hashIP(ip)
	id := identityKey(h)
	baseSlot := int(h & (ArraySize - 1))
	now := time.Now().Unix()
	newExp := uint16(now + windowSec)

	for probe := 0; probe < MaxProbes; probe++ {
		slot := (baseSlot + probe) & (ArraySize - 1)
		b := &l.Buckets[slot]

		for {
			cur := atomic.LoadUint64(&b.entry)
			curID := uint32(cur >> 32)
			exp16 := uint16(cur >> 16)
			count := uint16(cur)

			if curID == 0 || (curID != id && isExpired(exp16, now)) {
				next := packBucket(id, newExp, 1)
				if atomic.CompareAndSwapUint64(&b.entry, cur, next) {
					return true
				}
				continue
			}

			if curID != id {
				break
			}

			if isExpired(exp16, now) {
				next := packBucket(id, newExp, 1)
				if atomic.CompareAndSwapUint64(&b.entry, cur, next) {
					return true
				}
				continue
			}

			if count >= uint16(limit) {
				return false
			}
			if count == 65535 {
				return false
			}
			next := packBucket(id, exp16, count+1)
			if atomic.CompareAndSwapUint64(&b.entry, cur, next) {
				return true
			}
		}
	}

	return true
}

func NewLimiter(limit int, window time.Duration, errMsg string) fiber.Handler {
	l := &AtomicLimiter{}
	windowSec := int64(window.Seconds())
	if windowSec <= 0 {
		windowSec = 1
	}
	if windowSec > 32767 {
		windowSec = 32767
	}
	return func(c fiber.Ctx) error {
		if !l.Allow(c.IP(), limit, windowSec) {
			return c.Status(429).JSON(fiber.Map{"error": errMsg})
		}
		return c.Next()
	}
}
