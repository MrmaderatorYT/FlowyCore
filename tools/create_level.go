// Йоу, чат! Зараз розберемо як створити новий світ в майнкрафті!
// Цей файл створює level.dat - головний файл світу, який містить всі його налаштування

package main

import (
	// Стандартні пакети Go
	"compress/gzip" // Для стиснення файлу
	"os"            // Для роботи з файлами
	"path/filepath" // Для роботи з шляхами
	"time"          // Для часових міток

	// Пакети для роботи з форматом Minecraft
	"github.com/Tnze/go-mc/nbt"  // NBT формат даних
	"github.com/Tnze/go-mc/save" // Робота з файлами збереження
)

func main() {
	// Створюємо структуру даних світу
	level := &save.Level{
		Data: save.LevelData{
			// Версія гри
			Version: struct {
				ID       int32  `nbt:"Id"` // ID версії
				Name     string // Назва версії
				Series   string // Серія (main/snapshot)
				Snapshot byte   // Чи це снапшот
			}{
				ID:       2975,     // ID версії 1.19.4
				Name:     "1.19.4", // Назва версії
				Series:   "main",   // Основна серія
				Snapshot: 0,        // Не снапшот
			},

			// Базові налаштування світу
			LevelName: "world", // Назва світу
			GameType:  1,       // 1 = Creative режим

			// Час у світі
			Time:    0, // Загальний час (тіки)
			DayTime: 0, // Час дня (тіки)

			// Час останньої гри
			LastPlayed: time.Now().UnixMilli(),

			// Точка спавну гравців
			SpawnX:     48,  // X координата
			SpawnY:     100, // Y координата (висота)
			SpawnZ:     35,  // Z координата
			SpawnAngle: 0,   // Кут повороту при спавні

			// Налаштування складності
			Difficulty: 2,     // 2 = Normal складність
			HardCore:   false, // Хардкор вимкнений

			// Правила гри (keepInventory, doDaylightCycle і т.д.)
			GameRules: make(map[string]string),

			// Технічні параметри
			DataVersion:    3337,  // Версія формату даних
			Initialized:    true,  // Світ ініціалізований
			StorageVersion: 19133, // Версія формату збереження
		},
	}

	// Створюємо директорію world якщо її немає
	// 0755 = права доступу (читання для всіх, запис для власника)
	if err := os.MkdirAll("world", 0755); err != nil {
		panic(err)
	}

	// Відкриваємо файл level.dat для запису
	f, err := os.Create(filepath.Join("world", "level.dat"))
	if err != nil {
		panic(err)
	}
	// Закриваємо файл коли закінчимо
	defer f.Close()

	// Створюємо gzip writer для стиснення
	gw := gzip.NewWriter(f)
	// Закриваємо gzip коли закінчимо
	defer gw.Close()

	// Створюємо NBT encoder і записуємо дані
	enc := nbt.NewEncoder(gw)
	if err := enc.Encode(level, ""); err != nil {
		panic(err)
	}
}
