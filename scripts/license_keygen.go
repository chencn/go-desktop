//go:build ignore

package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

// main 生成一对 Ed25519 授权密钥，并用 RawURL base64 输出。
// publicKey 用于发布/运行时校验配置，privateKey 只用于离线签发授权码。
func main() {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "生成密钥失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("publicKey=%s\n", base64.RawURLEncoding.EncodeToString(publicKey))
	fmt.Printf("privateKey=%s\n", base64.RawURLEncoding.EncodeToString(privateKey))
}
