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

package client

import (
	"bytes"
	"encoding/binary"
	"sync/atomic"
	"unsafe"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"FlowyCore/world"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/chat/sign"
	"github.com/Tnze/go-mc/data/packetid"
	"github.com/Tnze/go-mc/level"
	pk "github.com/Tnze/go-mc/net/packet"
)

func (c *Client) SendPacket(id packetid.ClientboundPacketID, fields ...pk.FieldEncoder) {
	var buffer bytes.Buffer

	// Write the packet fields
	for i := range fields {
		if _, err := fields[i].WriteTo(&buffer); err != nil {
			c.log.Panic("Marshal packet error", zap.Error(err))
		}
	}

	// Send the packet data
	c.queue.Push(pk.Packet{
		ID:   int32(id),
		Data: buffer.Bytes(),
	})
}

func (c *Client) SendKeepAlive(id int64) {
	c.SendPacket(packetid.ClientboundKeepAlive, pk.Long(id))
}

// SendDisconnect send ClientboundDisconnect packet to client.
// Once the packet is sent, the connection will be closed.
func (c *Client) SendDisconnect(reason chat.Message) {
	c.log.Debug("Disconnect player", zap.String("reason", reason.ClearString()))
	c.SendPacket(packetid.ClientboundDisconnect, reason)
}

func (c *Client) SendLogin(w *world.World, p *world.Player) {
	hashedSeed := w.HashedSeed()
	c.SendPacket(
		packetid.ClientboundLogin,
		pk.Int(p.EntityID),
		pk.Boolean(false), // Is Hardcore
		pk.Byte(p.Gamemode),
		pk.Byte(-1),
		pk.Array([]pk.Identifier{
			pk.Identifier(w.Name()),
		}),
		pk.NBT(world.NetworkCodec),
		pk.Identifier("minecraft:overworld"),
		pk.Identifier(w.Name()),
		pk.Long(binary.BigEndian.Uint64(hashedSeed[:8])),
		pk.VarInt(0),              // Max players (ignored by client)
		pk.VarInt(p.ViewDistance), // View Distance
		pk.VarInt(p.ViewDistance), // Simulation Distance
		pk.Boolean(false),         // Reduced Debug Info
		pk.Boolean(false),         // Enable respawn screen
		pk.Boolean(false),         // Is Debug
		pk.Boolean(false),         // Is Flat
		pk.Boolean(false),         // Has Last Death Location
	)
}

func (c *Client) SendServerData(motd *chat.Message, favIcon string, enforceSecureProfile bool) {
	c.SendPacket(
		packetid.ClientboundServerData,
		motd,
		pk.Option[pk.String, *pk.String]{
			Has: favIcon != "",
			Val: pk.String(favIcon),
		},
		pk.Boolean(enforceSecureProfile),
	)
}

// Actions of [SendPlayerInfoUpdate]
const (
	PlayerInfoAddPlayer = iota
	PlayerInfoInitializeChat
	PlayerInfoUpdateGameMode
	PlayerInfoUpdateListed
	PlayerInfoUpdateLatency
	PlayerInfoUpdateDisplayName
	// PlayerInfoEnumGuard is the number of the enums
	PlayerInfoEnumGuard
)

func NewPlayerInfoAction(actions ...int) pk.FixedBitSet {
	enumSet := pk.NewFixedBitSet(PlayerInfoEnumGuard)
	for _, action := range actions {
		enumSet.Set(action, true)
	}
	return enumSet
}

func (c *Client) SendPlayerInfoUpdate(actions pk.FixedBitSet, players []*world.Player) {
	var buf bytes.Buffer
	_, _ = actions.WriteTo(&buf)
	_, _ = pk.VarInt(len(players)).WriteTo(&buf)
	for _, player := range players {
		_, _ = pk.UUID(player.UUID).WriteTo(&buf)
		if actions.Get(PlayerInfoAddPlayer) {
			_, _ = pk.String(player.Name).WriteTo(&buf)
			_, _ = pk.Array(player.Properties).WriteTo(&buf)
		}
		if actions.Get(PlayerInfoInitializeChat) {
			panic("not yet support InitializeChat")
		}
		if actions.Get(PlayerInfoUpdateGameMode) {
			_, _ = pk.VarInt(player.Gamemode).WriteTo(&buf)
		}
		if actions.Get(PlayerInfoUpdateListed) {
			_, _ = pk.Boolean(true).WriteTo(&buf)
		}
		if actions.Get(PlayerInfoUpdateLatency) {
			_, _ = pk.VarInt(player.Latency.Milliseconds()).WriteTo(&buf)
		}
		if actions.Get(PlayerInfoUpdateDisplayName) {
			panic("not yet support DisplayName")
		}
	}
	c.queue.Push(pk.Packet{
		ID:   int32(packetid.ClientboundPlayerInfoUpdate),
		Data: buf.Bytes(),
	})
}

