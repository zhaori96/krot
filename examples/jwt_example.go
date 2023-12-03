// examples/jwt_hook.go
package main

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/zhaori96/kr/pkg/kr"
)

// JWTKeySigningHook is a custom RotatorHook that signs JWTs using the current key.
var JWTKeySigningHook kr.RotatorHook = func(rotator *kr.Rotator) {
	// Retrieve the current key from the Rotator
	key, err := rotator.GetKey()
	if err != nil {
		fmt.Printf("Error getting key for signing JWT: %v\n", err)
		return
	}

	// Example claims for the JWT
	claims := jwt.MapClaims{
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
		// Add your custom claims here
	}

	// Create a new JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the current key's value
	tokenString, err := token.SignedString(key.Value)
	if err != nil {
		fmt.Printf("Error signing JWT: %v\n", err)
		return
	}

	// Use the signed JWT as needed (e.g., include it in API responses)
	fmt.Printf("Signed JWT: %s\n", tokenString)
}

func main() {
	// Initialize the Rotator
	rotator := kr.New()

	// Add the custom JWT signing hook to the AfterRotation hooks
	rotator.AfterRotation(JWTKeySigningHook)

	// Start the Rotator
	err := rotator.Start()
	if err != nil {
		fmt.Printf("Error starting rotator: %v\n", err)
		return
	}
	defer rotator.Stop()

	// Keep the program running to observe key rotations and JWT signings
	select {}
}
