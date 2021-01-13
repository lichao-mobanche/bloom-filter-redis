package bloom

import (
	"crypto/md5"
	"sync"
)

// BucketNumber is the number of cache buckets
const BucketNumber = 1 << 8

type safeMap struct {
	inerMap map[[15]byte]struct{}
	*sync.RWMutex
}

// NewKeyCache creates a new cache
func NewKeyCache(cacheSize uint) *keyCache {

	kc := &keyCache{
		buckets:    make([]*safeMap, BucketNumber, BucketNumber),
		bucketSize: cacheSize / uint(BucketNumber),
	}
	for i := 0; i < BucketNumber; i++ {
		kc.buckets[i] = &safeMap{
			make(map[[15]byte]struct{}, kc.bucketSize),
			&sync.RWMutex{},
		}
	}
	return kc
}

type keyCache struct {
	buckets    []*safeMap
	bucketSize uint
}

func (kc *keyCache) Reset() {
	for i := 0; i < BucketNumber; i++ {
		bucket := kc.buckets[i]
		bucket.Lock()
		bucket.inerMap = make(map[[15]byte]struct{}, kc.bucketSize)
		bucket.Unlock()
	}
}

func (kc *keyCache) Load(value []byte) {
	if len(value) == 0 {
		return
	}
	m := md5.Sum([]byte(value))
	index := int(m[15])
	var key [15]byte
	keySlice := key[:]
	copy(keySlice, m[:15])
	bucket := kc.buckets[index]
	bucket.Lock()
	if uint(len(bucket.inerMap)) >= kc.bucketSize {
		bucket.inerMap = make(map[[15]byte]struct{}, kc.bucketSize)
	}
	bucket.inerMap[key] = struct{}{}
	bucket.Unlock()
}

func (kc *keyCache) Check(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	bucket, key := kc.getBucketKey(value)
	bucket.RLock()
	_, ok := bucket.inerMap[key]
	bucket.RUnlock()
	return ok
}

func (kc *keyCache) CheckAndLoad(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	bucket, key := kc.getBucketKey(value)
	bucket.RLock()
	_, ok := bucket.inerMap[key]
	bucket.RUnlock()
	if ok {
		return true
	}
	bucket.Lock()
	_, ok = bucket.inerMap[key]
	if !ok {
		if uint(len(bucket.inerMap)) >= kc.bucketSize {
			bucket.inerMap = make(map[[15]byte]struct{}, kc.bucketSize)
		}
		bucket.inerMap[key] = struct{}{}
	}
	bucket.Unlock()
	return ok
}

func (kc *keyCache) Remove(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	bucket, key := kc.getBucketKey(value)
	bucket.Lock()
	_, ok := bucket.inerMap[key]
	if ok {
		delete(bucket.inerMap, key)
	}
	bucket.Unlock()
	return ok
}

func (kc keyCache) getBucketKey(value []byte) (*safeMap, [15]byte) {
	m := md5.Sum(value)
	index := int(m[15])
	var key [15]byte
	keySlice := key[:]
	copy(keySlice, m[:15])
	bucket := kc.buckets[index]
	return bucket, key
}
