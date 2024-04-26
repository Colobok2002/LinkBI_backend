package helpers

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
func DecryptWithPrivateKey(encryptedData, privateKeyBase64 string) (string, error) {
	privateKeyPEM := formatPEM(privateKeyBase64)
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

// EncryptWithPublicKey кодирует данные с использованием публичного ключа.
func EncryptWithPublicKey(data string, publicKeyBase64 string) (string, error) {
	publicKeyPEM := formatPEM(publicKeyBase64)

	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "", errors.New("failed to decode public key")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("failed to convert to RSA public key")
	}

	encryptedBytes, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, []byte(data))
	if err != nil {
		return "", err
	}

	encryptedData := base64.StdEncoding.EncodeToString(encryptedBytes)

	return encryptedData, nil
}

func formatPEM(keyBase64 string) string {
	if !strings.Contains(keyBase64, "\n") {
		keyBase64 = strings.ReplaceAll(keyBase64, "-----END PUBLIC KEY-----", "\n-----END PUBLIC KEY-----")
		keyBase64 = strings.ReplaceAll(keyBase64, "-----BEGIN PUBLIC KEY-----", "-----BEGIN PUBLIC KEY-----\n")
		keyBase64 = strings.ReplaceAll(keyBase64, "-----END RSA PRIVATE KEY-----", "\n-----END RSA PRIVATE KEY-----")
		keyBase64 = strings.ReplaceAll(keyBase64, "-----BEGIN RSA PRIVATE KEY-----", "-----BEGIN RSA PRIVATE KEY-----\n")
	}
	return keyBase64
}
