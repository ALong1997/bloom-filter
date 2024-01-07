package bloomfilter

import (
	"encoding/binary"
	"sync"
	"sync/atomic"
)

type (
	BloomFilter struct {
		// m：max number of elements
		// k：the number of encryptor
		// n：the number of elements entered
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
		bitmap:       make([]int32, m/32+1),
		encryptor:    encryptor,
		isConcurrent: isConcurrent,
	}
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

	atomic.AddInt32(&bf.n, 1)
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
