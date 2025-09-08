package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// PasswordConfig Argon2配置
type PasswordConfig struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

// DefaultPasswordConfig 默认配置
var DefaultPasswordConfig = &PasswordConfig{
	Time:    1,
	Memory:  64 * 1024,
	Threads: 4,
	KeyLen:  32,
}

// HashPassword 哈希密码
func HashPassword(password string) (string, error) {
	return HashPasswordWithConfig(password, DefaultPasswordConfig)
}

// HashPasswordWithConfig 使用指定配置哈希密码
func HashPasswordWithConfig(password string, config *PasswordConfig) (string, error) {
	// 生成随机盐
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	
	// 使用Argon2id哈希密码
	hash := argon2.IDKey([]byte(password), salt, config.Time, config.Memory, config.Threads, config.KeyLen)
	
	// 编码为字符串格式: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, config.Memory, config.Time, config.Threads, b64Salt, b64Hash)
	
	return encoded, nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, encoded string) (bool, error) {
	// 解析编码的哈希值
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid encoded hash format")
	}
	
	if parts[1] != "argon2id" {
		return false, fmt.Errorf("unsupported hash algorithm")
	}
	
	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, err
	}
	
	if version != argon2.Version {
		return false, fmt.Errorf("incompatible argon2 version")
	}
	
	config := &PasswordConfig{}
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&config.Memory, &config.Time, &config.Threads)
	if err != nil {
		return false, err
	}
	
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	
	config.KeyLen = uint32(len(hash))
	
	// 重新计算哈希值
	comparisonHash := argon2.IDKey([]byte(password), salt,
		config.Time, config.Memory, config.Threads, config.KeyLen)
	
	// 使用恒定时间比较
	if subtle.ConstantTimeCompare(hash, comparisonHash) == 1 {
		return true, nil
	}
	
	return false, nil
}

// GenerateRandomString 生成随机字符串（用于token等）
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateSessionID 生成会话ID
func GenerateSessionID() (string, error) {
	return GenerateRandomString(32)
}

// GenerateVerificationCode 生成验证码
func GenerateVerificationCode() (string, error) {
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	code := int(bytes[0])<<16 | int(bytes[1])<<8 | int(bytes[2])
	return fmt.Sprintf("%06d", code%1000000), nil
}