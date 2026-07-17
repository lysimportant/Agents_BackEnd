package verification

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"math/big"
	"net"
	"net/smtp"
	"strings"
	"time"

	"collector-backend/config"
	"github.com/redis/go-redis/v9"
)

type PasswordCodeService struct {
	redis *redis.Client
	email config.EmailConfig
	ttl   time.Duration
}

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

func (s *PasswordCodeService) Close() error {
	if s == nil || s.redis == nil {
		return nil
	}
	return s.redis.Close()
}

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
	code, err := generateCode()
	if err != nil {
		return fmt.Errorf("生成验证码失败")
	}
	if err := s.redis.Set(ctx, passwordCodeKey(userID), code, s.ttl).Err(); err != nil {
		return fmt.Errorf("写入验证码缓存失败")
	}
	if err := s.sendEmail(email, code); err != nil {
		_ = s.redis.Del(context.Background(), passwordCodeKey(userID)).Err()
		return fmt.Errorf("发送验证码失败")
	}
	return nil
}

func (s *PasswordCodeService) VerifyPasswordCode(ctx context.Context, userID int, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("请输入邮箱验证码")
	}
	if s == nil || s.redis == nil {
		return fmt.Errorf("验证码服务未初始化")
	}
	key := passwordCodeKey(userID)
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
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("清理验证码失败")
	}
	return nil
}

func (s *PasswordCodeService) emailReady() bool {
	return strings.TrimSpace(s.email.Host) != "" &&
		s.email.Port > 0 &&
		strings.TrimSpace(s.email.Username) != "" &&
		strings.TrimSpace(s.email.Password) != "" &&
		strings.TrimSpace(s.email.From) != ""
}

func (s *PasswordCodeService) sendEmail(to string, code string) error {
	address := net.JoinHostPort(s.email.Host, fmt.Sprintf("%d", s.email.Port))
	auth := smtp.PlainAuth("", s.email.Username, s.email.Password, s.email.Host)
	message := []byte(strings.Join([]string{
		fmt.Sprintf("From: %s", s.email.From),
		fmt.Sprintf("To: %s", to),
		"Subject: =?UTF-8?B?55So5oi35a+G56CB5L+u5pS55qCh6aqM56CB?=",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		fmt.Sprintf("您的密码修改验证码是：%s", code),
		fmt.Sprintf("验证码 %d 分钟内有效，请勿转发给他人。", int(s.ttl.Minutes())),
	}, "\r\n"))

	if s.email.Secure {
		conn, err := tls.Dial("tcp", address, &tls.Config{ServerName: s.email.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, s.email.Host)
		if err != nil {
			return err
		}
		defer client.Close()
		if err := client.Auth(auth); err != nil {
			return err
		}
		if err := client.Mail(s.email.From); err != nil {
			return err
		}
		if err := client.Rcpt(to); err != nil {
			return err
		}
		writer, err := client.Data()
		if err != nil {
			return err
		}
		if _, err := writer.Write(message); err != nil {
			_ = writer.Close()
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
		return client.Quit()
	}
	return smtp.SendMail(address, auth, s.email.From, []string{to}, message)
}

func generateCode() (string, error) {
	max := big.NewInt(1000000)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func passwordCodeKey(userID int) string {
	return fmt.Sprintf("collector:password-code:%d", userID)
}
