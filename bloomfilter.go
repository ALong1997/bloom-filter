package bloomfilter

import (
	"encoding/binary"
	"math"
	"sync"
)

type (
	BloomFilter struct {
		// m：the number of bits
		// k：the number of encryptor
		// n：the number of inserted bits
		m, k, n uint32

		// bitmap：len(bitmap) = m/32 + 1
		bitmap []uint32

		// encryptor: encrypt data of type any to type uint32
		encryptor []Encryptor

		// concurrent
		isConcurrent bool
		sync.RWMutex
	}
)

type (
	Encryptor interface {
		Encrypt(origin []byte) uint32
	}
)

// OptimalK calculates the optimal k value for creating a new Bloom filter
// maxN is the maximum anticipated number of elements
// optimal k = ceiling( m * ln2 / n )
func OptimalK(m, maxN uint32) uint32 {
	return uint32(math.Ceil(float64(m) * math.Ln2 / float64(maxN)))
}

// OptimalM calculates the optimal m value for creating a new Bloom filter
// maxN is the maximum anticipated number of elements
// p is the desired false positive probability
// optimal m = ceiling( - n * ln(p) / (ln2)^2 )
func OptimalM(maxN uint32, p float64) uint32 {
	return uint32(math.Ceil(-float64(maxN) * math.Log(p) / math.Pow(math.Ln2, 2)))
}

func NewBloomFilter(m uint32, encryptor []Encryptor, isConcurrent bool) *BloomFilter {
	if m == 0 || len(encryptor) == 0 {
		return nil
	}

	return &BloomFilter{
		m:            m,
		k:            uint32(len(encryptor)),
		bitmap:       make([]uint32, m>>5+1),
		encryptor:    encryptor,
		isConcurrent: isConcurrent,
	}
}

func (bf *BloomFilter) M() uint32 {
	if bf == nil {
		return 0
	}
	return bf.m
}

func (bf *BloomFilter) N() uint32 {
	if bf == nil {
		return 0
	}
	return bf.n
}

func (bf *BloomFilter) K() uint32 {
	if bf == nil {
		return 0
	}
	return bf.k
}

// P returns false positive probability
// P = (1 - e^(-kn/m))^k
func (bf *BloomFilter) P() float64 {
	if bf == nil {
		return 0
	}
	return math.Pow(1-math.Exp(float64(-bf.k*bf.n)/float64(bf.m)), float64(bf.k))
}

func (bf *BloomFilter) Bitmap() []uint32 {
	if bf.isConcurrent {
		bf.RLock()
		defer bf.RUnlock()
	}

	bitmap := make([]uint32, len(bf.bitmap))
	copy(bitmap, bf.bitmap)

	return bitmap
}

func (bf *BloomFilter) Exist(val []byte) bool {
	if bf == nil {
		return false
	}

	offsets := bf.getOffsets(val)

	if bf.isConcurrent {
		bf.RLock()
		defer bf.RUnlock()
	}

	for _, offset := range offsets {
		if bf.bitmap[offset>>5]&(1<<(offset&31)) == 0 {
			// bf.bitmap[offset / 32]&(1<<(offset % 32))
			return false
		}
	}

	return true
}

func (bf *BloomFilter) Set(val []byte) {
	offsets := bf.getOffsets(val)

	if bf.isConcurrent {
		bf.Lock()
		defer bf.Unlock()
	}

	for _, offset := range offsets {
		// bf.bitmap[offset / 32] |= 1<<(offset % 32)
		bf.bitmap[offset>>5] |= 1 << (offset & 31)
	}

	bf.n++
}

func (bf *BloomFilter) Reset() []uint32 {
	if bf.isConcurrent {
		bf.Lock()
		defer bf.Unlock()
	}

	oldBitmap := bf.bitmap
	bf.bitmap = make([]uint32, bf.m>>5+1)
	bf.n = 0

	return oldBitmap
}

func (bf *BloomFilter) getOffsets(val []byte) []uint32 {
	if bf == nil {
		return nil
	}

	origin := val
	var offsets = make([]uint32, 0, bf.k)
	for _, e := range bf.encryptor {
		offset := e.Encrypt(origin) % bf.m
		offsets = append(offsets, offset)
		// add suffix to avoid getting the same offset after when use same encryptor
		origin = binary.AppendVarint(origin, int64(offset))
	}

	return offsets
}
