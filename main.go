package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		go func() {
			time.Sleep(2 * time.Second)
			fmt.Println("Асинхронная задача завершена")
		}()
		fmt.Fprintf(w, "Привет, мир!")
	})

	fmt.Println("Сервер запущен на http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
