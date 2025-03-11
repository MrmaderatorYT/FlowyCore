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

// Йоу, чат! Сьогодні ми розберемо як влаштований гравець у нашому сервері!
// Це один з найважливіших файлів, бо він описує всю інформацію про гравця:
// його позицію, інвентар, налаштування клієнта і багато іншого.
// Давайте розберемо кожну структуру детально!

package world

import (
	"io"
	"sync"
	"time"

	"github.com/google/uuid"

	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/yggdrasil/user"
)

// ReadFrom читає інформацію про клієнт з мережевого потоку
// Використовує пакетний формат Minecraft для десеріалізації даних
func (i *ClientInfo) ReadFrom(r io.Reader) (n int64, err error) {
	return pk.Tuple{
		(*pk.String)(&i.Locale),                   // мова клієнта
		(*pk.Byte)(&i.ViewDistance),               // дальність прогрузки
		(*pk.VarInt)(&i.ChatMode),                 // налаштування чату
		(*pk.Boolean)(&i.ChatColors),              // кольоровий чат
		(*pk.UnsignedByte)(&i.DisplayedSkinParts), // видимі частини скіну
		(*pk.VarInt)(&i.MainHand),                 // основна рука
		(*pk.Boolean)(&i.EnableTextFiltering),     // фільтрація чату
		(*pk.Boolean)(&i.AllowServerListings),     // показ у списку серверів
	}.ReadFrom(r)
}

// Player - основна структура, що представляє гравця
// Містить всю інформацію про стан гравця у грі
type Player struct {
	Entity                     // наслідуємо базові поля сутності
	Name       string          // нікнейм гравця
	UUID       uuid.UUID       // унікальний ідентифікатор
	PubKey     *user.PublicKey // публічний ключ для верифікації
	Properties []user.Property // додаткові властивості (скін, плащ)
	Latency    time.Duration   // затримка з'єднання

	lastChatTimestamp time.Time // час останнього повідомлення
	lastChatSignature []byte    // підпис останнього повідомлення

	ChunkPos     [3]int32 // позиція в координатах чанків
	ViewDistance int32    // радіус прогрузки в чанках

	Gamemode       int32             // режим гри (0-виживання, 1-креатив...)
	EntitiesInView map[int32]*Entity // сутності в зоні видимості
	view           *playerViewNode   // вузол для оптимізації видимості
	teleport       *TeleportRequest  // запит на телепортацію

	Inputs Inputs // поточний стан вводу від клієнта
}

// chunkPosition повертає 2D координати чанка гравця
// Використовується для завантаження чанків навколо
func (p *Player) chunkPosition() [2]int32 { return [2]int32{p.ChunkPos[0], p.ChunkPos[2]} }

// chunkRadius повертає радіус прогрузки в чанках
// Визначає скільки чанків навколо гравця буде завантажено
func (p *Player) chunkRadius() int32 { return p.ViewDistance }

// getView розраховує куб видимості гравця
// Використовується для визначення які сутності видно гравцю
func (p *Player) getView() aabb3d {
	viewDistance := float64(p.ViewDistance) * 16 // переводимо чанки в блоки
	return aabb3d{
		Upper: vec3d{p.Position[0] + viewDistance, p.Position[1] + viewDistance, p.Position[2] + viewDistance},
		Lower: vec3d{p.Position[0] - viewDistance, p.Position[1] - viewDistance, p.Position[2] - viewDistance},
	}
}

// TeleportRequest - запит на телепортацію гравця
// Використовується для синхронізації позиції з клієнтом
type TeleportRequest struct {
	ID       int32 // унікальний ID телепортації
	Position       // нова позиція
	Rotation       // новий кут повороту
}

// Inputs - структура для зберігання стану вводу від клієнта
// Захищена мютексом для безпечного доступу з різних горутин
type Inputs struct {
	sync.Mutex               // захист від гонки даних
	ClientInfo               // налаштування клієнта
	Position                 // поточна позиція
	Rotation                 // поточний поворот
	OnGround                 // чи на землі
	Latency    time.Duration // затримка
	TeleportID int32         // ID останньої телепортації
}

// ClientInfo - налаштування та можливості клієнта
// Отримуються при підключенні гравця
type ClientInfo struct {
	Locale              string // мова клієнта
	ViewDistance        int8   // радіус прогрузки
	ChatMode            int32  // режим чату
	ChatColors          bool   // підтримка кольорів
	DisplayedSkinParts  byte   // видимі частини скіну
	MainHand            int32  // основна рука (0-ліва, 1-права)
	EnableTextFiltering bool   // фільтрація чату
	AllowServerListings bool   // дозвіл показу в списку
}
