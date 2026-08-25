package service

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// IDGen 生成带前缀的唯一 ID（前缀-时间戳-随机），保证重启后仍可区分且不冲突。
type IDGen struct {
	mu sync.Mutex
	c  int64
}

// NewIDGen 构造 ID 生成器。
func NewIDGen() *IDGen { return &IDGen{} }

// New 生成形如 "<prefix>-<base36time>-<rand>" 的 ID。
func (g *IDGen) New(prefix string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.c++
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	ts := time.Now().UTC().Format("20060102T150405")
	return prefix + "-" + ts + "-" + hex.EncodeToString(buf) + "-" + itoa(g.c)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
