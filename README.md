# KR - Key Rotation Package

KR is a robust and flexible Go package designed to manage and rotate keys or secrets, such as those used to sign JWTs. It's built with customization in mind, allowing users to implement their own strategies by leveraging the interfaces provided.

## Features

- **Flexible Key Rotation**: KR provides a smooth way to rotate keys or secrets, ensuring your application's security is always up-to-date.
- **Customizable**:
    - **Interfaces**: KR is designed to be adaptable to your needs. You can implement the provided interfaces to create a key rotation strategy that fits your application. Just by implementing one of thoses interfaces:`KeyGenerator` and `KeyStorage`.
    - **Hooks**: Enable the scheduling of triggers for events such as `OnStart`, `OnStop`, `BeforeRotation`, and `AfterRotation`. This feature empowers you to execute custom actions at crucial points in your application's lifecycle.
- **Easy to Use**: With a simple and intuitive API, KR is easy to integrate into your Go applications.

## Installation

To install the KR package, use the `go get` command:

```sh
go get -u github.com/zhaori96/kr
```

# Usage
Here's a comprehensive guide on how to effectively use the KR package in various scenarios:

## Basic Usage
A simple way to integrate KR into your application is by creating a rotator instance, starting it, and then fetching a key. Below is a basic example:

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

## Using the Global Instance
You can simplify the process by utilizing the global instance provided by KR:
```go
    kr.Start()
    defer kr.Stop()

    key, err := kr.GetKey()
    if err != nil {
        panic(err)
    }

    fmt.Printf("%v", key.ID)
```

## Custom Settings
For more control over the key rotation process, you can customize the rotator settings. Here are two approaches:

**Approach 1: Inline Settings**
```go
    settings := &kr.RotatorSettings{
        RotationKeyCount: 15,
        RotationInterval: kr.DefaultRotationInverval,
        KeyExpiration: kr.DefaultKeyExpiration,
        AutoClearExpiredKeys: false,
        KeyProvidingMode: kr.NonRepeatingCyclicKeyProvidingMode
    }

    rotator := kr.NewWithSettings(settings)
```

**Approach 2: Default Settings with Overrides**
```go
    settings := kr.DefaultRotatorSettings()
    settings.RotationKeyCount = 15
    settings.AutoClearExpiredKeys = false
    settings.KeyProvidingMode = kr.NonRepeatingCyclicKeyProvidingMode

    rotator := kr.New()
    rotator.SetSettings(settings)
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

### RedisKeyStorage Implementation
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

# KeyGenerator with RSA

The `KeyGenerator` interface defines the contract for generating keys, with a specific implementation using RSA as the key type. This interface provides a method, `Generate`, which creates a new RSA key pair. If the key pair cannot be generated, it returns an error.

### RSAKeyGenerator Implementation

Below is an example of implementing the `KeyGenerator` interface using RSA as the key type:

```go
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log"
	"os"
)

// RSAKeyGenerator implements the KeyGenerator interface using RSA as the key type.
type RSAKeyGenerator struct {
	KeySize int
}

// NewRSAKeyGenerator creates a new instance of RSAKeyGenerator with the specified key size.
func NewRSAKeyGenerator(keySize int) *RSAKeyGenerator {
	return &RSAKeyGenerator{
		KeySize: keySize,
	}
}

// Generate creates a new RSA key pair. If the key pair cannot be generated, it returns an error.
func (g *RSAKeyGenerator) Generate() (any, error) {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, g.KeySize)
	if err != nil {
		return nil, err
	}

	// Encode private key to PEM format
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}

	// Write private key to a file (for example purposes)
	privateKeyFile, err := os.Create("private_key.pem")
	if err != nil {
		return nil, err
	}
	defer privateKeyFile.Close()
	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		return nil, err
	}

	return privateKey, nil
}
```


# Contribution
We welcome and appreciate contributions from the community! If you find any issues, have new features to propose, or want to improve the documentation, feel free to contribute to the KR project.
