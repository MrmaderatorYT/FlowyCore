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

// Йоу, чат! Сьогодні ми розберемо як влаштований світ у нашому сервері!
// Це центральний файл, який керує всім світом: чанками, гравцями,
// сутностями та їх взаємодією. Тут використовується багато крутих
// оптимізацій, наприклад BVH дерево для швидкого пошуку сутностей.
// Давайте розберемо кожен компонент!

package world

import (
	"errors"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"FlowyCore/world/internal/bvh"
	"github.com/Tnze/go-mc/level"
	"github.com/Tnze/go-mc/level/block"
)

// World - головна структура, що представляє ігровий світ
// Містить всі компоненти для роботи світу та керування ним
type World struct {
	log           *zap.Logger   // логер для відлагодження
	config        Config        // конфігурація світу
	chunkProvider ChunkProvider // провайдер для завантаження чанків

	chunks   map[[2]int32]*LoadedChunk // завантажені чанки
	loaders  map[ChunkViewer]*loader   // завантажувачі чанків для гравців
	tickLock sync.Mutex                // м'ютекс для синхронізації тіків

	// playerViews - BVH дерево для зберігання зон видимості гравців
	// Використовується для швидкого визначення, яким гравцям надсилати
	// сповіщення про рух сутностей
	playerViews playerViewTree
	players     map[Client]*Player // активні гравці
}

// Config - налаштування світу
type Config struct {
	ViewDistance  int32    // радіус прогрузки в чанках
	SpawnAngle    float32  // кут повороту при спавні
	SpawnPosition [3]int32 // координати точки спавну
}

// playerView - структура для зберігання інформації про видимість гравця
type playerView struct {
	EntityViewer // інтерфейс для сповіщень про сутності
	*Player      // вказівник на гравця
}

// Типи для роботи з BVH деревом
type (
	vec3d          = bvh.Vec3[float64]                     // 3D вектор
	aabb3d         = bvh.AABB[float64, vec3d]              // обмежуючий об'єм
	playerViewNode = bvh.Node[float64, aabb3d, playerView] // вузол дерева
	playerViewTree = bvh.Tree[float64, aabb3d, playerView] // BVH дерево
)

// New створює новий світ з вказаними параметрами
func New(logger *zap.Logger, provider ChunkProvider, config Config) (w *World) {
	w = &World{
		log:           logger,
		config:        config,
		chunks:        make(map[[2]int32]*LoadedChunk),
		loaders:       make(map[ChunkViewer]*loader),
		players:       make(map[Client]*Player),
		chunkProvider: provider,
	}
	go w.tickLoop() // запускаємо цикл оновлення світу
	return
}

// Name повертає ідентифікатор світу
func (w *World) Name() string {
	return "minecraft:overworld"
}

// SpawnPositionAndAngle повертає координати та кут спавну
func (w *World) SpawnPositionAndAngle() ([3]int32, float32) {
	return w.config.SpawnPosition, w.config.SpawnAngle
}

// HashedSeed повертає хеш сіда світу
func (w *World) HashedSeed() [8]byte {
	return [8]byte{}
}

// AddPlayer додає гравця до світу
// Створює для нього завантажувач чанків та додає в BVH дерево
func (w *World) AddPlayer(c Client, p *Player, limiter *rate.Limiter) {
	w.tickLock.Lock()
	defer w.tickLock.Unlock()
	w.loaders[c] = newLoader(p, limiter)
	w.players[c] = p
	p.view = w.playerViews.Insert(p.getView(), playerView{c, p})
}

// RemovePlayer видаляє гравця зі світу
// Вивантажує його чанки та видаляє з BVH дерева
func (w *World) RemovePlayer(c Client, p *Player) {
	w.tickLock.Lock()
	defer w.tickLock.Unlock()
	w.log.Debug("Remove Player",
		zap.Int("loader count", len(w.loaders[c].loaded)),
		zap.Int("world count", len(w.chunks)),
	)
	// Видаляємо гравця з усіх завантажених чанків
	for pos := range w.loaders[c].loaded {
		if !w.chunks[pos].RemoveViewer(c) {
			w.log.Panic("viewer is not found in the loaded chunk")
		}
	}
	delete(w.loaders, c)
	delete(w.players, c)
	// Видаляємо гравця з системи сутностей
	w.playerViews.Delete(p.view)
	w.playerViews.Find(
		bvh.TouchPoint[vec3d, aabb3d](bvh.Vec3[float64](p.Position)),
		func(n *playerViewNode) bool {
			n.Value.ViewRemoveEntities([]int32{p.EntityID})
			delete(n.Value.EntitiesInView, p.EntityID)
			return true
		},
	)
}

