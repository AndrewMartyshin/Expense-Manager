package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func fail(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

// registerCRD вешает на группу четыре маршрута для сущности T:
//
//	POST   /path        — create (у каждой сущности своя валидация, поэтому передаётся снаружи)
//	GET    /path        — список
//	GET    /path/:id    — одна запись
//	DELETE /path/:id    — удаление
//
// Read и Delete одинаковы для всех таблиц, поэтому написаны один раз через дженерик.
// pk и order — константы из кода (не пользовательский ввод), так что склейка в SQL безопасна;
// значение :id всегда передаётся параметром.
func registerCRD[T any](g *gin.RouterGroup, db *gorm.DB, path, pk, order string, create gin.HandlerFunc) {
	g.POST(path, create)

	g.GET(path, func(c *gin.Context) {
		items := make([]T, 0) // пустой список отдаём как [], а не null
		if err := db.Order(order).Find(&items).Error; err != nil {
			fail(c, http.StatusInternalServerError, "db error")
			return
		}
		c.JSON(http.StatusOK, items)
	})

	g.GET(path+"/:id", func(c *gin.Context) {
		id, ok := parseID(c, pk)
		if !ok {
			return
		}
		var item T
		err := db.Where(pk+" = ?", id).First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "not found")
			return
		}
		if err != nil {
			fail(c, http.StatusInternalServerError, "db error")
			return
		}
		c.JSON(http.StatusOK, item)
	})

	g.DELETE(path+"/:id", func(c *gin.Context) {
		id, ok := parseID(c, pk)
		if !ok {
			return
		}
		res := db.Where(pk+" = ?", id).Delete(new(T))
		if errors.Is(res.Error, gorm.ErrForeignKeyViolated) {
			fail(c, http.StatusConflict, "record is referenced by other records")
			return
		}
		if res.Error != nil {
			fail(c, http.StatusInternalServerError, "db error")
			return
		}
		if res.RowsAffected == 0 {
			fail(c, http.StatusNotFound, "not found")
			return
		}
		c.Status(http.StatusNoContent)
	})
}

// parseID: для числовых таблиц id обязан быть числом (иначе Postgres вернёт ошибку
// приведения типа и получится 500 вместо 400); у currencies ключ — строковый код.
func parseID(c *gin.Context, pk string) (any, bool) {
	raw := c.Param("id")
	if pk != "id" {
		return strings.ToUpper(raw), true
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "id must be a positive integer")
		return nil, false
	}
	return n, true
}

// ---------- Create-обработчики ----------

func createProfile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in struct {
			Name string `json:"name" binding:"required"`
			Role string `json:"role"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		if in.Role == "" {
			in.Role = "user"
		}
		p := Profile{Name: in.Name, Role: in.Role}
		if err := db.Create(&p).Error; err != nil {
			fail(c, http.StatusInternalServerError, "db error")
			return
		}
		c.JSON(http.StatusCreated, p)
	}
}

func createCategory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in struct {
			Code string `json:"code" binding:"required"`
			Name string `json:"name" binding:"required"`
			Type string `json:"type" binding:"required,oneof=income expense"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		cat := Category{Code: in.Code, Name: in.Name, Type: in.Type}
		err := db.Create(&cat).Error
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			fail(c, http.StatusConflict, "category code already exists")
			return
		}
		if err != nil {
			fail(c, http.StatusInternalServerError, "db error")
			return
		}
		c.JSON(http.StatusCreated, cat)
	}
}

func createCurrency(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in struct {
			Code       string `json:"code" binding:"required,len=3"`
			Name       string `json:"name" binding:"required"`
			MinorUnits *int   `json:"minor_units" binding:"required,gte=0"` // указатель: 0 (иена) — валидное значение
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		cur := Currency{Code: strings.ToUpper(in.Code), Name: in.Name, MinorUnits: *in.MinorUnits}
		err := db.Create(&cur).Error
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			fail(c, http.StatusConflict, "currency already exists")
			return
		}
		if err != nil {
			fail(c, http.StatusInternalServerError, "db error")
			return
		}
		c.JSON(http.StatusCreated, cur)
	}
}

func createTransaction(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in struct {
			ProfileID      uint   `json:"profile_id" binding:"required"`
			CategoryID     uint   `json:"category_id" binding:"required"`
			Currency       string `json:"currency" binding:"required,len=3"`
			Type           string `json:"type" binding:"required,oneof=income expense"`
			AmountMinor    int64  `json:"amount_minor" binding:"required,gt=0"`
			OccurredOn     string `json:"occurred_on" binding:"required"` // YYYY-MM-DD
			Description    string `json:"description"`
			IdempotencyKey string `json:"idempotency_key"`
		}
		if err := c.ShouldBindJSON(&in); err != nil {
			fail(c, http.StatusBadRequest, err.Error())
			return
		}
		day, err := time.Parse("2006-01-02", in.OccurredOn)
		if err != nil {
			fail(c, http.StatusBadRequest, "occurred_on must be a date in YYYY-MM-DD format")
			return
		}

		tx := Transaction{
			ProfileID:   in.ProfileID,
			CategoryID:  in.CategoryID,
			Currency:    strings.ToUpper(in.Currency),
			Type:        in.Type,
			AmountMinor: in.AmountMinor,
			OccurredOn:  day,
			Description: in.Description,
		}
		if in.IdempotencyKey != "" {
			tx.IdempotencyKey = &in.IdempotencyKey

			// Повтор запроса с тем же ключом — отдаём уже созданную операцию, дубль не пишем.
			if existing, found := findByIdempotencyKey(db, tx.ProfileID, in.IdempotencyKey); found {
				c.JSON(http.StatusOK, existing)
				return
			}
		}

		err = db.Create(&tx).Error
		switch {
		case errors.Is(err, gorm.ErrForeignKeyViolated):
			fail(c, http.StatusBadRequest, "profile, category or currency does not exist")
		case errors.Is(err, gorm.ErrDuplicatedKey):
			// Два одинаковых запроса пришли одновременно: проигравший возвращает запись победителя.
			if existing, found := findByIdempotencyKey(db, tx.ProfileID, in.IdempotencyKey); found {
				c.JSON(http.StatusOK, existing)
				return
			}
			fail(c, http.StatusConflict, "duplicate transaction")
		case err != nil:
			fail(c, http.StatusInternalServerError, "db error")
		default:
			c.JSON(http.StatusCreated, tx)
		}
	}
}

func findByIdempotencyKey(db *gorm.DB, profileID uint, key string) (Transaction, bool) {
	var t Transaction
	err := db.Where("profile_id = ? AND idempotency_key = ?", profileID, key).First(&t).Error
	return t, err == nil
}
