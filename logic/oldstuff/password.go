package logic

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"math/big"
)

func (l *LogicData) getCryptRandKey(size int) string {
	out := ""
	large := big.NewInt(int64(1 << 60))
	large = large.Add(large, large)
	for len(out) < size {
		num, err := rand.Int(rand.Reader, large)
		if err != nil {
			panic("Error generating session key: " + err.Error())
		}
		out = fmt.Sprintf("%s%X", out, num)
	}

	if len(out) > size {
		out = out[:size]
	}
	return out
}

func (l *LogicData) hashPassword(pass string) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(l.PasswordSalt+pass)))
}

func (l *LogicData) generatePass() (string, error) {
	out := ""
	for len(out) < 20 {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(15)))
		if err != nil {
			return "", err
		}

		out = fmt.Sprintf("%s%X", out, num)
	}
	return out, nil
}
