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

// Йоу, чат! Зараз розберемо як працює список гравців на сервері!
// Це важлива частина серверу, яка відповідає за:
// - Відображення гравців у табі (Tab клавіша)
// - Перевірку з'єднання (пінг)
// - Оновлення інформації про гравців

package game

import (
	"time"

	"FlowyCore/client"
	"FlowyCore/world"
	"github.com/Tnze/go-mc/data/packetid"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/server"
)

// playerList керує списком гравців на сервері
type playerList struct {
	// keepAlive перевіряє чи гравці ще підключені
	keepAlive *server.KeepAlive
	// pingList зберігає список всіх гравців і їх пінг
	pingList *server.PlayerList
}

// addPlayer додає нового гравця до списку
// Відправляє інформацію про нового гравця всім іншим
// І відправляє новому гравцю інфу про всіх інших
func (pl *playerList) addPlayer(c *client.Client, p *world.Player) {
	// Додаємо гравця в список для відображення в табі
	pl.pingList.ClientJoin(c, server.PlayerSample{
		Name: p.Name,
		ID:   p.UUID,
	})
	pl.keepAlive.ClientJoin(c)
	c.AddHandler(packetid.ServerboundKeepAlive, keepAliveHandler(pl.keepAlive))
	players := make([]*world.Player, 0, pl.pingList.Len()+1)
	players = append(players, p)
	addPlayerAction := client.NewPlayerInfoAction(
		client.PlayerInfoAddPlayer,
		client.PlayerInfoUpdateListed,
	)
	pl.pingList.Range(func(c server.PlayerListClient, _ server.PlayerSample) {
		cc := c.(*client.Client)
		cc.SendPlayerInfoUpdate(addPlayerAction, []*world.Player{p})
		players = append(players, cc.GetPlayer())
	})
	c.SendPlayerInfoUpdate(addPlayerAction, players)
}

// updateLatency оновлює пінг гравця
// І відправляє цю інфу всім іншим гравцям
func (pl *playerList) updateLatency(c *client.Client, latency time.Duration) {
	updateLatencyAction := client.NewPlayerInfoAction(client.PlayerInfoUpdateLatency)
	p := c.GetPlayer()
	p.Inputs.Lock()
	p.Inputs.Latency = latency
	p.Inputs.Unlock()
	pl.pingList.Range(func(c server.PlayerListClient, _ server.PlayerSample) {
		c.(*client.Client).SendPlayerInfoUpdate(updateLatencyAction, []*world.Player{p})
	})
}

// removePlayer видаляє гравця зі списку
// І повідомляє про це всіх інших гравців
func (pl *playerList) removePlayer(c *client.Client) {
	pl.pingList.ClientLeft(c)
	pl.keepAlive.ClientLeft(c)
	p := c.GetPlayer()
	pl.pingList.Range(func(c server.PlayerListClient, _ server.PlayerSample) {
		c.(*client.Client).SendPlayerInfoRemove([]*world.Player{p})
	})
}

// keepAliveHandler створює обробник пакетів keepalive
// Ці пакети потрібні щоб перевіряти з'єднання з гравцем
func keepAliveHandler(k *server.KeepAlive) client.PacketHandler {
	return func(p pk.Packet, c *client.Client) error {
		var req pk.Long
		if err := p.Scan(&req); err != nil {
			return err
		}
		k.ClientTick(c)
		return nil
	}
}
