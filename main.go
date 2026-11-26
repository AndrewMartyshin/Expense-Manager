package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func setupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default() // Logger + Recovery: паника в обработчике не роняет сервер

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	api := r.Group("/api")
	registerCRD[Profile](api, db, "/profiles", "id", "id ASC", createProfile(db))
	registerCRD[Category](api, db, "/categories", "id", "id ASC", createCategory(db))
	registerCRD[Currency](api, db, "/currencies", "code", "code ASC", createCurrency(db))
	registerCRD[Transaction](api, db, "/transactions", "id", "occurred_on DESC, id DESC", createTransaction(db))

	return r
}

func main() {
	dsn := getenv("DATABASE_URL", "host=localhost user=expense password=expense dbname=expense port=5432 sslmode=disable")

	// TranslateError превращает ошибки Postgres в gorm.ErrDuplicatedKey / ErrForeignKeyViolated,
	// чтобы в хендлерах не разбирать коды ошибок драйвера вручную.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	// Для базового этапа таблицы создаёт AutoMigrate; позже его стоит заменить SQL-миграциями.
	if err := db.AutoMigrate(&Profile{}, &Category{}, &Currency{}, &Transaction{}); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	if err := setupRouter(db).Run(":" + getenv("PORT", "8080")); err != nil {
		log.Fatal(err)
	}
}
