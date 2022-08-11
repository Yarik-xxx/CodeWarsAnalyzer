package main

import (
	"CodeWarsAnalyzer/cwapi"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
)

var cache cwapi.Cache

//const port = "8080"

func main() {
	// Загрузка Cache
	err := cache.Init()
	checkError(err)

	// Получение порта
	port := getPort()

	// Роутинг
	mux := http.NewServeMux()
	mux.HandleFunc("/", HomePage)
	mux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("ui/static"))))

	// Запуск сервера
	log.Printf("Start routing http://localhost:%s/\n", port)
	err = http.ListenAndServe("localhost:"+port, mux)
	checkError(err)
}

func HomePage(w http.ResponseWriter, r *http.Request) {
	// Игнорирование сторонних адресов
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var response cwapi.AllInfoUser
	username := r.FormValue("search")

	// Генерация ответа при наличии запроса
	if username != "" {
		response = cwapi.GetAllInfoUser(username, &cache)
	}

	html, err := template.ParseFiles("ui/html/homepage.html")
	checkError(err)

	err = html.Execute(w, response)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getPort() string {
	args := os.Args
	if len(args) < 2 || len(args) > 2 {
		return "8080"
	}

	port, err := strconv.Atoi(args[1])
	checkError(err)

	return fmt.Sprintf("%d", port)
}
