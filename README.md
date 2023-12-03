# KR - Key Rotation Package

KR is a robust and flexible Go package designed to manage and rotate keys or secrets, such as those used to sign JWTs. It's built with customization in mind, allowing users to implement their own strategies by leveraging the interfaces provided.

## Features

- **Flexible Key Rotation**: KR provides a seamless way to rotate keys or secrets, ensuring your application's security is always up-to-date.
- **Customizable**: KR is designed to be adaptable to your needs. You can implement the provided interfaces to create a key rotation strategy that fits your application.
- **Easy to Use**: With a simple and intuitive API, KR is easy to integrate into your Go applications.

## Installation

To install the KR package, use the `go get` command:

```sh
go get -u github.com/zhaori96/kr
```

## Usage

Here's a basic example of how to use the KR package:

```go
package main

import (
    "fmt"

    "github.com/zhaori96/kr"
)

func main() {
    rotator := kr.New()
    rotator.Start()
    defer rotator.Stop()

    key, err := rotator.GetKey()
    if err != nil {
        panic(err)
    }

    fmt.Printf("ID: %s; Value: %v; Expiration: %s", key.ID, key.Value, key.Expiration)
}
```

# Hooks
Hooks allow you to execute custom logic before or after key rotation events. Use OnStart and OnStop hooks to perform actions when the Rotator starts or stops, respectively. Additionally, you can use BeforeRotation and AfterRotation hooks to execute logic before or after each key rotation.

```go
// Example of using hooks
rotator.OnStart(func(r *kr.Rotator) {
	log.Println("Rotator is starting...")
})

rotator.OnStop(func(r *kr.Rotator) {
	log.Println("Rotator is stopping...")
})

rotator.BeforeRotation(func(r *kr.Rotator) {
	log.Println("Before key rotation...")
})

rotator.AfterRotation(func(r *kr.Rotator) {
	log.Println("After key rotation...")
})
```


# KeyStorage with Redis

The RedisKeyStorage struct provides an implementation of the KeyStorage interface using Redis as the backend.

###### Implementing
```go
import (
    "context"
    "encoding/json"

    "github.com/go-redis/redis/v8"
)

type RedisKeyStorage struct {
    client *redis.Client
}

func NewRedisKeyStorage(client *redis.Client) *RedisKeyStorage {
    return &RedisKeyStorage{client: client}
}

func (r *RedisKeyStorage) Get(ctx context.Context, id string) (*Key, error) {
    val, err := r.client.Get(ctx, id).Result()
    if err == redis.Nil {
        return nil, ErrKeyNotFound
    } else if err != nil {
        return nil, err
    }

    var key Key
    err = json.Unmarshal([]byte(val), &key)
    if err != nil {
        return nil, err
    }

    return &key, nil
}

func (r *RedisKeyStorage) Add(ctx context.Context, keys ...*Key) error {
    for _, key := range keys {
        val, err := json.Marshal(key)
        if err != nil {
            return err
        }

        err = r.client.Set(ctx, key.ID, val, 0).Err()
        if err != nil {
            return err
        }
    }

    return nil
}

func (r *RedisKeyStorage) Delete(ctx context.Context, ids ...string) error {
    for _, id := range ids {
        err := r.client.Del(ctx, id).Err()
        if err != nil {
            return err
        }
    }

    return nil
}

func (r *RedisKeyStorage) Erase(ctx context.Context) error {
    return r.client.FlushDB(ctx).Err()
}
```

###### Using custom KeyStorage with Redis
```go
package main

import (
    "github.com/zhaori96/kr"
)

func main(){
    rotator := kr.New()

    redisClient := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })

    storage := NewRedisKeyStorage(redisClient)
    err := rotator.SetStorage()
    if err != nil {
        panic(err)
    }

    rotator.Start()
    defer rotator.Stop()

    key, err := rotator.GetKey()
    if err != nil {
        panic(err)
    }

    fmt.Printf("ID: %s; Value: %v; Expiration: %s", key.ID, key.Value, key.Expiration)
}
```
