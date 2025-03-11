// Йоу, чат! Зараз розберемо як створювати чанки в майнкрафті!
// Це утиліта для створення тестових чанків для нашого серверу

package main

import (
	// Стандартні пакети Go
	"bytes"         // Для роботи з байтами
	"compress/gzip" // Для стиснення даних
	"os"            // Для роботи з файлами
	"path/filepath" // Для роботи з шляхами

	// Пакети для роботи з форматом Minecraft
	"github.com/Tnze/go-mc/nbt"         // NBT формат даних
	"github.com/Tnze/go-mc/save/region" // Формат регіонів
)

func main() {
	// Створюємо директорію region в папці world
	// 0755 - права доступу (читання для всіх, запис для власника)
	if err := os.MkdirAll(filepath.Join("world", "region"), 0755); err != nil {
		panic(err)
	}

	// Шлях до файлу регіону
	// r.0.0.mca - регіон на координатах (0,0)
	// .mca - розширення файлів регіонів (Minecraft Anvil)
	regionPath := filepath.Join("world", "region", "r.0.0.mca")

	// Видаляємо старий файл якщо він існує
	if _, err := os.Stat(regionPath); err == nil {
		if err := os.Remove(regionPath); err != nil {
			panic(err)
		}
	}

	// Створюємо новий файл регіону
	r, err := region.Create(regionPath)
	if err != nil {
		panic(err)
	}
	// Закриваємо файл коли закінчимо
	defer r.Close()

	// Створюємо простий чанк з бедроком внизу
	chunkData := createSimpleChunk()

	// Записуємо чанк в регіон на позиції (0,0)
	if err := r.WriteSector(0, 0, chunkData); err != nil {
		panic(err)
	}

	// Перевіряємо що чанк дійсно записався
	if !r.ExistSector(0, 0) {
		panic("Чанк не було збережено!")
	}
}

// createSimpleChunk створює найпростіший чанк
// Він містить тільки шар бедроку внизу і біом "рівнини"
func createSimpleChunk() []byte {
	// Структура даних чанку в форматі NBT
	chunk := map[string]interface{}{
		// Версія формату даних (для 1.20)
		"DataVersion": int32(3465),

		// Позиція чанку в світі
		"xPos": int32(0),  // X координата
		"yPos": int32(-4), // Y координата (найнижча секція в 1.20)
		"zPos": int32(0),  // Z координата

		// Статус генерації чанку
		"Status": "full", // Повністю згенерований

		// Час останнього оновлення
		"LastUpdate": int64(0),

		// Карти висот для різних цілей
		"Heightmaps": map[string][]int64{
			"WORLD_SURFACE":             make([]int64, 22), // Поверхня світу
			"WORLD_SURFACE_WG":          make([]int64, 22), // Поверхня для генератора світу
			"OCEAN_FLOOR":               make([]int64, 22), // Дно океану
			"OCEAN_FLOOR_WG":            make([]int64, 22), // Дно для генератора
			"MOTION_BLOCKING":           make([]int64, 22), // Блоки що блокують рух
			"MOTION_BLOCKING_NO_LEAVES": make([]int64, 22), // Блоки без листя
		},

		// Секції чанку (16x16x16 блоків)
		"sections": []map[string]interface{}{
			{
				"Y": int8(-4), // Висота секції
				// Стани блоків в секції
				"block_states": map[string]interface{}{
					// Палітра блоків (які блоки є в секції)
					"palette": []map[string]interface{}{
						{"Name": "minecraft:bedrock"}, // Тільки бедрок
					},
					"data": []int64{0}, // Індекси блоків з палітри
				},
				// Біоми в секції
				"biomes": map[string]interface{}{
					"palette": []string{"minecraft:plains"}, // Тільки рівнини
					"data":    []int64{0},                   // Індекси біомів
				},
			},
		},
	}

	// Створюємо буфер для запису даних
	var buffer bytes.Buffer

	// 1 = використовуємо gzip стиснення
	buffer.WriteByte(1)

	// Створюємо gzip writer для стиснення
	gzipWriter := gzip.NewWriter(&buffer)

	// Записуємо NBT дані в стиснутий потік
	if err := nbt.NewEncoder(gzipWriter).Encode(chunk, ""); err != nil {
		panic(err)
	}

	// Закриваємо gzip writer щоб записати все
	if err := gzipWriter.Close(); err != nil {
		panic(err)
	}

	// Повертаємо стиснуті байти
	return buffer.Bytes()
}
