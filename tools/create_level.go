package main

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"time"

	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save"
)

func main() {
	// Створюємо базові дані рівня
	level := &save.Level{
		Data: save.LevelData{
			Version: struct {
				ID       int32 `nbt:"Id"`
				Name     string
				Series   string
				Snapshot byte
			}{
				ID:       2975,
				Name:     "1.19.4",
				Series:   "main",
				Snapshot: 0,
			},
			LevelName:      "world",
			GameType:       1, // Creative
			Time:           0,
			DayTime:        0,
			LastPlayed:     time.Now().UnixMilli(),
			SpawnX:         48,
			SpawnY:         100,
			SpawnZ:         35,
			SpawnAngle:     0,
			Difficulty:     2, // Normal
			HardCore:       false,
			GameRules:      make(map[string]string),
			DataVersion:    3337,
			Initialized:    true,
			StorageVersion: 19133,
		},
	}

	// Створюємо директорію world якщо її немає
	if err := os.MkdirAll("world", 0755); err != nil {
		panic(err)
	}

	// Відкриваємо файл для запису
	f, err := os.Create(filepath.Join("world", "level.dat"))
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Створюємо gzip writer
	gw := gzip.NewWriter(f)
	defer gw.Close()

	// Створюємо NBT encoder
	enc := nbt.NewEncoder(gw)
	if err := enc.Encode(level, ""); err != nil {
		panic(err)
	}
}
