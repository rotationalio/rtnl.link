package short

import (
	"encoding/binary"
	"net/url"

	"github.com/rotationalio/rtnl.link/pkg/base62"
	"github.com/twmb/murmur3"
)

func URL(rawURL string) (_ string, err error) {
	var u *url.URL
	if u, err = url.Parse(rawURL); err != nil {
		return "", err
	}
	return Shorten(u.String())
}

func Shorten(s string) (_ string, err error) {
	// Convert the string into a uint64 using a 64 bit murmur3 hash
	var sum []byte
	if sum, err = Hash([]byte(s)); err != nil {
		return "", err
	}

	var num uint64
	if num, err = Numeric(sum); err != nil {
		return "", err
	}

	return base62.Encode(num), nil
}

func Hash(s []byte) ([]byte, error) {
	hash := murmur3.New64()
	if _, err := hash.Write([]byte(s)); err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

func Numeric(s []byte) (uint64, error) {
	return binary.LittleEndian.Uint64(s), nil
}
