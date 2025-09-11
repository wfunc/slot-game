package utils

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// PasswordTestSuite 密码工具测试套件
type PasswordTestSuite struct {
	suite.Suite
}

// 测试密码哈希
func (suite *PasswordTestSuite) TestHashPassword() {
	password := "MySecurePassword123!"
	
	// 生成哈希
	hash, err := HashPassword(password)
	suite.NoError(err)
	suite.NotEmpty(hash)
	suite.NotEqual(password, hash) // 哈希不应该等于原始密码
	
	// 哈希应该是argon2id格式
	suite.True(strings.HasPrefix(hash, "$argon2"))
}

// 测试相同密码生成不同哈希
func (suite *PasswordTestSuite) TestHashPasswordUniqueness() {
	password := "SamePassword123"
	
	// 生成两个哈希
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)
	
	suite.NoError(err1)
	suite.NoError(err2)
	suite.NotEqual(hash1, hash2) // 相同密码应该生成不同的哈希（因为salt不同）
}

// 测试密码验证
func (suite *PasswordTestSuite) TestVerifyPassword() {
	password := "CorrectPassword456"
	hash, _ := HashPassword(password)
	
	// 验证正确的密码
	valid, err := VerifyPassword(password, hash)
	suite.NoError(err)
	suite.True(valid)
	
	// 验证错误的密码
	invalid, err := VerifyPassword("WrongPassword", hash)
	suite.NoError(err)
	suite.False(invalid)
	
	// 验证大小写敏感
	invalidCase, err := VerifyPassword("correctpassword456", hash)
	suite.NoError(err)
	suite.False(invalidCase)
}

// 测试使用自定义配置哈希密码
func (suite *PasswordTestSuite) TestHashPasswordWithConfig() {
	password := "CustomConfigPassword"
	
	// 使用自定义配置
	config := &PasswordConfig{
		Time:    2,
		Memory:  32 * 1024,
		Threads: 2,
		KeyLen:  16,
	}
	
	hash, err := HashPasswordWithConfig(password, config)
	suite.NoError(err)
	suite.NotEmpty(hash)
	
	// 验证密码
	valid, err := VerifyPassword(password, hash)
	suite.NoError(err)
	suite.True(valid)
}

// 测试空密码
func (suite *PasswordTestSuite) TestEmptyPassword() {
	// 哈希空密码
	hash, err := HashPassword("")
	suite.NoError(err)
	suite.NotEmpty(hash)
	
	// 验证空密码
	valid, err := VerifyPassword("", hash)
	suite.NoError(err)
	suite.True(valid)
	
	// 非空密码不应该匹配空密码的哈希
	invalid, err := VerifyPassword("notEmpty", hash)
	suite.NoError(err)
	suite.False(invalid)
}

// 测试长密码
func (suite *PasswordTestSuite) TestLongPassword() {
	// argon2id没有密码长度限制
	longPassword := strings.Repeat("a", 1000)
	
	hash, err := HashPassword(longPassword)
	suite.NoError(err)
	suite.NotEmpty(hash)
	
	// 验证长密码
	valid, err := VerifyPassword(longPassword, hash)
	suite.NoError(err)
	suite.True(valid)
}

// 测试特殊字符密码
func (suite *PasswordTestSuite) TestSpecialCharacterPassword() {
	passwords := []string{
		"P@$$w0rd!",
		"密码123",
		"🔐Security🔒",
		"Tab\tSpace New\nLine",
		"Quote'Double\"Quote",
	}
	
	for _, password := range passwords {
		hash, err := HashPassword(password)
		suite.NoError(err)
		suite.NotEmpty(hash)
		
		valid, err := VerifyPassword(password, hash)
		suite.NoError(err)
		suite.True(valid, "密码 %s 应该验证成功", password)
	}
}

// 测试无效哈希验证
func (suite *PasswordTestSuite) TestVerifyPasswordWithInvalidHash() {
	// 完全无效的哈希
	valid, err := VerifyPassword("password", "invalid-hash")
	suite.Error(err)
	suite.False(valid)
	
	// 空哈希
	valid, err = VerifyPassword("password", "")
	suite.Error(err)
	suite.False(valid)
	
	// 格式错误的argon2哈希
	valid, err = VerifyPassword("password", "$argon2$invalid$format")
	suite.Error(err)
	suite.False(valid)
}

