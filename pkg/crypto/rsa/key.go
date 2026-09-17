package rsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// GenerateRSAKey 生成指定位数的 RSA 密钥对，默认 2048 位
func GenerateRSAKey(bits int) (*rsa.PrivateKey, error) {
	if bits <= 0 {
		bits = 2048
	}
	return rsa.GenerateKey(rand.Reader, bits)
}

// GenerateRSAPEM 生成 RSA 私钥并编码为 PKCS1 PEM 字符串
func GenerateRSAPEM(bits int) (string, error) {
	priv, err := GenerateRSAKey(bits)
	if err != nil {
		return "", fmt.Errorf("生成 RSA 密钥失败: %w", err)
	}
	return EncodePrivateKeyPEM(priv), nil
}

// EncodePrivateKeyPEM 将 RSA 私钥编码为 PKCS1 标准 PEM 格式
func EncodePrivateKeyPEM(key *rsa.PrivateKey) string {
	if key == nil {
		return ""
	}
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return string(pem.EncodeToMemory(block))
}

// EncodePublicKeyPEM 将 RSA 公钥编码为 PKIX 标准 PEM 格式
func EncodePublicKeyPEM(pub *rsa.PublicKey) (string, error) {
	if pub == nil {
		return "", errors.New("公钥指针不能为空")
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("导出 PKIX 公钥失败: %w", err)
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}
	return string(pem.EncodeToMemory(block)), nil
}

// ParsePrivateKeyPEM 解析 PKCS1 或 PKCS8 格式的 RSA 私钥 PEM 字符串
func ParsePrivateKeyPEM(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("私钥 PEM 解码失败: 无效的 PEM 格式")
	}

	// 优先尝试 PKCS1
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	// 兼容 PKCS8
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		if rsaKey, ok := parsedKey.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("私钥类型不匹配: 期望 RSA, 实际得到 %T", parsedKey)
	}

	return nil, fmt.Errorf("解析 RSA 私钥失败: %w", err)
}

// ParsePublicKeyPEM 解析 PKIX 格式的 RSA 公钥 PEM 字符串
func ParsePublicKeyPEM(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("公钥 PEM 解码失败: 无效的 PEM 格式")
	}

	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析 PKIX 公钥失败: %w", err)
	}

	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("公钥类型不匹配: 期望 RSA, 实际得到 %T", pubInterface)
	}
	return pubKey, nil
}
