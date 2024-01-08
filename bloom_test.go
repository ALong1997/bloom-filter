package bloomfilter

import (
	"hash"
	"hash/crc32"
	"hash/fnv"
	"math"
	"reflect"
	"testing"
)

type hash32 struct{ hash.Hash32 }

func (h *hash32) Encrypt(origin []byte) uint32 {
	defer h.Reset()
	h.Write(origin)
	return h.Sum32()
}

func TestOptimalK(t *testing.T) {
	type args struct {
		m    uint32
		maxN uint32
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{
			name: "TestOptimalK 1",
			args: args{
				m:    100000,
				maxN: 50000,
			},
			want: uint32(math.Ceil(float64(100000) * math.Ln2 / float64(50000))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OptimalK(tt.args.m, tt.args.maxN); got != tt.want {
				t.Errorf("OptimalK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOptimalM(t *testing.T) {
	type args struct {
		maxN uint32
		p    float64
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{
			name: "TestOptimalM 1",
			args: args{
				maxN: 50000,
				p:    0.0001,
			},
			want: uint32(math.Ceil(-float64(50000) * math.Log(0.0001) / math.Pow(math.Ln2, 2))),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OptimalM(tt.args.maxN, tt.args.p); got != tt.want {
				t.Errorf("OptimalM() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewBloomFilter(t *testing.T) {
	type args struct {
		m            uint32
		encryptor    []Encryptor
		isConcurrent bool
	}

	tests := []struct {
		name string
		args args
		want *BloomFilter
	}{
		{
			name: "TestNewBloomFilter 1",
			args: args{
				m:            100000,
				encryptor:    []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}},
				isConcurrent: false,
			},
			want: &BloomFilter{
				m:            100000,
				k:            2,
				bitmap:       make([]uint32, 100000>>5+1),
				encryptor:    []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}},
				isConcurrent: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBloomFilter(tt.args.m, tt.args.encryptor, tt.args.isConcurrent); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBloomFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBloomFilter_M(t *testing.T) {
	tests := []struct {
		name string
		bf   *BloomFilter
		want uint32
	}{
		{
			name: "TestBloomFilter_M 1",
			bf:   NewBloomFilter(100000, []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}}, false),
			want: 100000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.M(); got != tt.want {
				t.Errorf("M() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBloomFilter_N(t *testing.T) {
	bf := NewBloomFilter(100000, []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}}, false)
	tests := []struct {
		name string
		bf   *BloomFilter
		want uint32
	}{
		{
			name: "TestBloomFilter_N 1",
			bf:   bf,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.N(); got != tt.want {
				t.Errorf("N() = %v, want %v", got, tt.want)
			}
		})
	}

	bf.Set([]byte("hello world"))
	bf.Set([]byte("hello world!"))

	tests[0].want = 2
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.N(); got != tt.want {
				t.Errorf("N() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBloomFilter_K(t *testing.T) {
	tests := []struct {
		name string
		bf   *BloomFilter
		want uint32
	}{
		{
			name: "TestBloomFilter_K 1",
			bf:   NewBloomFilter(100000, []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}}, false),
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.K(); got != tt.want {
				t.Errorf("K() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBloomFilter_P(t *testing.T) {
	bf := NewBloomFilter(100000, []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}}, false)
	tests := []struct {
		name string
		bf   *BloomFilter
		want float64
	}{
		{
			name: "TestBloomFilter_P 1",
			bf:   bf,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.P(); got != tt.want {
				t.Errorf("P() = %v, want %v", got, tt.want)
			}
		})
	}

	bf.Set([]byte("hello world"))
	bf.Set([]byte("hello world!"))

	tests[0].want = math.Pow(1-math.Exp(float64(-bf.K()*bf.N())/float64(bf.M())), float64(bf.k))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.P(); got != tt.want {
				t.Errorf("P() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBloomFilter_Bitmap(t *testing.T) {
	bf := NewBloomFilter(100000, []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}}, false)
	tests := []struct {
		name string
		bf   *BloomFilter
		want []uint32
	}{
		{
			name: "TestBloomFilter_P 1",
			bf:   bf,
			want: make([]uint32, 100000>>5+1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.Bitmap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bitmap() = %v, want %v", got, tt.want)
			}
		})
	}

	bf.Set([]byte("hello world"))
	bf.Set([]byte("hello world!"))

	tests[0].want = bf.bitmap
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.Bitmap(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Bitmap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBloomFilter_Exist(t *testing.T) {
	bf := NewBloomFilter(100000, []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}}, false)
	type args struct {
		val []byte
	}
	tests := []struct {
		name string
		bf   *BloomFilter
		args args
		want bool
	}{
		{
			name: "TestBloomFilter_Exist 1",
			bf:   bf,
			args: args{[]byte("hello world")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.Exist(tt.args.val); got != tt.want {
				t.Errorf("Exist() = %v, want %v", got, tt.want)
			}
		})
	}

	bf.Set([]byte("hello world"))
	bf.Set([]byte("hello world!"))
	tests[0].want = true
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.Exist(tt.args.val); got != tt.want {
				t.Errorf("Exist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBloomFilter_Reset(t *testing.T) {
	bf := NewBloomFilter(100000, []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}}, false)
	tests := []struct {
		name string
		bf   *BloomFilter
		want []uint32
	}{
		{
			name: "TestBloomFilter_Reset 1",
			bf:   bf,
			want: make([]uint32, 100000>>5+1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.Reset(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reset() = %v, want %v", got, tt.want)
			}
		})
	}

	bf.Set([]byte("hello world"))
	bf.Set([]byte("hello world!"))
	tests[0].want = bf.Bitmap()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.bf.Reset(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBloomFilter_Set(t *testing.T) {
	bf := NewBloomFilter(100000, []Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}}, false)
	type args struct {
		val []byte
	}
	tests := []struct {
		name string
		bf   *BloomFilter
		args args
	}{
		{
			name: "TestBloomFilter_Set 1",
			bf:   bf,
			args: args{[]byte("hello world")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.bf.Set(tt.args.val)
		})
	}
}