// 测试生成随机字符串
func (suite *PasswordTestSuite) TestGenerateRandomString() {
	// 测试不同长度
	lengths := []int{8, 16, 24, 32, 64}
	
	for _, length := range lengths {
		str, err := GenerateRandomString(length)
		suite.NoError(err)
		suite.Equal(length, len(str), "生成的字符串长度应该为 %d", length)
		
		// 验证是否只包含base64 URL安全字符
		for _, char := range str {
			isValid := (char >= 'A' && char <= 'Z') ||
				(char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9') ||
				char == '-' || char == '_'
			suite.True(isValid, "字符 %c 不是有效的base64 URL字符", char)
		}
	}
}

// 测试生成随机字符串的唯一性
func (suite *PasswordTestSuite) TestGenerateRandomStringUniqueness() {
	generated := make(map[string]bool)
	
	// 生成100个字符串
	for i := 0; i < 100; i++ {
		str, err := GenerateRandomString(16)
		suite.NoError(err)
		suite.False(generated[str], "不应该生成重复的字符串")
		generated[str] = true
	}
}

// 测试生成会话ID
func (suite *PasswordTestSuite) TestGenerateSessionID() {
	sessionID, err := GenerateSessionID()
	suite.NoError(err)
	suite.NotEmpty(sessionID)
	suite.Equal(32, len(sessionID), "会话ID应该是32个字符")
	
	// 验证唯一性
	sessionID2, err := GenerateSessionID()
	suite.NoError(err)
	suite.NotEqual(sessionID, sessionID2)
}

// 测试生成验证码
func (suite *PasswordTestSuite) TestGenerateVerificationCode() {
	code, err := GenerateVerificationCode()
	suite.NoError(err)
	suite.NotEmpty(code)
	suite.Equal(6, len(code), "验证码应该是6位")
	
	// 验证是否只包含数字
	for _, char := range code {
		suite.True(char >= '0' && char <= '9', "验证码应该只包含数字")
	}
}

// 测试并发密码哈希
func (suite *PasswordTestSuite) TestConcurrentPasswordHashing() {
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			password := fmt.Sprintf("Password%d", id)
			hash, err := HashPassword(password)
			suite.NoError(err)
			suite.NotEmpty(hash)
			
			valid, err := VerifyPassword(password, hash)
			suite.NoError(err)
			suite.True(valid)
			
			done <- true
		}(i)
	}
	
	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// 测试密码加盐
func (suite *PasswordTestSuite) TestPasswordSalt() {
	password := "TestPassword"
	
	// 生成多个哈希
	hashes := make([]string, 5)
	for i := range hashes {
		hash, err := HashPassword(password)
		suite.NoError(err)
		hashes[i] = hash
	}
	
	// 验证所有哈希都不同（因为salt不同）
	for i := 0; i < len(hashes); i++ {
		for j := i + 1; j < len(hashes); j++ {
			suite.NotEqual(hashes[i], hashes[j], "哈希应该因为不同的salt而不同")
		}
	}
	
	// 但所有哈希都应该能验证原始密码
	for _, hash := range hashes {
		valid, err := VerifyPassword(password, hash)
		suite.NoError(err)
		suite.True(valid)
	}
}

// 测试默认配置
func (suite *PasswordTestSuite) TestDefaultPasswordConfig() {
	// 直接使用HashPassword，它会使用默认配置
	password := "DefaultConfig"
	hash, err := HashPassword(password)
	suite.NoError(err)
	suite.NotEmpty(hash)
	
	// 验证密码
	valid, err := VerifyPassword(password, hash)
	suite.NoError(err)
	suite.True(valid)
}

// 测试argon2id格式
func (suite *PasswordTestSuite) TestArgon2IDFormat() {
	password := "TestFormat"
	hash, err := HashPassword(password)
	suite.NoError(err)
	
	// 验证格式：$argon2id$v=19$m=65536,t=3,p=4$...
	suite.True(strings.HasPrefix(hash, "$argon2id$"))
	suite.Contains(hash, "v=")
	suite.Contains(hash, "m=")
	suite.Contains(hash, "t=")
	suite.Contains(hash, "p=")
}

// 测试边界情况
func (suite *PasswordTestSuite) TestEdgeCases() {
	// 零长度随机字符串
	str, err := GenerateRandomString(0)
	suite.NoError(err)
	suite.Empty(str)
	
	// 非常大的长度应该成功
	str, err = GenerateRandomString(1024)
	suite.NoError(err)
	suite.Equal(1024, len(str))
}

func TestPasswordSuite(t *testing.T) {
	suite.Run(t, new(PasswordTestSuite))
}