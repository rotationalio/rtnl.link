package base62

import (
	"errors"
	"math"
	"strings"
)

const (
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length   = uint64(len(alphabet))
)

var (
	ErrOverflow = errors.New("input too long for uint64 decoding")
)

func Encode(num uint64) string {
	var buf strings.Builder
	buf.Grow(11)

	for ; num > 0; num = num / length {
		buf.WriteByte(alphabet[(num % length)])
	}

	return buf.String()
}

func Decode(s string) (uint64, error) {
	if len(s) > 11 {
		return 0, ErrOverflow
	}

	var num uint64
	for i, symbol := range s {
		pos := strings.IndexRune(alphabet, symbol)

		if pos == -1 {
			return 0, ErrInvalidCharacter(pos, symbol)
		}
		num += uint64(pos) * uint64(math.Pow(float64(length), float64(i)))
	}

	return num, nil
}
