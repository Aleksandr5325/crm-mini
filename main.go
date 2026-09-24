package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"

	"back/internal/handlers"
	"back/internal/models"

	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// proxy1CHandler принимает запросы с фронта и пересылает их в 1С
func proxy1CHandler(w http.ResponseWriter, r *http.Request) {
	targetURL := r.Header.Get("X-Target-Url")
	if targetURL == "" {
		http.Error(w, "Missing X-Target-Url header", http.StatusBadRequest)
		return
	}

	body, _ := io.ReadAll(r.Body)
	req, err := http.NewRequest(r.Method, targetURL, bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	if token := r.Header.Get("token"); token != "" {
		// Вместо req.Header.Set("token", token)
		// Используем прямое присвоение в мапу, чтобы избежать автокапитализации Go (token -> Token).
		// 1С чувствительна к регистру заголовков и ожидает именно "token" в нижнем регистре.
		req.Header["token"] = []string{token}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "1C proxy error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// corsMiddleware разрешает фронту (http://localhost:5500 или любой origin)
// обращаться к этому бэку, и корректно отвечает на preflight OPTIONS.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Я добавил метод PUT на всякий случай, так как мы используем его для обновления стадий
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		// Добавляем наши новые кастомные заголовки
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Target-Url, token")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // 204
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Строка подключения к PostgreSQL
	dsn := "host=localhost user=postgres password=admin123 dbname=crm port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("failed to connect database")
	}
	log.Println("Database connected successfully")

	err = db.AutoMigrate(
		&models.Source{},
		&models.Lead{},
		&models.LeadStage{},
		&models.ImportLog{},
	)
	if err != nil {
		panic("failed to migrate database: " + err.Error())
	}
	log.Println("Tables created successfully")

	// Создаём роутер
	r := mux.NewRouter()

	// Подключаем CORS на все запросы
	r.Use(corsMiddleware)

	// Создаём обработчики
	importHandler := &handlers.ImportHandler{DB: db}
	sourceHandler := &handlers.SourceHandler{DB: db}
	leadHandler := &handlers.LeadHandler{DB: db}

	// Регистрируем маршруты
	// Регистрируем маршруты
	// Импорт
	r.HandleFunc("/import/leads", importHandler.ImportLeads).Methods("POST", "OPTIONS")
	r.HandleFunc("/import/sources", importHandler.ImportSources).Methods("POST", "OPTIONS")
	r.HandleFunc("/import/employees", importHandler.ImportEmployees).Methods("POST", "OPTIONS")
	r.HandleFunc("/import/contacts", importHandler.ImportContacts).Methods("POST", "OPTIONS")
	// Источники
	r.HandleFunc("/sources", sourceHandler.GetSources).Methods("GET", "OPTIONS")

	// Прокси для 1С (решает проблему CORS)
	r.HandleFunc("/proxy/1c", proxy1CHandler).Methods("POST", "PUT", "PATCH", "OPTIONS")

	// Лиды
	r.HandleFunc("/leads", leadHandler.GetLeads).Methods("GET", "OPTIONS")
	r.HandleFunc("/leads", leadHandler.CreateLead).Methods("POST", "OPTIONS")
	r.HandleFunc("/leads/{id}", leadHandler.GetLead).Methods("GET", "OPTIONS")
	r.HandleFunc("/leads/{id}", leadHandler.PatchLead).Methods("PATCH", "PUT", "OPTIONS")
	r.HandleFunc("/leads/{id}/stages", leadHandler.GetLeadStages).Methods("GET", "OPTIONS")
	r.HandleFunc("/leads/{id}/move", leadHandler.MoveStage).Methods("POST", "OPTIONS")

	// Запускаем сервер
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
