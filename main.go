package main

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
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
	fc.mu.Lock() // лочим и анлочим мьютекс, чтобы кэш не переполнялся
	defer fc.mu.Unlock()

	now := time.Now() // смотрим какое сейчас время

	// чистим старые запросы
	requestTimes := fc.requests[userID]
	if requestTimes == nil { // пользователь не совершал запросы, после запуска флуд контроля
        requestTimes = list.New()
    }

    for front := requestTimes.Front(); front != nil; {
        value, ok := front.Value.(time.Time) // получаем самый старый запрос
        if !ok { // почему-то в очереди хранится не время
            return false, errors.New("в очереди хранится не время")
        }
        if now.Sub(value) <= fc.timeInterval { // запрос не старее 10 секунд
            break
        }
        next := front.Next() // идем дальше
        requestTimes.Remove(front) // удаляем старый запрос
        front = next // изменяем указатель на первый элемент
    }

	// смотрим, сколько запросов пришло
	if requestTimes.Len() >= fc.maxRequests {
		return false, errors.New("превышено максимальное количество запросов") // запросов >= K
	}

	// дописываем текущий обработанный запрос в кэш
	requestTimes.PushBack(now)
	fc.requests[userID] = requestTimes

	return true, nil
}

func main() {
	args := os.Args
	if len(args) != 3 {
		fmt.Println("Использование: go run main.go [количество_секунд] [максимальное_количество_запросов]")
	}

	N, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Println("Ошибка! Количество секунд должно быть целочисленным")
		return
	}
	if N <= 0 {
		fmt.Println("Ошибка! Количество секунд должно быть положительным")
		return
	}

	K, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Println("Ошибка! Количество запросов должно быть целочисленным")
		return
	}
	if K <= 0 {
		fmt.Println("Ошибка! Количество запросов должно быть положительным")
		return
	}

	fc := NewFloodControl(time.Second*time.Duration(N), K) // проверка на флуд за последние N секунд, максимум K запросов.

	for {
		// допустим, что общение между программами "глупое"
		// они просто пишут в терминал с запущенным флуд-контролем
		// и считывают ответ
		// скорее всего данный контроль развернут на каком-то микросервисе
		// и будут ходить в него, но по ТЗ это не просится реализовывать
		var userID int
		fmt.Scan(&userID)
		fmt.Println(fc.Check(context.Background(), int64(userID)))
	}
}
