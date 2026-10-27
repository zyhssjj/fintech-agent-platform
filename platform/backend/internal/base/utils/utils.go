package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"wfagent/internal/base/response"
)

const defaultPageSize = 20

type PageQuery struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

type Snowflake struct {
	mu        sync.Mutex
	epoch     int64
	nodeID    int64
	sequence  int64
	lastMilli int64
}

var defaultSnowflake = NewSnowflake(1)

func MD5String(value string) string {
	sum := md5.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}

func MD5Reader(reader io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func AESEncrypt(plainText []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(normalizeAESKey(key))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nonce, nonce, plainText, nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func AESDecrypt(encoded string, key []byte) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(normalizeAESKey(key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(raw) < gcm.NonceSize() {
		return nil, errors.New("cipher text is too short")
	}
	nonce := raw[:gcm.NonceSize()]
	cipherText := raw[gcm.NonceSize():]
	return gcm.Open(nil, nonce, cipherText, nil)
}

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func ParseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", value)
}

func NormalizePage(page, pageSize int) (int, int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize, (page - 1) * pageSize
}

func Pagination(page, pageSize int, total int64) response.Pagination {
	page, pageSize, _ = NormalizePage(page, pageSize)
	return response.Pagination{Page: page, PageSize: pageSize, Total: total}
}

func Require(value string, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

func NewSnowflake(nodeID int64) *Snowflake {
	if nodeID < 0 {
		nodeID = 0
	}
	return &Snowflake{epoch: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(), nodeID: nodeID & 0x3ff}
}

func NextID() int64 {
	return defaultSnowflake.Next()
}

func (s *Snowflake) Next() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	if now < s.lastMilli {
		now = s.lastMilli
	}
	if now == s.lastMilli {
		s.sequence = (s.sequence + 1) & 0xfff
		if s.sequence == 0 {
			for now <= s.lastMilli {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}
	s.lastMilli = now
	return ((now - s.epoch) << 22) | (s.nodeID << 12) | s.sequence
}

func Placeholder(operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodDelete {
			response.NoContent(c)
			return
		}
		response.Success(c, gin.H{
			"operation": operation,
			"status":    "placeholder",
			"message":   "route contract reserved; implement business logic in the target module",
		})
	}
}

func normalizeAESKey(key []byte) []byte {
	normalized := make([]byte, 32)
	copy(normalized, key)
	return normalized
}
