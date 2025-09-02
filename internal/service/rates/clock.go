package rates

import "time"

// Clock — абстракция времени, чтобы тесты были детерминированны
type Clock interface {
	Now() time.Time
}

// realClock — prod реализация: текущее время в UTC
type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// NewRealClock - экспортируем фабрику, чтобы внешний пакет мог получить Clock
func NewRealClock() Clock {
	return realClock{}
}
