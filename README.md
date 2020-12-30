# bloom-filter-redis

## Install

You can get the library with ``go get``

```
go get -u github.com/lichao-mobanche/bloom-filter-redis
```

## Usage
bloom过滤器，支持缓存，支持redis存储后端，如需使用其他存储后端请实现 storage interface


```
package main

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/lichao-mobanche/bloom-filter-redis/bloom"
	"os"
)

var client *redis.Client = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "", // no password set
	DB:       0,  // use default DB
})

func main() {
	bf, err := bloom.NewBloomFilter(bloom.Redis, client, "bloomtestkey", 100000, 0.01, 10000)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	exist, err := bf.Exists([]byte("hello"))
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	fmt.Println("hello is ", exist)

	err = bf.Append([]byte("hello"))
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	fmt.Println("hello is ", exist)

	exist, err = bf.ExistsAndAppend([]byte("hello"))
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	fmt.Println("hello is ", exist)
}
```

## License
  MIT licensed.