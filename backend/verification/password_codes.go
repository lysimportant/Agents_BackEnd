package verification

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"net/smtp"
	"strings"
	"time"

	"collector-backend/config"
	"github.com/redis/go-redis/v9"
)

// PasswordCodeService 定义对应业务的数据结构与调用契约。
type PasswordCodeService struct {
	// redis 表示变量 redis。
	redis *redis.Client
	// email 表示邮箱地址。
	email config.EmailConfig
	// ttl 表示有效期。
	ttl time.Duration
}

// NewPasswordCodeService 构造并返回对应业务实例。
func NewPasswordCodeService(cfg config.Config) *PasswordCodeService {
	return &PasswordCodeService{
		redis: redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddress,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		}),
		email: cfg.Email,
		ttl:   cfg.PasswordCodeTTL,
	}
}

// Close 删除或清理对应业务记录。
func (s *PasswordCodeService) Close() error {
	if s == nil || s.redis == nil {
		return nil
	}
	return s.redis.Close()
}

// SendPasswordCode 执行对应业务操作。
func (s *PasswordCodeService) SendPasswordCode(ctx context.Context, userID int, email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("当前账号未绑定邮箱")
	}
	if s == nil || s.redis == nil {
		return fmt.Errorf("验证码服务未初始化")
	}
	if !s.emailReady() {
		return fmt.Errorf("邮箱发送配置不完整")
	}
	return s.sendCode(ctx, passwordCodeKey(userID), email, "您的密码修改验证码是")
}

// SendRegistrationCode 发送注册邮箱验证码，不创建临时用户记录。
func (s *PasswordCodeService) SendRegistrationCode(ctx context.Context, username, email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("请输入邮箱地址")
	}
	if s == nil || s.redis == nil {
		return fmt.Errorf("验证码服务未初始化")
	}
	if !s.emailReady() {
		return fmt.Errorf("邮箱发送配置不完整")
	}
	return s.sendCode(ctx, registrationCodeKey(username, email), email, "您的注册验证码是")
}

// VerifyRegistrationCode 校验并消费注册邮箱验证码。
func (s *PasswordCodeService) VerifyRegistrationCode(ctx context.Context, username, email, code string) error {
	return s.verifyCode(ctx, registrationCodeKey(username, email), code)
}

// sendCode 生成验证码、写入缓存并发送邮件。
func (s *PasswordCodeService) sendCode(ctx context.Context, key, email, subject string) error {
	code, err := generateCode()
	if err != nil {
		return fmt.Errorf("生成验证码失败")
	}
	if err := s.redis.Set(ctx, key, code, s.ttl).Err(); err != nil {
		return fmt.Errorf("写入验证码缓存失败")
	}
	if err := s.sendEmail(email, code, subject); err != nil {
		_ = s.redis.Del(context.Background(), key).Err()
		return fmt.Errorf("发送验证码失败")
	}
	return nil
}

// VerifyPasswordCode 实现对应业务逻辑。
func (s *PasswordCodeService) VerifyPasswordCode(ctx context.Context, userID int, code string) error {
	return s.verifyCode(ctx, passwordCodeKey(userID), code)
}

// verifyCode 校验并消费指定缓存键中的验证码。
func (s *PasswordCodeService) verifyCode(ctx context.Context, key, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("请输入邮箱验证码")
	}
	if s == nil || s.redis == nil {
		return fmt.Errorf("验证码服务未初始化")
	}
	// stored、err 保存当前操作结果以及可能返回的错误状态。
	stored, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("验证码已过期，请重新获取")
	}
	if err != nil {
		return fmt.Errorf("读取验证码缓存失败")
	}
	if stored != code {
		return fmt.Errorf("验证码错误")
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("清理验证码失败")
	}
	return nil
}

// emailReady 实现对应业务逻辑。
func (s *PasswordCodeService) emailReady() bool {
	return strings.TrimSpace(s.email.Host) != "" &&
		s.email.Port > 0 &&
		strings.TrimSpace(s.email.Username) != "" &&
		strings.TrimSpace(s.email.Password) != "" &&
		strings.TrimSpace(s.email.From) != ""
}

// sendEmail 执行对应业务操作。
func (s *PasswordCodeService) sendEmail(to string, code string, subject string) error {
	// address 保存变量 address。
	address := net.JoinHostPort(s.email.Host, fmt.Sprintf("%d", s.email.Port))
	// auth 保存认证。
	auth := smtp.PlainAuth("", s.email.Username, s.email.Password, s.email.Host)
	// message 保存消息。
	message := []byte(strings.Join([]string{
		fmt.Sprintf("From: %s", s.email.From),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: =?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject))),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		fmt.Sprintf("%s：%s", subject, code),
		fmt.Sprintf("验证码 %d 分钟内有效，请勿转发给他人。", int(s.ttl.Minutes())),
	}, "\r\n"))

	if s.email.Secure {
		// conn、err 保存当前操作结果以及可能返回的错误状态。
		conn, err := tls.Dial("tcp", address, &tls.Config{ServerName: s.email.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		// client、err 保存当前操作结果以及可能返回的错误状态。
		client, err := smtp.NewClient(conn, s.email.Host)
		if err != nil {
			return err
		}
		defer client.Close()
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := client.Auth(auth); err != nil {
			return err
		}
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := client.Mail(s.email.From); err != nil {
			return err
		}
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := client.Rcpt(to); err != nil {
			return err
		}
		// writer、err 保存当前操作结果以及可能返回的错误状态。
		writer, err := client.Data()
		if err != nil {
			return err
		}
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := writer.Write(message); err != nil {
			_ = writer.Close()
			return err
		}
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := writer.Close(); err != nil {
			return err
		}
		return client.Quit()
	}
	return smtp.SendMail(address, auth, s.email.From, []string{to}, message)
}

// generateCode 实现对应业务逻辑。
func generateCode() (string, error) {
	// max 保存变量 max。
	max := big.NewInt(1000000)
	// value、err 保存当前操作结果以及可能返回的错误状态。
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

// passwordCodeKey 实现对应业务逻辑。
func passwordCodeKey(userID int) string {
	return fmt.Sprintf("collector:password-code:%d", userID)
}

// registrationCodeKey 生成与账号和邮箱绑定的注册验证码缓存键。
func registrationCodeKey(username, email string) string {
	identity := strings.ToLower(strings.TrimSpace(username)) + "\x00" + strings.ToLower(strings.TrimSpace(email))
	return fmt.Sprintf("collector:registration-code:%x", sha256.Sum256([]byte(identity)))
}
