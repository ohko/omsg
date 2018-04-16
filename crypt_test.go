package omsg

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func Test_newCrypt(t *testing.T) {
	c := newCrypt([]byte("1234567812345678"))
	text := "abcdefghijklmnopqrstuvwxyz"
	text2 := c.decrypt(c.encrypt([]byte(text)))
	fmt.Println(hex.Dump([]byte(text)))
	fmt.Println(hex.Dump(text2))
}