func (c *Client) SendPlayerInfoRemove(players []*world.Player) {
	var buff bytes.Buffer

	if _, err := pk.VarInt(len(players)).WriteTo(&buff); err != nil {
		c.log.Panic("Marshal packet error", zap.Error(err))
	}
	for _, p := range players {
		if _, err := pk.UUID(p.UUID).WriteTo(&buff); err != nil {
			c.log.Panic("Marshal packet error", zap.Error(err))
		}
	}

	c.queue.Push(pk.Packet{
		ID:   int32(packetid.ClientboundPlayerInfoRemove),
		Data: buff.Bytes(),
	})
}

func (c *Client) SendLevelChunkWithLight(pos level.ChunkPos, chunk *level.Chunk) {
	c.SendPacket(packetid.ClientboundLevelChunkWithLight, pos, chunk)
}

func (c *Client) SendForgetLevelChunk(pos level.ChunkPos) {
	c.SendPacket(packetid.ClientboundForgetLevelChunk, pos)
}

func (c *Client) SendAddPlayer(p *world.Player) {
	c.SendPacket(
		packetid.ClientboundAddPlayer,
		pk.VarInt(p.EntityID),
		pk.UUID(p.UUID),
		pk.Double(p.Position[0]),
		pk.Double(p.Position[1]),
		pk.Double(p.Position[2]),
		pk.Angle(p.Rotation[0]),
		pk.Angle(p.Rotation[1]),
	)
}

// Йоу, чат! Зараз розберемо як працює рух сутностей в майнкрафті!
// В майні є кілька типів руху - звичайний рух, телепортація і поворот голови
// Для економії трафіку використовуються різні формати даних

// SendMoveEntitiesPos відправляє відносний рух сутності
// Використовується коли сутність просто йде, без повороту
func (c *Client) SendMoveEntitiesPos(eid int32, delta [3]int16, onGround bool) {
	// Відправляємо пакет руху
	c.SendPacket(
		// ID пакету руху сутності
		packetid.ClientboundMoveEntityPos,
		// ID сутності яка рухається
		pk.VarInt(eid),
		// Зміщення по X відносно поточної позиції (в 1/4096 блока)
		pk.Short(delta[0]),
		// Зміщення по Y - можемо рухатись вверх-вниз
		pk.Short(delta[1]),
		// Зміщення по Z
		pk.Short(delta[2]),
		// Чи стоїть сутність на землі - важливо для анімації
		pk.Boolean(onGround),
	)
}

// SendMoveEntitiesPosAndRot відправляє рух з поворотом
// Використовується коли сутність йде і одночасно повертається
func (c *Client) SendMoveEntitiesPosAndRot(eid int32, delta [3]int16, rot [2]int8, onGround bool) {
	c.SendPacket(
		// ID пакету руху з поворотом
		packetid.ClientboundMoveEntityPosRot,
		// ID сутності
		pk.VarInt(eid),
		// Зміщення по X, Y, Z як і в звичайному русі
		pk.Short(delta[0]),
		pk.Short(delta[1]),
		pk.Short(delta[2]),
		// Новий кут повороту вліво-вправо
		pk.Angle(rot[0]),
		// Новий кут нахилу голови
		pk.Angle(rot[1]),
		// Чи на землі
		pk.Boolean(onGround),
	)
}

// SendMoveEntitiesRot відправляє тільки поворот
// Коли сутність стоїть на місці і тільки крутиться
func (c *Client) SendMoveEntitiesRot(eid int32, rot [2]int8, onGround bool) {
	c.SendPacket(
		// ID пакету повороту
		packetid.ClientboundMoveEntityRot,
		// ID сутності
		pk.VarInt(eid),
		// Нові кути повороту
		pk.Angle(rot[0]),
		pk.Angle(rot[1]),
		// Чи на землі
		pk.Boolean(onGround),
	)
}

// SendRotateHead відправляє поворот голови окремо від тіла
// В майні голова може крутитись незалежно від тіла
func (c *Client) SendRotateHead(eid int32, yaw int8) {
	c.SendPacket(
		// ID пакету повороту голови
		packetid.ClientboundRotateHead,
		// ID сутності
		pk.VarInt(eid),
		// Кут повороту голови (тільки вліво-вправо)
		pk.Angle(yaw),
	)
}

// SendTeleportEntity телепортує сутність на нові координати
// Використовується для великих переміщень, коли відносний рух не підходить
func (c *Client) SendTeleportEntity(eid int32, pos [3]float64, rot [2]int8, onGround bool) {
	c.SendPacket(
		// ID пакету телепортації
		packetid.ClientboundTeleportEntity,
		// ID сутності
		pk.VarInt(eid),
		// Абсолютні координати X, Y, Z
		// Використовуємо double для точності
		pk.Double(pos[0]),
		pk.Double(pos[1]),
		pk.Double(pos[2]),
		// Нові кути повороту
		pk.Angle(rot[0]),
		pk.Angle(rot[1]),
		// Чи на землі після телепорту
		pk.Boolean(onGround),
	)
}

