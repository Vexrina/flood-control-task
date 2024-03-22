package main_test

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// Изначальный интерфейс
type FloodControl interface {
	Check(ctx context.Context, userID int64) (bool, error)
}

// реализация интерфейса FloodControl.
type FloodControlImpl struct {
	mu           sync.Mutex           // мьютекс, чтобы данные в функцию не летели из разных экземпляров приложения одновременно, не вызывали проблемы в...
	requests     map[int64]*list.List // кэшэ, который хранит запросы за...
	timeInterval time.Duration        // интервал времени (N секунд)
	maxRequests  int                  // максимальное количество запросов (K запросов или вызовов функций)
}

// функция для создания нового флудконтрола
func NewFloodControl(timeInterval time.Duration, maxRequests int) *FloodControlImpl {
	return &FloodControlImpl{
		requests:     make(map[int64]*list.List), // создаем пустой кэш запросов
		timeInterval: timeInterval,               // задаем интервал времени (N)
		maxRequests:  maxRequests,                // задаем максимальное количество вызовов функций (K)
	}
}

func (fc *FloodControlImpl) Check(ctx context.Context, userID int64) (bool, error) {
    fc.mu.Lock()
    defer fc.mu.Unlock()

    now := time.Now()

    // Получение списка запросов пользователя
    requestTimes := fc.requests[userID]

    // Инициализация списка, если он пустой
    if requestTimes == nil {
        requestTimes = list.New()
    }

    // Удаление старых запросов
    for front := requestTimes.Front(); front != nil; {
        value, ok := front.Value.(time.Time)
        if !ok {
            return false, errors.New("в очереди хранится не время")
        }
        if now.Sub(value) <= fc.timeInterval {
            break
        }
        next := front.Next()
        requestTimes.Remove(front)
        front = next
    }

    // Проверка количества запросов
    if requestTimes.Len() >= fc.maxRequests {
        return false, errors.New("превышено максимальное количество запросов")
    }

    // Добавление текущего запроса в список
    requestTimes.PushBack(now)
    fc.requests[userID] = requestTimes

    return true, nil
}



func TestFloodControlImpl_Check(t *testing.T) {
	fc := NewFloodControl(time.Second*10, 5)

	// убедимся, что запросы от разных пользователей не влияют друг на друга
	userID1 := int64(1)
	userID2 := int64(2)

	ok, err := fc.Check(context.Background(), userID1)
	if err != nil {
		t.Errorf("Ошибка при проверке первого запроса: %v", err)
	}
	if !ok {
		t.Error("Первый запрос должен пройти успешно")
	}

	ok, err = fc.Check(context.Background(), userID2)
	if err != nil {
		t.Errorf("Ошибка при проверке второго запроса: %v", err)
	}
	if !ok {
		t.Error("Второй запрос должен пройти успешно")
	}

	// проверка, что превышение максимального количества запросов возвращает ошибку
	for i := 0; i < 5; i++ {
		fc.Check(context.Background(), userID1)
	}

	ok, _ = fc.Check(context.Background(), userID1)
	if ok {
		t.Error("Превышение максимального количества запросов должно вернуть ошибку")
	}
	ok, err = fc.Check(context.Background(), userID2)
	if err != nil {
		t.Errorf("Ошибка при проверке второго запроса: %v", err)
	}
	if !ok {
		t.Error("Второй запрос должен пройти успешно")
	}
}