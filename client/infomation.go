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

// Йоу, чат! Зараз розберемо як клієнт передає свої налаштування серверу!
// Коли гравець заходить на сервер, його клієнт відправляє інформацію про себе

package client

import (
	// Імпортуємо наш пакет world, де зберігається структура ClientInfo
	"FlowyCore/world"
	// pk - пакети майнкрафта
	pk "github.com/Tnze/go-mc/net/packet"
)

// clientInformation обробляє пакет з налаштуваннями клієнта
// Цей пакет відправляється при підключенні і коли гравець змінює налаштування
// Наприклад: змінив мову, дальність прогрузки, чат і т.д.
func clientInformation(p pk.Packet, client *Client) error {
	// ClientInfo містить:
	// - Мову клієнта (en_US, uk_UA і т.д.)
	// - Дальність прогрузки в чанках
	// - Налаштування чату (показувати все, тільки безпечні, тільки системні)
	// - Показувати скіни гравців чи ні
	// - Показувати cape чи ні
	// - Основна рука (ліва/права)
	// - Фільтрація тексту (вкл/викл)
	// - Дозволити серверу показувати іконки гравців
	var info world.ClientInfo

	// Читаємо всі налаштування з пакету
	if err := p.Scan(&info); err != nil {
		return err
	}

	// Блокуємо доступ до Inputs щоб інші горутини не змінили дані
	client.Inputs.Lock()
	// Зберігаємо нові налаштування
	client.Inputs.ClientInfo = info
	// Розблоковуємо доступ
	client.Inputs.Unlock()
	return nil
}
