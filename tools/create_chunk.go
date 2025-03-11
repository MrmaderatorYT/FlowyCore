package main

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"

	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save/region"
)

func main() {
	// Створюємо директорію для регіонів
	if err := os.MkdirAll(filepath.Join("world", "region"), 0755); err != nil {
		panic(err)
	}

	// Видаляємо існуючий файл регіону, якщо він є
	regionPath := filepath.Join("world", "region", "r.0.0.mca")
	if _, err := os.Stat(regionPath); err == nil {
		if err := os.Remove(regionPath); err != nil {
			panic(err)
		}
	}

	// Відкриваємо файл регіону
	r, err := region.Create(regionPath)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	// Створюємо найпростіший чанк
	chunkData := createSimpleChunk()

	// Записуємо дані до регіону
	if err := r.WriteSector(0, 0, chunkData); err != nil {
		panic(err)
	}

	// Перевіряємо, що чанк було збережено
	if !r.ExistSector(0, 0) {
		panic("Chunk was not saved")
	}
}

// Створює найпростіший чанк з бедроком на дні
func createSimpleChunk() []byte {
	// Структура чанку
	chunk := map[string]interface{}{
		"DataVersion": int32(3465), // Версія даних для Minecraft 1.20
		"xPos":        int32(0),
		"yPos":        int32(-4), // Найнижча секція для 1.20
		"zPos":        int32(0),
		"Status":      "full",
		"LastUpdate":  int64(0),
		"Heightmaps": map[string][]int64{
			"WORLD_SURFACE":             make([]int64, 22),
			"WORLD_SURFACE_WG":          make([]int64, 22),
			"OCEAN_FLOOR":               make([]int64, 22),
			"OCEAN_FLOOR_WG":            make([]int64, 22),
			"MOTION_BLOCKING":           make([]int64, 22),
			"MOTION_BLOCKING_NO_LEAVES": make([]int64, 22),
		},
		"sections": []map[string]interface{}{
			{
				"Y": int8(-4), // Змінюємо з 0 на -4, щоб відповідало yPos
				"block_states": map[string]interface{}{
					"palette": []map[string]interface{}{
						{"Name": "minecraft:bedrock"},
					},
					"data": []int64{0},
				},
				"biomes": map[string]interface{}{
					"palette": []string{"minecraft:plains"},
					"data":    []int64{0},
				},
			},
		},
	}

	// Створюємо буфер для даних
	var buffer bytes.Buffer
	buffer.WriteByte(1) // Тип компресії: 1 = gzip

	// Створюємо gzip writer
	gzipWriter := gzip.NewWriter(&buffer)

	// Записуємо NBT дані
	if err := nbt.NewEncoder(gzipWriter).Encode(chunk, ""); err != nil {
		panic(err)
	}

	// Закриваємо gzip writer
	if err := gzipWriter.Close(); err != nil {
		panic(err)
	}

	return buffer.Bytes()
}
