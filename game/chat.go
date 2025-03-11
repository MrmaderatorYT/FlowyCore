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

// Йоу, чат! Зараз розберемо як працює чат в майнкрафті!
// В 1.19+ з'явилася система безпечного чату з підписами повідомлень

package game

import (
	"time"

	// zap - крутий логер для Go
	"go.uber.org/zap"

	// Наші та зовнішні пакети
	"FlowyCore/client"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/chat/sign"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/registry"
	"github.com/Tnze/go-mc/server"
)

// Повідомлення в чаті живуть 5 хвилин
// Після цього їх не можна відправити (захист від спаму)
const MsgExpiresTime = time.Minute * 5

// globalChat керує всім чатом на сервері
type globalChat struct {
	// Логер для запису подій чату
	log *zap.Logger
	// Список всіх гравців на сервері
	players *playerList
	// Типи повідомлень (чат, система, шепіт і т.д.)
	chatTypeCodec *registry.Registry[registry.ChatType]
}

// broadcastSystemChat відправляє системне повідомлення всім гравцям
// Наприклад: "Гравець приєднався" або "Сервер перезавантажується"
func (g *globalChat) broadcastSystemChat(msg chat.Message, overlay bool) {
	// Логуємо повідомлення
	g.log.Info(msg.String(), zap.Bool("overlay", overlay))
	// Відправляємо кожному гравцю
	g.players.pingList.Range(func(c server.PlayerListClient, _ server.PlayerSample) {
		c.(*client.Client).SendSystemChat(msg, overlay)
	})
}

// Handle обробляє повідомлення від гравців
func (g *globalChat) Handle(p pk.Packet, c *client.Client) error {
	// Дані повідомлення:
	var (
		// Сам текст
		message pk.String
		// Час відправки (для перевірки актуальності)
		timestampLong pk.Long
		// Сіль для підпису
		salt pk.Long
		// Цифровий підпис повідомлення
		signature pk.Option[sign.Signature, *sign.Signature]
		// Останні бачені повідомлення
		lastSeen sign.HistoryUpdate
	)

	// Читаємо всі дані з пакету
	err := p.Scan(
		&message,
		&timestampLong,
		&salt,
		&signature,
		&lastSeen,
	)
	if err != nil {
		return err
	}

	// Отримуємо інфу про гравця
	player := c.GetPlayer()
	// Конвертуємо час з мілісекунд
	timestamp := time.UnixMilli(int64(timestampLong))
	// Створюємо логер з даними про відправника
	logger := g.log.With(
		zap.String("sender", player.Name),
		zap.Time("timestamp", timestamp),
	)

	// Перевіряємо заборонені символи
	// § - символ форматування кольору
	// Символи менше пробілу - керуючі символи
	// 0x7F - символ видалення
	if existInvalidCharacter(string(message)) {
		c.SendDisconnect(chat.TranslateMsg("multiplayer.disconnect.illegal_characters"))
		return nil
	}

	// Перевіряємо що повідомлення прийшли в правильному порядку
	if !player.SetLastChatTimestamp(timestamp) {
		c.SendDisconnect(chat.TranslateMsg("multiplayer.disconnect.out_of_order_chat"))
		return nil
	}

	// TODO: Перевірка чи гравець не вимкнув чат
	if false {
		c.SendSystemChat(chat.TranslateMsg("chat.disabled.options").SetColor(chat.Red), false)
		return nil
	}

	// TODO: Перевірка підпису повідомлення
	//var playerMsg sign.PlayerMessage
	////if player.PubKey != nil {
	////}

	// Перевіряємо що повідомлення не застаріло
	if time.Since(timestamp) > MsgExpiresTime {
		logger.Warn("Player send expired message", zap.String("msg", string(message)))
		return nil
	}

	// Створюємо тип повідомлення "чат"
	chatTypeID, decorator := g.chatTypeCodec.Find("minecraft:chat")
	chatType := chat.Type{
		ID: chatTypeID,
		// Ім'я відправника
		SenderName: chat.Text(player.Name),
		// Отримувач (nil = всім)
		TargetName: nil,
	}

	// Форматуємо повідомлення за шаблоном
	decorated := chatType.Decorate(chat.Text(string(message)), &decorator.Chat)
	// Логуємо готове повідомлення
	logger.Info(decorated.String())

	// Відправляємо повідомлення всім гравцям
	g.players.pingList.Range(func(c server.PlayerListClient, _ server.PlayerSample) {
		c.(*client.Client).SendPlayerChat(
			// UUID відправника
			player.UUID,
			// Індекс повідомлення
			0,
			// Цифровий підпис
			signature,
			// Тіло повідомлення:
			&sign.PackedMessageBody{
				// Текст
				PlainMsg: string(message),
				// Час відправки
				Timestamp: timestamp,
				// Сіль для підпису
				Salt: int64(salt),
				// Історія повідомлень
				LastSeen: []sign.PackedSignature{},
			},
			// Неформатований текст
			nil,
			// Фільтр чату
			&sign.FilterMask{Type: 0},
			// Тип повідомлення
			&chatType,
		)
	})
	return nil
}

// existInvalidCharacter перевіряє заборонені символи в повідомленні
func existInvalidCharacter(msg string) bool {
	for _, c := range msg {
		// § - символ форматування
		// < пробілу - керуючі символи
		// 0x7F - символ видалення
		if c == '§' || c < ' ' || c == '\x7F' {
			return true
		}
	}
	return false
}