// loadChunk завантажує чанк за вказаними координатами
// Якщо чанк не існує - генерує новий порожній чанк
// Повертає:
// - true якщо чанк успішно завантажено або згенеровано
// - false якщо виникла помилка або досягнуто ліміт завантаження
func (w *World) loadChunk(pos [2]int32) bool {
	// Створюємо логер з координатами чанку для зручного дебагу
	logger := w.log.With(zap.Int32("x", pos[0]), zap.Int32("z", pos[1]))
	logger.Debug("Loading chunk")

	// Намагаємось завантажити чанк через провайдер
	c, err := w.chunkProvider.GetChunk(pos)
	if err != nil {
		if errors.Is(err, errChunkNotExist) {
			// Чанк не існує - генеруємо новий
			logger.Debug("Generate chunk")

			// ТИМЧАСОВО: Створюємо порожній чанк заповнений камінням
			// TODO: Додати нормальний генератор світу
			c = level.EmptyChunk(24)                // 24 секції по висоті (384 блоки)
			stone := block.ToStateID[block.Stone{}] // ID блоку каменю
			for s := range c.Sections {             // для кожної секції (16x16x16 блоків)
				for i := 0; i < 16*16*16; i++ { // заповнюємо всі блоки
					c.Sections[s].SetBlock(i, stone)
				}
			}
			c.Status = level.StatusFull // позначаємо чанк як повністю згенерований
			logger.Debug("Created empty chunk", zap.Int("sections", len(c.Sections)))

		} else if !errors.Is(err, ErrReachRateLimit) {
			// Якщо помилка не пов'язана з лімітом завантаження - логуємо її
			logger.Error("GetChunk error", zap.Error(err))
			return false
		}
	}

	// Перевіряємо що чанк не nil після завантаження/генерації
	if c == nil {
		logger.Error("Chunk is nil after loading")
		return false
	}

	// Логуємо успішне завантаження
	logger.Debug("Loaded chunk",
		zap.Int("sections", len(c.Sections)),
		zap.String("status", string(c.Status)))

	// Зберігаємо чанк в мапі завантажених чанків
	w.chunks[pos] = &LoadedChunk{Chunk: c}
	return true
}

// unloadChunk вивантажує чанк та зберігає його
func (w *World) unloadChunk(pos [2]int32) {
	logger := w.log.With(zap.Int32("x", pos[0]), zap.Int32("z", pos[1]))
	logger.Debug("Unloading chunk")
	c, ok := w.chunks[pos]
	if !ok {
		logger.Panic("Unloading an non-exist chunk")
	}
	// Сповіщаємо всіх спостерігачів про вивантаження
	for _, viewer := range c.viewers {
		viewer.ViewChunkUnload(pos)
	}
	// Зберігаємо чанк через провайдер
	err := w.chunkProvider.PutChunk(pos, c.Chunk)
	if err != nil {
		logger.Error("Store chunk data error", zap.Error(err))
	}
	delete(w.chunks, pos)
}

// LoadedChunk - структура завантаженого чанку
type LoadedChunk struct {
	sync.Mutex                 // м'ютекс для синхронізації
	viewers      []ChunkViewer // список спостерігачів
	*level.Chunk               // дані чанку
}

// AddViewer додає нового спостерігача до чанку
// ВАЖЛИВО: Метод панікує якщо спостерігач вже існує!
// Це зроблено для виявлення логічних помилок в коді,
// бо дублювання спостерігачів може призвести до:
// - Подвійної відправки пакетів клієнту
// - Витоку пам'яті
// - Проблем при видаленні спостерігача
func (lc *LoadedChunk) AddViewer(v ChunkViewer) {
	lc.Lock()         // блокуємо доступ до чанку
	defer lc.Unlock() // розблокуємо при виході з функції

	// Перевіряємо чи спостерігач вже є в списку
	for _, v2 := range lc.viewers {
		if v2 == v {
			panic("append an exist viewer") // панікуємо якщо знайшли дублікат
		}
	}

	// Додаємо нового спостерігача в кінець списку
	lc.viewers = append(lc.viewers, v)
}

// RemoveViewer видаляє спостерігача з чанку
// Використовує "swap and pop" алгоритм для ефективного видалення:
// 1. Знаходимо індекс спостерігача якого треба видалити
// 2. Переміщуємо останній елемент на його місце
// 3. Відрізаємо останній елемент зі слайсу
// Повертає:
// - true якщо спостерігач був знайдений і видалений
// - false якщо спостерігач не знайдений
func (lc *LoadedChunk) RemoveViewer(v ChunkViewer) bool {
	lc.Lock()         // блокуємо доступ до чанку
	defer lc.Unlock() // розблокуємо при виході з функції

	for i, v2 := range lc.viewers { // шукаємо спостерігача
		if v2 == v { // знайшли!
			last := len(lc.viewers) - 1      // індекс останнього елемента
			lc.viewers[i] = lc.viewers[last] // переміщуємо останній на місце видаленого
			lc.viewers = lc.viewers[:last]   // відрізаємо останній елемент
			return true                      // успішно видалили
		}
	}
	return false // спостерігач не знайдений
}
