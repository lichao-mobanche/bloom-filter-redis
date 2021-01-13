package bloom

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"sync"

	"github.com/go-redis/redis/v7"
)

// UnavailableBloomType bloom error
type UnavailableBloomType string

func (e UnavailableBloomType) Error() string {
	return fmt.Sprintf("unavailable type %s", string(e))
}

// BloomFilter holds all the storage filters.
type BloomFilter struct {
	key     string
	n       uint
	p       float64
	storage storage
	cache   *keyCache
	*sync.RWMutex
}

// NewBloomFilter creates and returns a new bloom filter using Redis as a backend.
// if you do not want to use cache, please set cacheSize to 0.
func NewBloomFilter(st StorageType, client *redis.Client, key string, n uint, p float64, cacheSize uint) (*BloomFilter, error) {

	bloom := &BloomFilter{key, n, p, nil, nil, &sync.RWMutex{}}
	if cacheSize != 0 {
		bloom.cache = NewKeyCache(cacheSize)
	}
	size, hashIter := EstimateParameters(n, p)
	partitionSize := math.Ceil(float64(size) / float64(hashIter))
	//st types should Synchronization with StorageType
	switch st {
	case Redis:
		var err error
		bloom.storage, err = NewRedisStorage(client, bloom, key, hashIter, uint(partitionSize))
		if err != nil {
			return nil, err
		}
	default:
		return nil, UnavailableBloomType(fmt.Sprint(st))
	}
	return bloom, nil
}

// Append is used to append a value to the queue.
func (b *BloomFilter) Append(value []byte) (err error) {
	b.RLock()
	defer b.RUnlock()
	if b.cache != nil {
		b.cache.Load(value)
	}
	return b.storage.Append(value)
}

// Exists checks if the given value is in the bloom filter or not. False positives might occur.
func (b *BloomFilter) Exists(value []byte) (exists bool, err error) {
	b.RLock()
	defer b.RUnlock()
	if b.cache != nil {
		if b.cache.Check(value) {
			return true, nil
		}
	}
	return b.storage.Exists(value)
}

// ExistsAndAppend check and append
func (b *BloomFilter) ExistsAndAppend(value []byte) (exists bool, err error) {
	b.RLock()
	defer b.RUnlock()
	if b.cache != nil {
		if b.cache.CheckAndLoad(value) {
			return true, nil
		}
	}
	
	return b.storage.ExistsAndAppend(value)
}

// Reset TODO
func (b *BloomFilter) Reset(n uint, p float64) (err error) {
	b.Lock()
	defer b.Unlock()
	size, hashIter := EstimateParameters(n, p)
	partitionSize := math.Ceil(float64(size) / float64(hashIter))
	err = b.storage.Init(hashIter, uint(partitionSize))
	if err == nil {
		b.cache.Reset()
	}
	return
}

func getOffset(hash1, hash2, iter, size uint) uint {
	return (hash1 + hash2*iter) % size
}

// hashValue takes care of hashing the value that is being stored in the bloom filter.
func hashValue(value *[]byte) (hash1, hash2 uint) {
	hasher := fnv.New64()
	hasher.Write(*value)
	sum := hasher.Sum(nil)

	hash1 = uint(binary.BigEndian.Uint32(sum[0:4]))
	hash2 = uint(binary.BigEndian.Uint32(sum[4:8]))

	return
}

func EstimateRate(m, k, count uint) (p float64) {
	//p = pow(1 - exp(-k / (m / count)), k)
	return math.Pow(1-math.Exp(-float64(k)/(float64(m)/float64(count))), float64(k))
}

func EstimateParameters(n uint, p float64) (uint, uint) {
	//m = ceil((n * log(p)) / log(1 / pow(2, log(2))))
	m := math.Ceil(float64(n) * math.Log(p) / math.Log(1.0/math.Pow(2.0, math.Ln2)))
	//k = round((m / n) * log(2));
	k := math.Ln2*m/float64(n) + 0.5

	return uint(m), uint(k)
}

func EstimateCount(n uint, p float64, limitP float64) uint {
	//m = ceil((n * log(p)) / log(1 / pow(2, log(2))))
	m := math.Ceil(float64(n) * math.Log(p) / math.Log(1.0/math.Pow(2.0, math.Ln2)))
	//k = round((m / n) * log(2));
	k := math.Ln2*m/float64(n) + 0.5
	//count = -(m/k)*log(1-pow(limitp,1/k))
	count := -(m / k) * math.Log(1-math.Pow(limitP, 1/k))
	return uint(count)
}
