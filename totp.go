package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func newSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return b32.EncodeToString(b)
}

func totpCode(secret string, step int64) string {
	key, err := b32.DecodeString(strings.ToUpper(strings.ReplaceAll(secret, " ", "")))
	if err != nil {
		return "------"
	}
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, uint64(step))
	m := hmac.New(sha1.New, key)
	m.Write(msg)
	h := m.Sum(nil)
	o := h[len(h)-1] & 0x0f
	bin := (uint32(h[o])&0x7f)<<24 | uint32(h[o+1])<<16 | uint32(h[o+2])<<8 | uint32(h[o+3])
	return fmt.Sprintf("%06d", bin%1000000)
}

func currentStep() int64 { return time.Now().Unix() / 30 }

// verifyLocked — одноразовая проверка кода (под stMu). Принимает коды за ~2 минуты.
func verifyLocked(secret, input string) bool {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, input)
	if secret == "" || len(digits) != 6 {
		return false
	}
	now := currentStep()
	for s := now - 4; s <= now+1; s++ {
		if s <= st.LastStep {
			continue
		}
		if hmac.Equal([]byte(totpCode(secret, s)), []byte(digits)) {
			st.LastStep = s
			return true
		}
	}
	return false
}

func prettyKey(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
