# Bloom Filter

This is a simple and thread-safe bloom filter implemented by [Go](https://go.dev/).

## Convenience functions

Other than the classic `Exist`, `Set` and `Reset`, some more estimation functions are implemented that makes this bloom filter implementation very easy and straight forward to use
in real applications.

## Getting started

### Prerequisites
- **[Go](https://go.dev/) version 1.18+**

### Getting
With [Go module](https://github.com/golang/go/wiki/Modules) support, simply add the following import

```
import "github.com/ALong1997/bloomfilter"
```

Otherwise, run the following Go command to install the `bloomfilter` package:

```sh
$ go get -u https://github.com/ALong1997/bloomfilter
```

### Quick Start

```go
package main

import (
    "fmt"
	"hash"
	"hash/crc32"
	"hash/fnv"
	
	"github.com/ALong1997/bloomfilter"
)

type hash32 struct{ hash.Hash32 }

func (h *hash32) Encrypt(origin []byte) uint32 {
	defer h.Reset()
	h.Write(origin)
	return h.Sum32()
}

func main() {
	bf := bloomfilter.NewBloomFilter(100000, []bloomfilter.Encryptor{&hash32{crc32.NewIEEE()}, &hash32{fnv.New32()}}, false)
	
	bf.Set([]byte("hello world"))
	bf.Set([]byte("hello world!"))
    
    // true
    fmt.Println(bf.Exist([]byte("hello world")))
	// false
    fmt.Println(bf.Exist([]byte("bloom filter")))
    
    // 100000
    fmt.Println(bf.M())
    // 2
    fmt.Println(bf.N())
    // 2
    fmt.Println(bf.K())
    
    bf.Reset()
}

```
