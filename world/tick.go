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

// Йоу, чат! Сьогодні ми розберемо як працює система тіків у нашому сервері!
// Тік - це основна одиниця часу в Minecraft, що триває 50мс (1/20 секунди).
// За цей час сервер оновлює стан світу: рухає сутності, завантажує чанки,
// синхронізує позиції гравців тощо. Давайте розберемо як це працює!

package world

import (
	"github.com/Tnze/go-mc/chat"
	"math"
	"time"

	"go.uber.org/zap"

	"FlowyCore/world/internal/bvh"
)

// tickLoop запускає головний цикл оновлення світу
// Викликається кожні 50мс (20 разів на секунду)
func (w *World) tickLoop() {
	var n uint
	for range time.Tick(time.Microsecond * 20) {
		w.tick(n)
		n++
	}
}

// tick виконує одне оновлення світу
// Розділений на підтіки для різних систем
func (w *World) tick(n uint) {
	w.tickLock.Lock()
	defer w.tickLock.Unlock()

	if n%8 == 0 { // кожен 8-й тік (4 рази на секунду)
		w.subtickChunkLoad() // оновлюємо завантаження чанків
	}
	w.subtickUpdatePlayers()  // оновлюємо стан гравців
	w.subtickUpdateEntities() // оновлюємо стан сутностей
}

// subtickChunkLoad відповідає за завантаження та вивантаження чанків
// Викликається 4 рази на секунду для оптимізації навантаження
func (w *World) subtickChunkLoad() {
	// Оновлюємо центр завантаження для кожного гравця
	for c, p := range w.players {
		x := int32(p.Position[0]) >> 4 // конвертуємо координати в чанки
		y := int32(p.Position[1]) >> 4 // діленням на 16 (зсув на 4)
		z := int32(p.Position[2]) >> 4
		if newChunkPos := [3]int32{x, y, z}; newChunkPos != p.ChunkPos {
			p.ChunkPos = newChunkPos
			c.SendSetChunkCacheCenter([2]int32{x, z})
		}
	}

	// Завантажуємо нові чанки для кожного гравця
LoadChunk:
	for viewer, loader := range w.loaders {
		loader.calcLoadingQueue() // розраховуємо які чанки потрібно завантажити
		for _, pos := range loader.loadQueue {
			if !loader.limiter.Allow() { // перевіряємо ліміт завантаження
				break
			}
			if _, ok := w.chunks[pos]; !ok {
				if !w.loadChunk(pos) {
					break LoadChunk // досягнуто глобальний ліміт
				}
			}
			loader.loaded[pos] = struct{}{}
			lc := w.chunks[pos]
			lc.AddViewer(viewer)
			lc.Lock()

			// Перевіряємо чанк перед відправкою
			if lc.Chunk == nil {
				w.log.Error("Chunk is nil before ViewChunkLoad",
					zap.Int32("x", pos[0]),
					zap.Int32("z", pos[1]))
			} else {
				w.log.Debug("Sending chunk to viewer",
					zap.Int32("x", pos[0]),
					zap.Int32("z", pos[1]),
					zap.Int("sections", len(lc.Chunk.Sections)),
					zap.String("status", string(lc.Chunk.Status)))
			}

			viewer.ViewChunkLoad(pos, lc.Chunk)
			lc.Unlock()
		}
	}

	// Вивантажуємо непотрібні чанки
	for viewer, loader := range w.loaders {
		loader.calcUnusedChunks() // шукаємо чанки поза зоною видимості
		for _, pos := range loader.unloadQueue {
			delete(loader.loaded, pos)
			if !w.chunks[pos].RemoveViewer(viewer) {
				w.log.Panic("viewer is not found in the loaded chunk")
			}
			viewer.ViewChunkUnload(pos)
		}
	}

	// Вивантажуємо чанки без спостерігачів
	var unloadQueue [][2]int32
	for pos, chunk := range w.chunks {
		if len(chunk.viewers) == 0 {
			unloadQueue = append(unloadQueue, pos)
		}
	}
	for i := range unloadQueue {
		w.unloadChunk(unloadQueue[i])
	}
}

