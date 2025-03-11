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

// Йоу, чат! Сьогодні ми тестуємо колізії в майнкрафті!
// BVH (Bounding Volume Hierarchy) - це система для швидкого
// визначення зіткнень між об'єктами. Давайте розберемо як це працює!

package bvh

import "testing"

// TestAABB_WithIn перевіряє чи правильно працює визначення
// знаходження точки всередині AABB (Axis-Aligned Bounding Box)
// AABB - це прямокутник/куб, вирівняний по осях координат
func TestAABB_WithIn(t *testing.T) {
	// Спочатку тестуємо 2D прямокутник
	// Створюємо прямокутник від (-1,-1) до (2,2)
	aabb := AABB[float64, Vec2[float64]]{
		Upper: Vec2[float64]{2, 2},   // Верхній правий кут
		Lower: Vec2[float64]{-1, -1}, // Нижній лівий кут
	}

	// Перевіряємо точку (0,0) - має бути всередині
	if !aabb.WithIn(Vec2[float64]{0, 0}) {
		panic("(0, 0) should included")
	}

	// Перевіряємо точку (-2,-2) - має бути зовні
	if aabb.WithIn(Vec2[float64]{-2, -2}) {
		panic("(-2, -2) shouldn't included")
	}

	// Тепер тестуємо 3D куб
	// Створюємо куб від (-1,-1,-1) до (1,1,1)
	aabb2 := AABB[int, Vec3[int]]{
		Upper: Vec3[int]{1, 1, 1},    // Верхній кут
		Lower: Vec3[int]{-1, -1, -1}, // Нижній кут
	}

	// Перевіряємо точку (0,0,0) - має бути всередині
	if !aabb2.WithIn(Vec3[int]{0, 0, 0}) {
		panic("(0, 0, 0) should included")
	}

	// Перевіряємо точку (-2,-2,0) - має бути зовні
	if aabb2.WithIn(Vec3[int]{-2, -2, 0}) {
		panic("(-2, -2, 0) shouldn't included")
	}

	// Тестуємо сферу (коло в 2D)
	// Створюємо сферу з центром в (0,0) і радіусом 1
	sphere := Sphere[float64, Vec2[float64]]{
		Center: Vec2[float64]{0, 0}, // Центр
		R:      1.0,                 // Радіус
	}

	// Перевіряємо точку (0,0) - має бути всередині
	if !sphere.WithIn(Vec2[float64]{0, 0}) {
		t.Errorf("(0,0) is in")
	}

	// Перевіряємо точку (1,1) - має бути зовні
	// (бо відстань від (0,0) до (1,1) = √2 > 1)
	if sphere.WithIn(Vec2[float64]{1, 1}) {
		t.Errorf("(1,1) isn't in")
	}
}
