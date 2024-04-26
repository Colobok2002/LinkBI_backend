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

func EncryptWithPublicKey(data string, publicKeyBase64 string) (string, error) {
	// Приведение ключа к правильному формату PEM
	publicKeyPEM := formatPEM(publicKeyBase64)

	// Декодирование PEM-формата ключа
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "", errors.New("failed to decode public key")
	}

	// Преобразование ключа в структуру rsa.PublicKey
	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	// Приведение ключа к нужному типу
	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("failed to convert to RSA public key")
	}

	// Шифрование данных
	encryptedBytes, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, []byte(data))
	if err != nil {
		return "", err
	}

	// Кодирование зашифрованных данных в base64
	encryptedData := base64.StdEncoding.EncodeToString(encryptedBytes)

	return encryptedData, nil
}

func formatPEM(publicKeyBase64 string) string {
	// Проверка, что ключ уже в правильном формате
	if !strings.Contains(publicKeyBase64, "\n") {
		// Если нет, добавляем необходимые переносы строк
		publicKeyBase64 = strings.ReplaceAll(publicKeyBase64, "-----END PUBLIC KEY-----", "\n-----END PUBLIC KEY-----")
		publicKeyBase64 = strings.ReplaceAll(publicKeyBase64, "-----BEGIN PUBLIC KEY-----", "-----BEGIN PUBLIC KEY-----\n")
	}

	return publicKeyBase64
}
