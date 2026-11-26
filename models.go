package main

import "time"

// Модели 1-в-1 повторяют схему БД: profiles, categories, currencies, transactions.
// Имена таблиц GORM выводит сам (Profile -> profiles, Category -> categories, ...).

type Profile struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Role      string    `gorm:"not null" json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Category struct {
	ID   uint   `gorm:"primaryKey" json:"id"`
	Code string `gorm:"not null;uniqueIndex" json:"code"`
	Name string `gorm:"not null" json:"name"`
	Type string `gorm:"not null;check:chk_category_type,type IN ('income','expense')" json:"type"`
}

// Currency: первичный ключ — сам код валюты (RUB, USD), а не числовой id.
type Currency struct {
	Code       string `gorm:"primaryKey;size:3" json:"code"`
	Name       string `gorm:"not null" json:"name"`
	MinorUnits int    `gorm:"not null;check:chk_currency_minor,minor_units >= 0" json:"minor_units"`
}

type Transaction struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	ProfileID  uint `gorm:"not null;index;uniqueIndex:ux_tx_profile_idem" json:"profile_id"`
	CategoryID uint `gorm:"not null;index" json:"category_id"`
	// Колонка называется currency (как на схеме) и ссылается на currencies.code.
	Currency    string    `gorm:"size:3;not null" json:"currency"`
	Type        string    `gorm:"not null;check:chk_tx_type,type IN ('income','expense')" json:"type"`
	AmountMinor int64     `gorm:"not null;check:chk_tx_amount,amount_minor > 0" json:"amount_minor"`
	OccurredOn  time.Time `gorm:"type:date;not null;index" json:"occurred_on"`
	Description string    `json:"description"`
	// Указатель, чтобы «нет ключа» хранилось как NULL: в Postgres NULL-ы в
	// уникальном индексе не конфликтуют, а непустые ключи уникальны в рамках профиля.
	IdempotencyKey *string   `gorm:"uniqueIndex:ux_tx_profile_idem" json:"idempotency_key,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Связи нужны только для создания внешних ключей в БД, в JSON не отдаём.
	Profile     Profile  `gorm:"constraint:OnDelete:RESTRICT" json:"-"`
	Category    Category `gorm:"constraint:OnDelete:RESTRICT" json:"-"`
	CurrencyRef Currency `gorm:"foreignKey:Currency;references:Code;constraint:OnDelete:RESTRICT" json:"-"`
}
