package omsg

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"log"
)

type crypt struct {
	c cipher.Block
	k []byte
}

// 16, 24, or 32 bytes to select
// AES-128, AES-192, or AES-256.
func newCrypt(key []byte) *crypt {
	c, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalln(err)
	}
	return &crypt{c: c, k: key}
}

func (o *crypt) encrypt(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)
	out = pkcs5Padding(out, aes.BlockSize)
	cipher.NewCBCEncrypter(o.c, o.k[:aes.BlockSize]).CryptBlocks(out, out)
	return out
}
func (o *crypt) decrypt(data []byte) []byte {
	out := make([]byte, len(data))
	copy(out, data)
	cipher.NewCBCDecrypter(o.c, o.k[:aes.BlockSize]).CryptBlocks(out, out)
	return pkcs5UnPadding(out)
}

func pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pkcs5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	pos := length - unpadding
	if pos < 0 || pos >= len(origData) {
		return origData
	}
	return origData[:pos]
}
