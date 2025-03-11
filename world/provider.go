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

// Йоу, чат! Сьогодні ми розберемо як працює система збереження даних!
// Цей файл відповідає за завантаження та збереження чанків і даних гравців.
// Тут використовується формат Minecraft Region (.mca файли) для чанків
// та NBT формат для даних гравців. Погнали розбиратися!

package world

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/time/rate"

	"github.com/Tnze/go-mc/level"
	"github.com/Tnze/go-mc/save"
	"github.com/Tnze/go-mc/save/region"
	"github.com/Tnze/go-mc/yggdrasil/user"
)

// ChunkProvider реалізує систему збереження чанків
// Використовує .mca файли для зберігання даних світу
type ChunkProvider struct {
	dir     string        // директорія з регіонами
	limiter *rate.Limiter // обмежувач швидкості завантаження
}

// NewProvider створює новий провайдер чанків
// dir - шлях до директорії з регіонами
// limiter - обмежувач швидкості завантаження
func NewProvider(dir string, limiter *rate.Limiter) ChunkProvider {
	return ChunkProvider{dir: dir, limiter: limiter}
}

// ErrReachRateLimit повертається коли перевищено ліміт завантаження
var ErrReachRateLimit = errors.New("reach rate limit")

// GetChunk завантажує чанк за його координатами
// Спочатку перевіряє ліміт, потім шукає потрібний регіон
// і завантажує з нього дані чанку
func (p *ChunkProvider) GetChunk(pos [2]int32) (c *level.Chunk, errRet error) {
	if !p.limiter.Allow() {
		return nil, ErrReachRateLimit
	}
	// Отримуємо регіон, в якому знаходиться чанк
	r, err := p.getRegion(region.At(int(pos[0]), int(pos[1])))
	if err != nil {
		return nil, fmt.Errorf("open region fail: %w", err)
	}
	defer func(r *region.Region) {
		err2 := r.Close()
		if errRet == nil && err2 != nil {
			errRet = fmt.Errorf("close region fail: %w", err2)
		}
	}(r)

	// Перевіряємо чи існує чанк в регіоні
	x, z := region.In(int(pos[0]), int(pos[1]))
	if !r.ExistSector(x, z) {
		return nil, errChunkNotExist
	}

	// Читаємо дані чанку
	data, err := r.ReadSector(x, z)
	if err != nil {
		return nil, fmt.Errorf("read sector fail: %w", err)
	}

	// Парсимо NBT дані чанку
	var chunk save.Chunk
	if err := chunk.Load(data); err != nil {
		return nil, fmt.Errorf("parse chunk data fail: %w", err)
	}

	// Конвертуємо в структуру level.Chunk
	c, err = level.ChunkFromSave(&chunk)
	if err != nil {
		return nil, fmt.Errorf("load chunk data fail: %w", err)
	}
	return c, nil
}

// getRegion повертає об'єкт регіону за координатами
// Якщо файл не існує - створює новий
func (p *ChunkProvider) getRegion(rx, rz int) (*region.Region, error) {
	filename := fmt.Sprintf("r.%d.%d.mca", rx, rz)
	path := filepath.Join(p.dir, filename)
	r, err := region.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		r, err = region.Create(path)
	}
	return r, err
}

// PutChunk зберігає чанк у файл регіону
// Наразі функція закоментована, але вона конвертує чанк в NBT
// та зберігає його у відповідний .mca файл
func (p *ChunkProvider) PutChunk(pos [2]int32, c *level.Chunk) (err error) {
	// Закоментований код збереження чанку
	return nil
}

// errChunkNotExist повертається коли чанк не знайдено
var errChunkNotExist = errors.New("ErrChunkNotExist")

// PlayerProvider відповідає за збереження даних гравців
type PlayerProvider struct {
	dir string // директорія з файлами гравців
}

// NewPlayerProvider створює новий провайдер даних гравців
func NewPlayerProvider(dir string) PlayerProvider {
	return PlayerProvider{dir: dir}
}

// GetPlayer завантажує дані гравця з файлу
// Дані зберігаються в .dat файлі в форматі NBT з GZIP стисненням
func (p *PlayerProvider) GetPlayer(name string, id uuid.UUID, pubKey *user.PublicKey, properties []user.Property) (player *Player, errRet error) {
	// Відкриваємо файл гравця за його UUID
	f, err := os.Open(filepath.Join(p.dir, id.String()+".dat"))
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		err2 := f.Close()
		if errRet == nil && err2 != nil {
			errRet = fmt.Errorf("close player data fail: %w", err2)
		}
	}(f)

	// Розпаковуємо GZIP
	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("open gzip reader fail: %w", err)
	}

	// Читаємо NBT дані
	data, err := save.ReadPlayerData(r)
	if err != nil {
		return nil, fmt.Errorf("read player data fail: %w", err)
	}
	if err := r.Close(); err != nil {
		return nil, fmt.Errorf("close gzip reader fail: %w", err)
	}

	// Створюємо об'єкт гравця з завантажених даних
	player = &Player{
		Entity: Entity{
			EntityID: NewEntityID(),
			Position: data.Pos,
			Rotation: data.Rotation,
		},
		Name:       name,
		UUID:       id,
		PubKey:     pubKey,
		Properties: properties,
		ChunkPos: [3]int32{ // Розраховуємо позицію в чанках
			int32(data.Pos[0]) >> 5,
			int32(data.Pos[1]) >> 5,
			int32(data.Pos[2]) >> 5,
		},
		Gamemode:       data.PlayerGameType,
		EntitiesInView: make(map[int32]*Entity),
		ViewDistance:   10,
	}
	return
}
