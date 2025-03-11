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

// Йоу, чат! Сьогодні ми розберемо як працює система чату в нашому сервері!
// В Minecraft 1.19+ з'явилася нова система безпеки чату,
// яка використовує криптографічні підписи для перевірки
// справжності повідомлень. Це захищає від спуфінгу та спаму.

package world

import "time"

// SetLastChatTimestamp оновлює час останнього повідомлення гравця
// та повертає true, якщо новий час пізніший за попередній.
// Це потрібно для:
// 1. Перевірки порядку повідомлень
// 2. Захисту від спаму
// 3. Синхронізації чату між клієнтами
func (p *Player) SetLastChatTimestamp(t time.Time) bool {
	if p.lastChatTimestamp.Before(t) {
		p.lastChatTimestamp = t
		return true
	}
	return false
}

// GetPrevChatSignature повертає підпис попереднього повідомлення
// Підпис використовується для створення ланцюжка повідомлень,
// де кожне нове повідомлення містить підпис попереднього
func (p *Player) GetPrevChatSignature() []byte {
	return p.lastChatSignature
}

// SetPrevChatSignature зберігає підпис останнього повідомлення
// Цей підпис буде використано при відправці наступного повідомлення
// для підтвердження послідовності повідомлень
func (p *Player) SetPrevChatSignature(sig []byte) {
	p.lastChatSignature = sig
}
