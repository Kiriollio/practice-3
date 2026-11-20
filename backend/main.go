package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	dsn := "host=" + env("DB_HOST", "db") +
		" port=" + env("DB_PORT", "5432") +
		" user=" + env("DB_USER", "app") +
		" password=" + env("DB_PASSWORD", "secret") +
		" dbname=" + env("DB_NAME", "appdb") +
		" sslmode=disable"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	staticDir := env("STATIC_DIR", "/app/static")
	fs := http.FileServer(http.Dir(staticDir))

	// Проверка здоровья приложения и подключения к БД
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Список записей из БД
	http.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, name FROM items ORDER BY id")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer rows.Close()

		items := []Item{}
		for rows.Next() {
			var it Item
			if err := rows.Scan(&it.ID, &it.Name); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			items = append(items, it)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	})

	// Количество записей для сверки с БД в CI-тестах
	http.HandleFunc("/api/items/count", func(w http.ResponseWriter, r *http.Request) {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%d", count)
	})

	// Статический фронтенд
	http.Handle("/", fs)

	addr := ":" + env("HTTP_PORT", "8080")
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
