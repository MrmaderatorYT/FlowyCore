// Йоу, чат! Зараз розберемо як працює клієнт в нашому сервері!
// Це основний файл який керує з'єднанням з гравцем

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
	// zap - крутий логер для Go
	"go.uber.org/zap"

	// Наші та зовнішні пакети
	"FlowyCore/world"
	"github.com/Tnze/go-mc/data/packetid"
	"github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/net/queue"
	"github.com/Tnze/go-mc/server"
)

// Client представляє підключеного гравця
// Він обробляє всі пакети які приходять від клієнта і відправляє пакети назад
type Client struct {
	// Логер для цього клієнта
	log *zap.Logger
	// Мережеве з'єднання
	conn *net.Conn
	// Дані гравця (позиція, інвентар і т.д.)
	player *world.Player
	// Черга пакетів для відправки
	queue server.PacketQueue
	// Обробники різних типів пакетів
	handlers []PacketHandler
	// Вказівник на Input гравця (кнопки, миша)
	// Використовуємо вбудоване поле щоб не писати c.player.Inputs
	*world.Inputs
}

// PacketHandler - функція яка обробляє конкретний тип пакету
type PacketHandler func(p pk.Packet, c *Client) error

// New створює нового клієнта
func New(log *zap.Logger, conn *net.Conn, player *world.Player) *Client {
	return &Client{
		log:    log,
		conn:   conn,
		player: player,
		// Черга на 256 пакетів
		queue: queue.NewChannelQueue[pk.Packet](256),
		// Копіюємо стандартні обробники
		handlers: defaultHandlers[:],
		// Вказівник на інпути гравця
		Inputs: &player.Inputs,
	}
}

// Start запускає обробку пакетів
// Створює дві горутини - для відправки і отримання
func (c *Client) Start() {
	// Канал для синхронізації завершення горутин
	stopped := make(chan struct{}, 2)
	done := func() {
		stopped <- struct{}{}
	}
	// Запускаємо горутини
	// Якщо будь-яка з них впаде - інша теж зупиниться
	go c.startSend(done)
	go c.startReceive(done)
	// Чекаємо поки одна з горутин завершиться
	<-stopped
}

// startSend відправляє пакети клієнту
func (c *Client) startSend(done func()) {
	defer done()
	for {
		// Беремо пакет з черги
		p, ok := c.queue.Pull()
		if !ok {
			return
		}
		// Відправляємо його
		err := c.conn.WritePacket(p)
		if err != nil {
			c.log.Debug("Send packet fail", zap.Error(err))
			return
		}
		// Якщо це пакет disconnect - виходимо
		if packetid.ClientboundPacketID(p.ID) == packetid.ClientboundDisconnect {
			return
		}
	}
}

// startReceive отримує пакети від клієнта
func (c *Client) startReceive(done func()) {
	defer done()
	var packet pk.Packet
	for {
		// Читаємо пакет
		err := c.conn.ReadPacket(&packet)
		if err != nil {
			c.log.Debug("Receive packet fail", zap.Error(err))
			return
		}
		// Перевіряємо що ID пакету валідний
		if packet.ID < 0 || packet.ID >= int32(len(c.handlers)) {
			c.log.Debug("Invalid packet id", zap.Int32("id", packet.ID), zap.Int("len", len(packet.Data)))
			return
		}
		// Якщо є обробник для цього типу пакету - викликаємо його
		if handler := c.handlers[packet.ID]; handler != nil {
			err = handler(packet, c)
			if err != nil {
				c.log.Error("Handle packet error", zap.Int32("id", packet.ID), zap.Error(err))
				return
			}
		}
	}
}

// AddHandler додає новий обробник пакетів
func (c *Client) AddHandler(id packetid.ServerboundPacketID, handler PacketHandler) {
	c.handlers[id] = handler
}

// GetPlayer повертає дані гравця
func (c *Client) GetPlayer() *world.Player { return c.player }

// defaultHandlers - стандартні обробники пакетів
// Кожен обробник відповідає за свій тип пакету:
var defaultHandlers = [packetid.ServerboundPacketIDGuard]PacketHandler{
	// Підтвердження телепортації
	packetid.ServerboundAcceptTeleportation: clientAcceptTeleportation,
	// Налаштування клієнта (мова, дальність прогрузки і т.д.)
	packetid.ServerboundClientInformation: clientInformation,
	// Рух гравця:
	// - Тільки позиція
	packetid.ServerboundMovePlayerPos: clientMovePlayerPos,
	// - Позиція + поворот
	packetid.ServerboundMovePlayerPosRot: clientMovePlayerPosRot,
	// - Тільки поворот
	packetid.ServerboundMovePlayerRot: clientMovePlayerRot,
	// - Тільки стан "на землі"
	packetid.ServerboundMovePlayerStatusOnly: clientMovePlayerStatusOnly,
	// Рух транспорту (поки не реалізовано)
	packetid.ServerboundMoveVehicle: clientMoveVehicle,
}
