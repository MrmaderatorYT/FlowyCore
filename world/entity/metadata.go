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

// Йоу, чат! Зараз розберемо як працюють метадані сутностей в майнкрафті!
// Метадані - це додаткова інформація про сутність:
// - Поза (стоїть, сидить, лежить)
// - Здоров'я
// - Ефекти
// - Кастомні назви
// і багато іншого!

package entity

import (
	"io"

	pk "github.com/Tnze/go-mc/net/packet"
)

// MetadataSet - це набір метаданих сутності
// Кожна сутність має свій набір полів з різними значеннями
type MetadataSet []MetadataField

// MetadataField - одне поле метаданих
// Складається з індексу (що це за поле) та значення
type MetadataField struct {
	Index         byte // Номер поля
	MetadataValue      // Значення поля
}

// WriteTo записує всі метадані в пакет
// В кінці записує 0xFF як маркер кінця даних
func (m MetadataSet) WriteTo(w io.Writer) (n int64, err error) {
	var tmpN int64
	// Записуємо кожне поле
	for _, v := range m {
		// Записуємо індекс поля
		tmpN, err = pk.UnsignedByte(v.Index).WriteTo(w)
		n += tmpN
		if err != nil {
			return
		}
		// Записуємо значення
		tmpN, err = v.WriteTo(w)
		if err != nil {
			return
		}
	}
	// Записуємо маркер кінця (0xFF)
	tmpN, err = pk.UnsignedByte(0xFF).WriteTo(w)
	return n + tmpN, err
}

// WriteTo записує одне поле метаданих
// Спочатку тип даних, потім саме значення
func (m *MetadataField) WriteTo(w io.Writer) (n int64, err error) {
	// Записуємо ID типу даних
	n1, err := pk.VarInt(m.MetadataValue.TypeID()).WriteTo(w)
	if err != nil {
		return n1, err
	}
	// Записуємо значення
	n2, err := m.MetadataValue.WriteTo(w)
	return n1 + n2, err
}

// MetadataValue - інтерфейс для різних типів значень
// Кожен тип повинен вміти:
// - Повертати свій ID
// - Записувати себе в пакет
type MetadataValue interface {
	TypeID() int32 // Повертає ID типу даних
	pk.Field       // Інтерфейс для запису в пакет
}

// Різні типи метаданих:
type (
	Byte struct{ pk.Byte } // Для маленьких чисел (0-255)
	// VarInt для великих чисел
	// Float для дробових чисел
	// String для тексту
	// Chat для повідомлень в чаті
	// OptionalChat для необов'язкових повідомлень
	// Slot для предметів
	// Boolean для true/false
	// Rotation для кутів повороту
	// Position для координат

	// Pose - поза сутності
	Pose int32
)

// TypeID повертає ID типу даних
func (b *Byte) TypeID() int32 { return 0 }  // Байт = тип 0
func (p *Pose) TypeID() int32 { return 18 } // Поза = тип 18

// Всі можливі пози сутності
const (
	Standing    Pose = iota // Стоїть
	FallFlying              // Летить з елітрами
	Sleeping                // Спить
	Swimming                // Плаває
	SpinAttack              // Атакує з розворотом
	Crouching               // Присів
	LongJumping             // Довгий стрибок
	Dying                   // Помирає
	Croaking                // Квакає (жаба)
	UsingTongue             // Використовує язик (жаба)
	Roaring                 // Реве (варден)
	Sniffing                // Нюхає (варден)
	Emerging                // Виходить з землі (варден)
	Digging                 // Копає (варден)
)

// WriteTo записує позу в пакет
func (p Pose) WriteTo(w io.Writer) (n int64, err error) {
	return pk.VarInt(p).WriteTo(w)
}

// ReadFrom читає позу з пакету
func (p *Pose) ReadFrom(r io.Reader) (n int64, err error) {
	return (*pk.VarInt)(p).ReadFrom(r)
}
