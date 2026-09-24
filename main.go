package main

import (
	"log"
	"net/http"

	"back/internal/handlers"
	"back/internal/models"

	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// corsMiddleware разрешает фронту (http://localhost:5500 или любой origin)
// обращаться к этому бэку, и корректно отвечает на preflight OPTIONS.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Я добавил метод PUT на всякий случай, так как мы используем его для обновления стадий
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

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
