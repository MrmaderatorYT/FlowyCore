// This file is part of go-mc/server project.
// Copyright (C) 2023.  Tnze
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Йоу, чат! Зараз розберемо конфігурацію нашого сервера!
// Тут зберігаються всі налаштування які можна змінити

package game

import (
	// time потрібен для роботи з часом
	"time"

	// rate використовуємо для обмеження навантаження
	"golang.org/x/time/rate"
)

// Config - головна структура з налаштуваннями сервера
// Поля з тегом `toml` читаються з конфіг файлу
type Config struct {
	// Максимальна кількість гравців на сервері
	MaxPlayers int `toml:"max-players"`

	// На яку відстань (в чанках) гравці бачать світ
	// 1 чанк = 16 блоків, тобто при значенні 10 видно на 160 блоків
	ViewDistance int32 `toml:"view-distance"`

	// IP адреса і порт на якому запускається сервер
	// Наприклад "0.0.0.0:25565"
	ListenAddress string `toml:"listen-address"`

	// MOTD (Message Of The Day) - повідомлення яке показується в списку серверів
	MessageOfTheDay string `toml:"motd"`

	// При якому розмірі пакети будуть стискатися
	// Менші пакети відправляються без стиснення для економії CPU
	NetworkCompressionThreshold int `toml:"network-compression-threshold"`

	// Чи перевіряти ліцензію гравців
	// true = тільки ліцензійні акаунти
	// false = можна зайти з піратки
	OnlineMode bool `toml:"online-mode"`

	// Назва папки де зберігається світ
	LevelName string `toml:"level-name"`

	// Чи вимагати від гравців безпечний профіль
	// Безпечний профіль = підписані повідомлення в чаті
	EnforceSecureProfile bool `toml:"enforce-secure-profile"`

	// Обмежувачі навантаження:
	// ChunkLoadingLimiter - скільки чанків можна завантажити за раз
	ChunkLoadingLimiter Limiter `toml:"chunk-loading-limiter"`
	// PlayerChunkLoadingLimiter - скільки чанків може завантажити один гравець
	PlayerChunkLoadingLimiter Limiter `toml:"player-chunk-loading-limiter"`
}

// Limiter - структура для обмеження частоти дій
// Наприклад: не більше 100 чанків кожні 5 секунд
type Limiter struct {
	// Як часто можна виконувати дію
	// Наприклад "5s" = кожні 5 секунд
	Every duration `toml:"every"`

	// Скільки разів можна виконати дію за цей період
	N int
}

// Limiter перетворює наші налаштування в готовий rate.Limiter
// rate.Limiter - це структура з бібліотеки golang.org/x/time/rate
// Вона стежить щоб не перевищувати ліміти
func (l *Limiter) Limiter() *rate.Limiter {
	return rate.NewLimiter(rate.Every(l.Every.Duration), l.N)
}

// duration - обгортка навколо time.Duration
// Потрібна щоб читати тривалість з конфіг файлу
type duration struct {
	time.Duration
}

// UnmarshalText перетворює текст з конфігу в time.Duration
// Наприклад "5s" -> 5 секунд
func (d *duration) UnmarshalText(text []byte) (err error) {
	d.Duration, err = time.ParseDuration(string(text))
	return
}
