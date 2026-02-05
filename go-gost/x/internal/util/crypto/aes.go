package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// AESCrypto AES 加密器结构
type AESCrypto struct {
	key []byte
}

// NewAESCrypto 创建新的 AES 加密器
// secret: 用于生成密钥的密码字符串
func NewAESCrypto(secret string) (*AESCrypto, error) {
	if secret == "" {
		return nil, fmt.Errorf("密钥不能为空")
	}

	// 使用 SHA256 将密码转换为 32 字节密钥
	hash := sha256.Sum256([]byte(secret))
	
	return &AESCrypto{
		key: hash[:],
	}, nil
}

// Encrypt 加密数据
// data: 要加密的原始数据
// 返回: base64编码的加密数据
func (a *AESCrypto) Encrypt(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("待加密数据不能为空")
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return "", fmt.Errorf("创建 AES cipher 失败: %v", err)
	}

	// 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %v", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %v", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// 组合 nonce + ciphertext
	encrypted := append(nonce, ciphertext...)

	// 返回 base64 编码结果
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt 解密数据
// encryptedData: base64编码的加密数据
// 返回: 解密后的原始数据
func (a *AESCrypto) Decrypt(encryptedData string) ([]byte, error) {
	if encryptedData == "" {
		return nil, fmt.Errorf("加密数据不能为空")
	}

	// base64 解码
	encrypted, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("base64 解码失败: %v", err)
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %v", err)
	}

	// 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("加密数据长度不足")
	}

	// 分离 nonce 和 ciphertext
	nonce := encrypted[:nonceSize]
	ciphertext := encrypted[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	return plaintext, nil
}

// EncryptString 加密字符串并返回加密后的字符串
func (a *AESCrypto) EncryptString(data string) (string, error) {
	return a.Encrypt([]byte(data))
}

// DecryptString 解密字符串并返回解密后的字符串
func (a *AESCrypto) DecryptString(encryptedData string) (string, error) {
	plaintext, err := a.Decrypt(encryptedData)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
} 