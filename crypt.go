package omsg

import (
	"crypto/aes"
	"crypto/cipher"
	"log"
)

type crypt struct {
	c cipher.Block
}

// 16, 24, or 32 bytes to select
// AES-128, AES-192, or AES-256.
func newCrypt(key []byte) *crypt {
	c, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalln(err)
	}
	return &crypt{c: c}
}

func (o *crypt) Encrypt(data []byte) {
	count := len(data) / 16
	for i := 0; i < count; i++ {
		o.c.Encrypt(data[i*16:(i+1)*16], data[i*16:(i+1)*16])
	}
}
func (o *crypt) Decrypt(data []byte) {
	count := len(data) / 16
	for i := 0; i < count; i++ {
		o.c.Decrypt(data[i*16:(i+1)*16], data[i*16:(i+1)*16])
	}
}
