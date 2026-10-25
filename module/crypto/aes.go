package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// AESGCMEncrypt 使用 AES-256-GCM 加密明文。
// key 经 SHA-256 派生为 32 字节，密文格式为 base64(nonce + ciphertext)，
// 与数据一同存储时自带随机 nonce，同一明文每次加密结果不同。
func AESGCMEncrypt(key, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	block, err := aes.NewCipher(aesKeyBytes(key))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESGCMDecrypt 解密 AESGCMEncrypt 生成的密文；空字符串原样返回。
func AESGCMDecrypt(key, ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(aesKeyBytes(key))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, data := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// aesKeyBytes 任意长度的 key 统一派生为 32 字节。
func aesKeyBytes(key string) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}
