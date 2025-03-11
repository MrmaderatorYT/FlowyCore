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

// Йоу, чат! Сьогодні ми розберемо як працюють вектори в нашому BVH дереві!
// Тут у нас є два типи векторів - Vec2 для 2D та Vec3 для 3D простору.
// Вони використовуються для зберігання координат та розмірів наших AABB.

package bvh

import (
	"math"

	"golang.org/x/exp/constraints"
)

// Vec2 - двовимірний вектор
// I може бути будь-яким числовим типом (int, float64 тощо)
type Vec2[I constraints.Signed | constraints.Float] [2]I

// Add додає інший вектор до поточного
func (v Vec2[I]) Add(other Vec2[I]) Vec2[I] { return Vec2[I]{v[0] + other[0], v[1] + other[1]} }

// Sub віднімає інший вектор від поточного
func (v Vec2[I]) Sub(other Vec2[I]) Vec2[I] { return Vec2[I]{v[0] - other[0], v[1] - other[1]} }

// Mul множить вектор на скаляр
func (v Vec2[I]) Mul(i I) Vec2[I] { return Vec2[I]{v[0] * i, v[1] * i} }

// Max повертає вектор з максимальними координатами
func (v Vec2[I]) Max(other Vec2[I]) Vec2[I] { return Vec2[I]{max(v[0], other[0]), max(v[1], other[1])} }

// Min повертає вектор з мінімальними координатами
func (v Vec2[I]) Min(other Vec2[I]) Vec2[I] { return Vec2[I]{min(v[0], other[0]), min(v[1], other[1])} }

// Less перевіряє чи всі координати менші за other
func (v Vec2[I]) Less(other Vec2[I]) bool { return v[0] < other[0] && v[1] < other[1] }

// More перевіряє чи всі координати більші за other
func (v Vec2[I]) More(other Vec2[I]) bool { return v[0] > other[0] && v[1] > other[1] }

// Norm повертає довжину вектора
func (v Vec2[I]) Norm() float64 { return sqrt(v[0]*v[0] + v[1]*v[1]) }

// Sum повертає суму всіх координат
func (v Vec2[I]) Sum() I { return v[0] + v[1] }

// Vec3 - тривимірний вектор
// I може бути будь-яким числовим типом (int, float64 тощо)
type Vec3[I constraints.Signed | constraints.Float] [3]I

// Add додає інший вектор до поточного
func (v Vec3[I]) Add(other Vec3[I]) Vec3[I] {
	return Vec3[I]{v[0] + other[0], v[1] + other[1], v[2] + other[2]}
}

// Sub віднімає інший вектор від поточного
func (v Vec3[I]) Sub(other Vec3[I]) Vec3[I] {
	return Vec3[I]{v[0] - other[0], v[1] - other[1], v[2] - other[2]}
}

// Mul множить вектор на скаляр
func (v Vec3[I]) Mul(i I) Vec3[I] { return Vec3[I]{v[0] * i, v[1] * i, v[2] * i} }

// Max повертає вектор з максимальними координатами
func (v Vec3[I]) Max(other Vec3[I]) Vec3[I] {
	return Vec3[I]{max(v[0], other[0]), max(v[1], other[1]), max(v[2], other[2])}
}

// Min повертає вектор з мінімальними координатами
func (v Vec3[I]) Min(other Vec3[I]) Vec3[I] {
	return Vec3[I]{min(v[0], other[0]), min(v[1], other[1]), min(v[2], other[2])}
}

// Less перевіряє чи всі координати менші за other
func (v Vec3[I]) Less(other Vec3[I]) bool { return v[0] < other[0] && v[1] < other[1] }

// More перевіряє чи всі координати більші за other
func (v Vec3[I]) More(other Vec3[I]) bool { return v[0] > other[0] && v[1] > other[1] }

// Norm повертає довжину вектора
func (v Vec3[I]) Norm() float64 { return sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]) }

// Sum повертає суму всіх координат
func (v Vec3[I]) Sum() I { return v[0] + v[1] }

// Допоміжні функції для роботи з числами

// max повертає більше з двох чисел
func max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// min повертає менше з двох чисел
func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// sqrt обчислює квадратний корінь з числа
// конвертує вхідне число у float64 для обчислення
func sqrt[T constraints.Signed | constraints.Float](v T) float64 {
	return math.Sqrt(float64(v))
}
