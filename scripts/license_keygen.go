//go:build ignore

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "生成密钥失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("publicKey=%s\n", base64.RawURLEncoding.EncodeToString(publicKey))
	fmt.Printf("privateKey=%s\n", base64.RawURLEncoding.EncodeToString(privateKey))
}
