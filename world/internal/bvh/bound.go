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

// Йоу, чат! Сьогодні ми розберемо як працюють колізії в майнкрафті!
// Цей файл містить реалізацію двох типів обмежувальних об'ємів:
// - AABB (Axis-Aligned Bounding Box) - прямокутники/куби
// - Sphere - сфери/кола
// Вони використовуються для швидкої перевірки зіткнень!

package bvh

import (
	"math"

	"golang.org/x/exp/constraints"
)

// AABB - прямокутник або куб, вирівняний по осях координат
// I - тип для координат (int або float64)
// V - тип для векторів (Vec2 або Vec3)
type AABB[I constraints.Signed | constraints.Float, V interface {
	Add(V) V     // Додавання векторів
	Sub(V) V     // Віднімання векторів
	Max(V) V     // Максимум по компонентах
	Min(V) V     // Мінімум по компонентах
	Less(V) bool // Порівняння < по всіх компонентах
	More(V) bool // Порівняння > по всіх компонентах
	Sum() I      // Сума всіх компонент
}] struct {
	Upper, Lower V // Верхня та нижня точки прямокутника/куба
}

// WithIn перевіряє чи точка знаходиться всередині AABB
func (aabb AABB[I, V]) WithIn(point V) bool {
	return aabb.Lower.Less(point) && aabb.Upper.More(point)
}

// Touch перевіряє чи перетинаються два AABB
func (aabb AABB[I, V]) Touch(other AABB[I, V]) bool {
	return aabb.Lower.Less(other.Upper) && other.Lower.Less(aabb.Upper) &&
		aabb.Upper.More(other.Lower) && other.Upper.More(aabb.Lower)
}

// Union повертає найменший AABB, що містить обидва вхідні AABB
func (aabb AABB[I, V]) Union(other AABB[I, V]) AABB[I, V] {
	return AABB[I, V]{
		Upper: aabb.Upper.Max(other.Upper), // Беремо максимум верхніх точок
		Lower: aabb.Lower.Min(other.Lower), // Беремо мінімум нижніх точок
	}
}

// Surface повертає площу поверхні AABB
func (aabb AABB[I, V]) Surface() I {
	return aabb.Upper.Sub(aabb.Lower).Sum() * 2
}

// Sphere - сфера або коло
// I - тип для координат (float64)
// V - тип для векторів (Vec2 або Vec3)
type Sphere[I constraints.Float, V interface {
	Add(V) V     // Додавання векторів
	Sub(V) V     // Віднімання векторів
	Mul(I) V     // Множення на число
	Max(V) V     // Максимум по компонентах
	Min(V) V     // Мінімум по компонентах
	Less(V) bool // Порівняння < по всіх компонентах
	More(V) bool // Порівняння > по всіх компонентах
	Norm() I     // Довжина вектора
	Sum() I      // Сума всіх компонент
}] struct {
	Center V // Центр сфери
	R      I // Радіус сфери
}

// WithIn перевіряє чи точка знаходиться всередині сфери
// Для цього рахуємо відстань від центра до точки
func (s Sphere[I, V]) WithIn(point V) bool {
	return s.Center.Sub(point).Norm() < s.R
}

// Touch перевіряє чи перетинаються дві сфери
// Сфери перетинаються якщо відстань між центрами менша за суму радіусів
func (s Sphere[I, V]) Touch(other Sphere[I, V]) bool {
	return s.Center.Sub(other.Center).Norm() < s.R+other.R
}

// Union повертає найменшу сферу, що містить обидві вхідні сфери
func (s Sphere[I, V]) Union(other Sphere[I, V]) Sphere[I, V] {
	// Рахуємо відстань між центрами
	d := other.Center.Sub(s.Center).Norm()
	// Коефіцієнт для інтерполяції центрів
	r1r2d := (s.R - other.R) / d
	return Sphere[I, V]{
		// Новий центр - зважена сума старих центрів
		Center: s.Center.Mul(1 + r1r2d).Add(other.Center.Mul(1 - r1r2d)),
		// Новий радіус включає обидві сфери
		R: d + s.R + other.R,
	}
}

// Surface повертає площу поверхні сфери
func (s Sphere[I, V]) Surface() I {
	return 2 * math.Pi * s.R
}
