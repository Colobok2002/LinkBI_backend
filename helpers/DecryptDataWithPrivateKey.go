package Helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"strings"
)

// DecryptDataWithPrivateKey расшифровывает данные с использованием приватного ключа.
func DecryptDataWithPrivateKey(encryptedData, privateKeyBase64 string) (string, error) {
	if !strings.Contains(privateKeyBase64, "\n") {
		privateKeyBase64 = strings.ReplaceAll(privateKeyBase64, "-----END RSA PRIVATE KEY-----", "\n-----END RSA PRIVATE KEY-----")
		privateKeyBase64 = strings.ReplaceAll(privateKeyBase64, "-----BEGIN RSA PRIVATE KEY-----", "-----BEGIN RSA PRIVATE KEY-----\n")
	}

	privateKeyPEM := privateKeyBase64
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", errors.New("failed to decode private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	decryptedBytes, err := decryptRSA(encryptedData, privateKey)
	if err != nil {
		return "", err
	}

	return string(decryptedBytes), nil
}

// decryptRSA расшифровывает данные с использованием RSA приватного ключа.
func decryptRSA(encryptedData string, privateKey *rsa.PrivateKey) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	return rsa.DecryptPKCS1v15(rand.Reader, privateKey, data)
}
