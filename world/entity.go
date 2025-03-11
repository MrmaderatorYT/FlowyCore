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

// Йоу, чат! Сьогодні ми розберемо як працюють сутності в нашому сервері!
// Сутність (Entity) - це будь-який об'єкт у грі, який може рухатися:
// гравці, мобі, предмети на землі, стріли тощо.
// Кожна сутність має унікальний ID та координати у просторі.

package world

import (
	"math"
	"sync/atomic"
)

// entityCounter - атомарний лічильник для генерації унікальних ID сутностей
// Використовуємо atomic для безпечної роботи в багатопотоковому середовищі
var entityCounter atomic.Int32

// NewEntityID генерує новий унікальний ID для сутності
// Використовує атомарний інкремент для уникнення дублікатів
func NewEntityID() int32 {
	return entityCounter.Add(1)
}

// Entity - базова структура для всіх сутностей у грі
// Містить основні поля, які є у кожної сутності:
// - EntityID: унікальний ідентифікатор
// - Position: позиція у світі (x, y, z)
// - Rotation: кут повороту (yaw, pitch)
// - OnGround: чи стоїть на землі
// - pos0, rot0: попередні значення для інтерполяції руху
type Entity struct {
	EntityID int32
	Position          // x, y, z координати
	Rotation          // кути повороту
	OnGround          // чи на землі
	pos0     Position // попередня позиція
	rot0     Rotation // попередній поворот
}

// Position - позиція у 3D просторі
// [0] - x (схід/захід)
// [1] - y (верх/низ)
// [2] - z (північ/південь)
type Position [3]float64

// Rotation - кути повороту
// [0] - yaw (поворот навколо вертикальної осі)
// [1] - pitch (нахил голови)
type Rotation [2]float32

// OnGround - прапорець, що вказує чи сутність на землі
// Використовується для фізики та анімацій
type OnGround bool

// getPoint повертає 2D координати сутності (x, z)
// Використовується для колізій та пошуку в просторі
func (e *Entity) getPoint() [2]float64 {
	return [2]float64{e.Position[0], e.Position[2]}
}

// IsValid перевіряє чи координати позиції є допустимими числами
// Повертає true якщо всі координати:
// 1. Не є NaN (Not a Number - невизначене значення)
// 2. Не є Inf (нескінченність)
// Це потрібно щоб:
// - Запобігти багам з телепортацією
// - Захиститись від експлойтів з некоректними координатами
// - Уникнути краху сервера через математичні помилки
func (p *Position) IsValid() bool {
	return !math.IsNaN((*p)[0]) && !math.IsNaN((*p)[1]) && !math.IsNaN((*p)[2]) && // перевіряємо на NaN
		!math.IsInf((*p)[0], 0) && !math.IsInf((*p)[1], 0) && !math.IsInf((*p)[2], 0) // перевіряємо на Inf
}
