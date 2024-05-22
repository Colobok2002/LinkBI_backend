package helpers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// addPadding добавляет паддинг к данным в соответствии с PKCS#7
func addPadding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	paddedData := make([]byte, len(data)+padding)
	copy(paddedData, data)
	for i := len(data); i < len(paddedData); i++ {
		paddedData[i] = byte(padding)
	}
	return paddedData
}

// removePadding удаляет паддинг из данных в соответствии с PKCS#7
func removePadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("data length is zero")
	}
	padding := int(data[length-1])
	if padding > length {
		return nil, fmt.Errorf("invalid padding size")
	}
	return data[:length-padding], nil
}

type UserData struct {
	User_id uint
}

// EncryptAES шифрует данные с использованием AES
func EncryptAES(data UserData) (string, error) {
	secretKey := sha256.Sum256([]byte(os.Getenv("TOKEN_KEY")))
	block, err := aes.NewCipher(secretKey[:])
	if err != nil {
		return "", err
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	paddedData := addPadding(dataBytes, aes.BlockSize)
	ciphertext := make([]byte, aes.BlockSize+len(paddedData))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCBCEncrypter(block, iv)
	stream.CryptBlocks(ciphertext[aes.BlockSize:], paddedData)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAES расшифровывает данные с использованием AES
func DecryptAES(encrypted string) (UserData, error) {
	secretKey := sha256.Sum256([]byte(os.Getenv("TOKEN_KEY")))

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return UserData{}, err
	}

	block, err := aes.NewCipher(secretKey[:])
	if err != nil {
		return UserData{}, err
	}

	if len(ciphertext) < aes.BlockSize {
		return UserData{}, fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCBCDecrypter(block, iv)
	stream.CryptBlocks(ciphertext, ciphertext)
	decryptedData, err := removePadding(ciphertext)
	if err != nil {
		return UserData{}, err
	}

	var data UserData
	err = json.Unmarshal(decryptedData, &data)
	if err != nil {
		return UserData{}, err
	}

	return data, nil
}
