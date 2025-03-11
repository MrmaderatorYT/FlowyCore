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

// Йоу, чат! Сьогодні ми розберемо як працює система спостерігачів у нашому сервері!
// Це дуже важлива частина, яка відповідає за те, що бачать гравці:
// чанки, інших гравців, їх рухи тощо. Тут у нас є кілька інтерфейсів,
// які описують різні аспекти того, що може бачити гравець.
// Давайте розберемо кожен з них!

package world

import (
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/level"
)

// Client - головний інтерфейс для взаємодії з клієнтом гравця
// Об'єднує в собі можливості бачити чанки (ChunkViewer) та сутності (EntityViewer),
// а також базові операції як відключення та телепортація
type Client interface {
	ChunkViewer                                                           // для роботи з чанками
	EntityViewer                                                          // для роботи з сутностями
	SendDisconnect(reason chat.Message)                                   // відправити повідомлення про відключення
	SendPlayerPosition(pos [3]float64, rot [2]float32) (teleportID int32) // телепортувати гравця
	SendSetChunkCacheCenter(chunkPos [2]int32)                            // встановити центр завантаження чанків
}

// ChunkViewer - інтерфейс для роботи з чанками
// Описує методи для завантаження та вивантаження чанків,
// які видно гравцю в радіусі прогрузки
type ChunkViewer interface {
	ViewChunkLoad(pos level.ChunkPos, c *level.Chunk) // завантажити чанк
	ViewChunkUnload(pos level.ChunkPos)               // вивантажити чанк
}

// EntityViewer - інтерфейс для роботи з сутностями
// Містить методи для відображення всіх можливих дій сутностей:
// появи, зникнення, руху, повороту голови тощо
type EntityViewer interface {
	ViewAddPlayer(p *Player)                                                      // додати гравця в зону видимості
	ViewRemoveEntities(entityIDs []int32)                                         // видалити сутності
	ViewMoveEntityPos(id int32, delta [3]int16, onGround bool)                    // рух сутності
	ViewMoveEntityPosAndRot(id int32, delta [3]int16, rot [2]int8, onGround bool) // рух + поворот
	ViewMoveEntityRot(id int32, rot [2]int8, onGround bool)                       // поворот сутності
	ViewRotateHead(id int32, yaw int8)                                            // поворот голови
	ViewTeleportEntity(id int32, pos [3]float64, rot [2]int8, onGround bool)      // телепортація
}
