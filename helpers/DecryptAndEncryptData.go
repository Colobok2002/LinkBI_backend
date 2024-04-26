package helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

func stripPEMHeaders(pemStr string) string {
	var lines []string = strings.Split(pemStr, "\n")
	var keyBody string
	for _, line := range lines {
		if !strings.Contains(line, "-----BEGIN PUBLIC KEY-----") && !strings.Contains(line, "-----END PUBLIC KEY-----") {
			keyBody += line
		}
	}
	return keyBody
}

// DecryptDataWithPrivateKey расшифровывает данные с использованием приватного ключа.
func DecryptWithPrivateKey(encryptedData, privateKeyBase64 string) (string, error) {
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

func loadRSAPublicKeyFromPEM(pubKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKIX public key: %s", err)
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		return nil, fmt.Errorf("key type is not RSA")
	}
}

func EncryptWithPublicKey(msg string, publicKeyPEM string) (string, error) {
	pubKey, err := loadRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return "", fmt.Errorf("loading public key failed: %s", err)
	}
	fmt.Printf(msg)
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte(msg), nil)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %s", err)
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
