package bloomfilter

import (
	"encoding/binary"
	"math"
	"sync"
)

type (
	BloomFilter struct {
		// m：max number of bits
		// k：the number of encryptor
		// n：the number of bits entered
		m, k, n int32

		// bitmap：len(bitmap) = m/32 + 1
		bitmap []int32

		// encryptor: encrypt data of type any to type int32
		encryptor []Encryptor

		// concurrent
		isConcurrent bool
		sync.RWMutex
	}
)

type (
	Encryptor interface {
		Encrypt(origin []byte) int32
	}
)

func NewLocalBloomService(m int32, encryptor []Encryptor, isConcurrent bool) *BloomFilter {
	if m <= 0 || len(encryptor) == 0 {
		return nil
	}

	return &BloomFilter{
		m:            m,
		k:            int32(len(encryptor)),
		bitmap:       make([]int32, m>>5+1),
		encryptor:    encryptor,
		isConcurrent: isConcurrent,
	}
}

func (bf *BloomFilter) M() int32 {
	return bf.m
}

func (bf *BloomFilter) N() int32 {
	return bf.n
}

func (bf *BloomFilter) K() int32 {
	return bf.k
}

// P = (1 - e^(-kn/m))^k , false positive probability
func (bf *BloomFilter) P() float64 {
	return math.Pow(1-math.Exp(float64(-bf.k*bf.n)/float64(bf.m)), float64(bf.k))
}

func (bf *BloomFilter) Bitmap() []int32 {
	if bf.isConcurrent {
		bf.RLock()
		defer bf.RUnlock()
	}

	bitmap := make([]int32, len(bf.bitmap))
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

func (bf *BloomFilter) Reset() []int32 {
	if bf.isConcurrent {
		bf.Lock()
		defer bf.Unlock()
	}

	oldBitmap := bf.bitmap
	bf.bitmap = make([]int32, bf.m>>5+1)
	bf.n = 0

	return oldBitmap
}

func (bf *BloomFilter) getOffsets(val []byte) []int32 {
	if bf == nil {
		return nil
	}

	origin := val
	var offsets = make([]int32, 0, bf.k)
	for _, e := range bf.encryptor {
		offset := e.Encrypt(origin) % bf.m
		offsets = append(offsets, offset)
		// add suffix to avoid getting the same offset after when use same encryptor
		origin = binary.AppendVarint(origin, int64(offset))
	}

	return offsets
}
