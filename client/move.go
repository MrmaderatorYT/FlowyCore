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

// Йоу, чат! Зараз розберемо як працює рух в майнкрафті!
// Це файл обробляє всі пакети руху які приходять від клієнта

package client

import (
	"bytes"

	// pk - це пакети майнкрафта
	pk "github.com/Tnze/go-mc/net/packet"
)

// clientAcceptTeleportation обробляє підтвердження телепортації від клієнта
// Коли сервер телепортує гравця, клієнт має підтвердити що телепортація відбулась
func clientAcceptTeleportation(p pk.Packet, c *Client) error {
	// TeleportID - унікальний номер телепортації
	var TeleportID pk.VarInt
	// Читаємо ID з пакету
	_, err := TeleportID.ReadFrom(bytes.NewReader(p.Data))
	if err != nil {
		return err
	}
	// Блокуємо доступ до Inputs щоб інші горутини не змінили дані
	c.Inputs.Lock()
	// Зберігаємо ID телепортації
	c.Inputs.TeleportID = int32(TeleportID)
	// Розблоковуємо доступ
	c.Inputs.Unlock()
	return nil
}

// clientMovePlayerPos обробляє рух гравця (тільки позиція)
// Викликається коли гравець рухається без повороту
func clientMovePlayerPos(p pk.Packet, c *Client) error {
	// Координати гравця (X, Y, Z)
	var X, FeetY, Z pk.Double
	// OnGround - чи стоїть гравець на землі
	var OnGround pk.Boolean
	// Читаємо дані з пакету
	if err := p.Scan(&X, &FeetY, &Z, &OnGround); err != nil {
		return err
	}
	// Блокуємо доступ до Inputs
	c.Inputs.Lock()
	// Оновлюємо позицію гравця
	c.Inputs.Position = [3]float64{float64(X), float64(FeetY), float64(Z)}
	c.Inputs.Unlock()
	return nil
}

// clientMovePlayerPosRot обробляє рух гравця з поворотом
// Викликається коли гравець рухається і одночасно крутиться
func clientMovePlayerPosRot(p pk.Packet, c *Client) error {
	// Координати (X, Y, Z)
	var X, FeetY, Z pk.Double
	// Кути повороту:
	// Yaw - поворот вліво-вправо (0-360 градусів)
	// Pitch - нахил голови вверх-вниз (-90 до +90 градусів)
	var Yaw, Pitch pk.Float
	var OnGround pk.Boolean
	// Читаємо всі дані з пакету
	if err := p.Scan(&X, &FeetY, &Z, &Yaw, &Pitch, &OnGround); err != nil {
		return err
	}
	c.Inputs.Lock()
	// Оновлюємо і позицію і кути повороту
	c.Inputs.Position = [3]float64{float64(X), float64(FeetY), float64(Z)}
	c.Inputs.Rotation = [2]float32{float32(Yaw), float32(Pitch)}
	c.Inputs.Unlock()
	return nil
}

// clientMovePlayerRot обробляє поворот гравця
// Викликається коли гравець тільки крутиться на місці
func clientMovePlayerRot(p pk.Packet, c *Client) error {
	// Тільки кути повороту
	var Yaw, Pitch pk.Float
	var OnGround pk.Boolean
	if err := p.Scan(&Yaw, &Pitch, &OnGround); err != nil {
		return err
	}
	c.Inputs.Lock()
	// Оновлюємо тільки кути
	c.Inputs.Rotation = [2]float32{float32(Yaw), float32(Pitch)}
	c.Inputs.Unlock()
	return nil
}

// clientMovePlayerStatusOnly обробляє зміну стану "на землі"
// Викликається коли змінюється тільки OnGround
func clientMovePlayerStatusOnly(p pk.Packet, c *Client) error {
	// OnGround передається як 1 байт
	var OnGround pk.UnsignedByte
	if err := p.Scan(&OnGround); err != nil {
		return err
	}
	c.Inputs.Lock()
	// Конвертуємо в bool: 0 = false, не 0 = true
	c.Inputs.OnGround = OnGround != 0
	c.Inputs.Unlock()
	return nil
}

// clientMoveVehicle обробляє рух транспорту
// Поки що не реалізовано, просто заглушка
func clientMoveVehicle(_ pk.Packet, _ *Client) error {
	return nil
}