// Лічильник для ID телепортацій
// Atomic щоб безпечно використовувати з різних потоків
var teleportCounter atomic.Int32

// SendPlayerPosition телепортує самого гравця
// Найскладніший тип телепортації, потребує підтвердження від клієнта
func (c *Client) SendPlayerPosition(pos [3]float64, rot [2]float32) (teleportID int32) {
	// Генеруємо новий ID для цієї телепортації
	teleportID = teleportCounter.Add(1)

	c.SendPacket(
		// ID пакету телепортації гравця
		packetid.ClientboundPlayerPosition,
		// Нові координати X, Y, Z
		pk.Double(pos[0]),
		pk.Double(pos[1]),
		pk.Double(pos[2]),
		// Нові кути повороту
		pk.Float(rot[0]),
		pk.Float(rot[1]),
		// Флаги телепортації (0 = абсолютні координати)
		pk.Byte(0),
		// ID телепортації для підтвердження
		pk.VarInt(teleportID),
	)
	return
}

// SendSetDefaultSpawnPosition встановлює точку відродження
// Сюди гравець потрапить після смерті
func (c *Client) SendSetDefaultSpawnPosition(xyz [3]int32, angle float32) {
	c.SendPacket(
		// ID пакету встановлення спавну
		packetid.ClientboundSetDefaultSpawnPosition,
		// Координати спавну (цілі числа)
		pk.Position{X: int(xyz[0]), Y: int(xyz[1]), Z: int(xyz[2])},
		// Кут повороту при появі
		pk.Float(angle),
	)
	return
}

// SendRemoveEntities видаляє сутності зі світу
// Використовується коли сутності виходять з радіусу видимості
func (c *Client) SendRemoveEntities(entityIDs []int32) {
	c.SendPacket(
		// ID пакету видалення сутностей
		packetid.ClientboundRemoveEntities,
		// Список ID сутностей для видалення
		// Використовуємо unsafe.Pointer для конвертації типів без копіювання
		pk.Array(*(*[]pk.VarInt)(unsafe.Pointer(&entityIDs))),
	)
}

// SendSystemChat відправляє системне повідомлення
// Наприклад "Гравець приєднався" або "Сервер перезавантажується"
func (c *Client) SendSystemChat(msg chat.Message, overlay bool) {
	c.SendPacket(
		// ID пакету системного чату
		packetid.ClientboundSystemChat,
		// Текст повідомлення
		msg,
		// Чи показувати як оверлей (над хотбаром)
		pk.Boolean(overlay),
	)
}

func (c *Client) SendPlayerChat(
	sender uuid.UUID,
	index int32,
	signature pk.Option[sign.Signature, *sign.Signature],
	body *sign.PackedMessageBody,
	unsignedContent *chat.Message,
	filter *sign.FilterMask,
	chatType *chat.Type,
) {
	c.SendPacket(
		packetid.ClientboundPlayerChat,
		pk.UUID(sender),
		pk.VarInt(index),
		signature,
		body,
		pk.OptionEncoder[*chat.Message]{
			Has: unsignedContent != nil,
			Val: unsignedContent,
		},
		filter,
		chatType,
	)
}

func (c *Client) SendSetChunkCacheCenter(chunkPos [2]int32) {
	c.SendPacket(
		packetid.ClientboundSetChunkCacheCenter,
		pk.VarInt(chunkPos[0]),
		pk.VarInt(chunkPos[1]),
	)
}

func (c *Client) ViewChunkLoad(pos level.ChunkPos, chunk *level.Chunk) {
	c.SendLevelChunkWithLight(pos, chunk)
}
func (c *Client) ViewChunkUnload(pos level.ChunkPos)   { c.SendForgetLevelChunk(pos) }
func (c *Client) ViewAddPlayer(p *world.Player)        { c.SendAddPlayer(p) }
func (c *Client) ViewRemoveEntities(entityIDs []int32) { c.SendRemoveEntities(entityIDs) }
func (c *Client) ViewMoveEntityPos(id int32, delta [3]int16, onGround bool) {
	c.SendMoveEntitiesPos(id, delta, onGround)
}

func (c *Client) ViewMoveEntityPosAndRot(id int32, delta [3]int16, rot [2]int8, onGround bool) {
	c.SendMoveEntitiesPosAndRot(id, delta, rot, onGround)
}

func (c *Client) ViewMoveEntityRot(id int32, rot [2]int8, onGround bool) {
	c.SendMoveEntitiesRot(id, rot, onGround)
}

func (c *Client) ViewRotateHead(id int32, yaw int8) {
	c.SendRotateHead(id, yaw)
}

func (c *Client) ViewTeleportEntity(id int32, pos [3]float64, rot [2]int8, onGround bool) {
	c.SendTeleportEntity(id, pos, rot, onGround)
}
