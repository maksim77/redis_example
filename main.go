package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
)

// User представляет структуру пользователя.
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

var (
	db     *sql.DB
	rdb    *redis.Client
	tracer = otel.Tracer("redis_example")
)

// initDB инициализирует подключение к базе данных.
func initDB() error {
	var err error
	connStr := "postgres://user:password@localhost/mydb?sslmode=disable"
	db, err = otelsql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %v", err)
	}
	return db.Ping()
}

// initRedis инициализирует подключение к Redis/KeyDB.
func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	if err := redisotel.InstrumentTracing(rdb); err != nil {
		panic(err)
	}
}

// getUserByID возвращает пользователя из БД по ID, с кешированием в Redis.
func getUserByID(ctx context.Context, id int) (*User, error) {
	ctxLocal, span := tracer.Start(ctx, "getUserByID")
	defer span.End()

	// Пытаемся получить данные из кеша.
	cachedUser, err := rdb.Get(ctxLocal, strconv.Itoa(id)).Result()
	if err == nil {
		var user User
		if err := json.Unmarshal([]byte(cachedUser), &user); err == nil {
			return &user, nil
		}
	}

	var user User
	query := "SELECT id, name, age FROM users WHERE id = $1"
	err = db.QueryRowContext(ctxLocal, query, id).Scan(&user.ID, &user.Name, &user.Age)
	if err == sql.ErrNoRows {
		return nil, nil // пользователь не найден
	} else if err != nil {
		return nil, fmt.Errorf("ошибка запроса: %v", err)
	}

	// Сохраняем результаты в кеш.
	encodedUser, err := json.Marshal(user)
	if err == nil {
		rdb.Set(ctxLocal, strconv.Itoa(user.ID), encodedUser, time.Second*10)
	}

	return &user, nil
}

// userHandler обрабатывает запросы для получения пользователя.
func userHandler(w http.ResponseWriter, r *http.Request) {
	ctxLocal, span := tracer.Start(r.Context(), "handler")
	defer span.End()

	ids := r.URL.Query().Get("id")
	if ids == "" {
		http.Error(w, "id не указан", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(ids)
	if err != nil {
		http.Error(w, "id должен быть числом", http.StatusBadRequest)
		return
	}

	user, err := getUserByID(ctxLocal, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка получения пользователя: %v", err), http.StatusInternalServerError)
		return
	}

	if user == nil {
		http.Error(w, "пользователь не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, fmt.Sprintf("ошибка кодирования ответа: %v", err), http.StatusInternalServerError)
	}
}

func main() {
	shutdown, err := InstallExportPipeline()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			log.Fatal(err.Error())
		}
	}()

	// Инициализируем подключение к БД.
	if err := initDB(); err != nil {
		log.Fatalf("не удалось инициализировать БД: %v", err)
	}
	defer db.Close()

	// Инициализируем подключение к Redis.
	initRedis()
	defer rdb.Close()

	// Регистрируем обработчик.
	http.HandleFunc("/user", userHandler)

	// Запускаем сервер.
	fmt.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
