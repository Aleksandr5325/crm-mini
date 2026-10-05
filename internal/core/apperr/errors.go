// Package apperr содержит общие ошибки приложения.
//
// Они не знают ни про HTTP, ни про базу данных: репозиторий переводит
// в них ошибки базы, а HTTP-слой переводит их в коды ответа.
package apperr

import "errors"

var (
	// ErrNotFound - запрошенная запись не существует
	ErrNotFound = errors.New("not found")

	// ErrInvalid - входные данные неверны: пустое имя, неизвестная стадия и т.п.
	ErrInvalid = errors.New("invalid input")

	// ErrConflict — операция противоречит текущему состоянию данных:
	// запись уже существует или её успели изменить одновременно с нами.
	ErrConflict = errors.New("conflict")
)
