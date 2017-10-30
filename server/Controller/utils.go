package Controller

import (
	"crypto/aes"
	"crypto/cipher"
	"log"
)

func GCMEncrypter(data []byte, key [32]byte, nonce []byte) []byte {
	// The key argument should be the AES key, either 16 or 32 bytes
	// to select AES-128 or AES-256.
	block, err := aes.NewCipher(key[:])
	if err != nil {
		log.Println(err.Error())
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Println(err.Error())
	}

	ciphertext := aesgcm.Seal(nil, nonce, data, nil)
	return ciphertext
}

func GCMDecrypter(encData []byte, key [32]byte, nonce []byte) ([]byte, error) {
	// The key argument should be the AES key, either 16 or 32 bytes
	// to select AES-128 or AES-256.

	block, err := aes.NewCipher(key[:])
	if err != nil {
		log.Println(err.Error())
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		log.Println(err.Error())
	}

	plaintext, err := aesgcm.Open(nil, nonce, encData, nil)

	return plaintext, err
	// Output: exampleplaintext
}