// subtickUpdatePlayers оновлює стан всіх гравців
// Обробляє рух, телепортацію та зону видимості
func (w *World) subtickUpdatePlayers() {
	for c, p := range w.players {
		if !p.Inputs.TryLock() {
			continue
		}
		inputs := &p.Inputs

		// Оновлюємо радіус видимості
		if p.ViewDistance != int32(inputs.ViewDistance) {
			p.ViewDistance = int32(inputs.ViewDistance)
			p.view = w.playerViews.Insert(p.getView(), w.playerViews.Delete(p.view))
		}

		// Видаляємо сутності поза зоною видимості
		for id, e := range p.EntitiesInView {
			if !p.view.Box.WithIn(vec3d(e.Position)) {
				delete(p.EntitiesInView, id)
				p.view.Value.ViewRemoveEntities([]int32{id})
			}
		}

		// Обробляємо телепортацію або рух
		if p.teleport != nil {
			if inputs.TeleportID == p.teleport.ID {
				p.pos0 = p.teleport.Position
				p.rot0 = p.teleport.Rotation
				p.teleport = nil
			}
		} else {
			// Перевіряємо швидкість руху
			delta := [3]float64{
				inputs.Position[0] - p.Position[0],
				inputs.Position[1] - p.Position[1],
				inputs.Position[2] - p.Position[2],
			}
			distance := math.Sqrt(delta[0]*delta[0] + delta[1]*delta[1] + delta[2]*delta[2])
			if distance > 100 {
				// Завелика швидкість - можливий чіт
				teleportID := c.SendPlayerPosition(p.Position, p.Rotation)
				p.teleport = &TeleportRequest{
					ID:       teleportID,
					Position: p.Position,
					Rotation: p.Rotation,
				}
			} else if inputs.Position.IsValid() {
				p.pos0 = inputs.Position
				p.rot0 = inputs.Rotation
				p.OnGround = inputs.OnGround
			} else {
				w.log.Info("Player move invalid",
					zap.Float64("x", inputs.Position[0]),
					zap.Float64("y", inputs.Position[1]),
					zap.Float64("z", inputs.Position[2]),
				)
				c.SendDisconnect(chat.TranslateMsg("multiplayer.disconnect.invalid_player_movement"))
			}
		}
		p.Inputs.Unlock()
	}
}

// subtickUpdateEntities оновлює стан всіх сутностей
// Наразі обробляє тільки гравців, бо інших сутностей ще немає
func (w *World) subtickUpdateEntities() {
	for _, e := range w.players {
		// Розраховуємо дельту позиції та повороту
		var delta [3]int16
		var rot [2]int8
		if e.Position != e.pos0 { // TODO: відправляти пакет телепортації якщо відстань > 8
			delta = [3]int16{
				int16((e.pos0[0] - e.Position[0]) * 32 * 128),
				int16((e.pos0[1] - e.Position[1]) * 32 * 128),
				int16((e.pos0[2] - e.Position[2]) * 32 * 128),
			}
		}
		if e.Rotation != e.rot0 {
			rot = [2]int8{
				int8(e.rot0[0] * 256 / 360),
				int8(e.rot0[1] * 256 / 360),
			}
		}

		// Шукаємо гравців у зоні видимості
		cond := bvh.TouchPoint[vec3d, aabb3d](vec3d(e.Position))
		w.playerViews.Find(cond,
			func(n *playerViewNode) bool {
				if n.Value.Player == e {
					return true // не надсилаємо гравцю його власні рухи
				}
				// Додаємо сутність в список видимих
				if _, ok := n.Value.EntitiesInView[e.EntityID]; !ok {
					n.Value.ViewAddPlayer(e)
					n.Value.EntitiesInView[e.EntityID] = &e.Entity
				}
				return true
			},
		)

		// Вибираємо тип пакету руху
		var sendMove func(v EntityViewer)
		switch {
		case e.Position != e.pos0 && e.Rotation != e.rot0:
			sendMove = func(v EntityViewer) {
				v.ViewMoveEntityPosAndRot(e.EntityID, delta, rot, bool(e.OnGround))
				v.ViewRotateHead(e.EntityID, rot[0])
			}
		case e.Position != e.pos0:
			sendMove = func(v EntityViewer) {
				v.ViewMoveEntityPos(e.EntityID, delta, bool(e.OnGround))
			}
		case e.Rotation != e.rot0:
			sendMove = func(v EntityViewer) {
				v.ViewMoveEntityRot(e.EntityID, rot, bool(e.OnGround))
				v.ViewRotateHead(e.EntityID, rot[0])
			}
		default:
			continue
		}

		// Оновлюємо позицію
		e.Position = e.pos0
		e.Rotation = e.rot0

		// Надсилаємо оновлення всім гравцям в зоні видимості
		w.playerViews.Find(cond,
			func(n *playerViewNode) bool {
				if n.Value.Player == e {
					return true // пропускаємо самого гравця
				}
				if _, ok := n.Value.EntitiesInView[e.EntityID]; ok {
					sendMove(n.Value.EntityViewer)
				} else {
					n.Value.ViewAddPlayer(e)
					n.Value.EntitiesInView[e.EntityID] = &e.Entity
				}
				return true
			},
		)
	}
}
