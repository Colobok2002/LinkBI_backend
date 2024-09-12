package helpers

import "fmt"

// Моковая функция для отправки email предупреждения при изменении IP
func SendEmailWarning(email string, newIP string) {
	// Логируем предупреждение в консоль
	fmt.Printf("Warning: suspicious activity detected for user %s. New IP: %s\n", email, newIP)
}
