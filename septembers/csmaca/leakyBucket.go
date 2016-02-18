package csmaca

import (
	"sync/atomic"
	"time"
)

// Thread-safe leaky bucket
type leakyBucket struct {
	bucketSize        int64
	waterDropInterval time.Duration
	waterDropSize     int64

	bucket  int64
	started bool
}

func NewLeakyBucket(bucketSize int, waterDropInterval time.Duration, waterDropSize int) *leakyBucket {
	return &leakyBucket{
		bucketSize:        int64(bucketSize),
		waterDropInterval: waterDropInterval,
		waterDropSize:     int64(waterDropSize),
	}
}

func (this *leakyBucket) In(size int64) bool {
	if atomic.LoadInt64(&this.bucket) > this.bucketSize {
		return false
	}
	atomic.AddInt64(&this.bucket, size)
	return true
}

func (b *leakyBucket) Go() {
	if !b.started {
		b.started = true
		go func() {
			ticker := time.NewTicker(b.waterDropInterval)
			for _ = range ticker.C {
				if atomic.LoadInt64(&b.bucket) > 0 {
					atomic.AddInt64(&b.bucket, -b.waterDropSize)
				}
			}
		}()
	} else {
		panic("leaky bucket Go() called more than once")
	}
}

func (b *leakyBucket) Usage() float64 {
	return float64(atomic.LoadInt64(&b.bucket)) / float64(b.bucketSize)
}
