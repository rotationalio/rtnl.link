package base62

import "fmt"

type InvalidCharacter struct {
	pos  int
	char rune
}

func ErrInvalidCharacter(pos int, char rune) error {
	return &InvalidCharacter{pos, char}
}

func (c *InvalidCharacter) Error() string {
	return fmt.Sprintf("invalid character %s at position %d", string(c.char), c.pos)
}
