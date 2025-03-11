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

// Йоу, чат! Сьогодні ми тестуємо BVH дерево!
// BVH (Bounding Volume Hierarchy) - це структура даних,
// яка дозволяє швидко знаходити об'єкти, що перетинаються.
// Це як октодерево, але для довільних обмежувальних об'ємів!

package bvh

import (
	"math/rand"
	"testing"
)

// TestTree2_Insert тестує додавання AABB в дерево
// Ми створюємо кілька прямокутників і додаємо їх в дерево
func TestTree2_Insert(t *testing.T) {
	// Створюємо набір тестових AABB
	// Кожен AABB - це прямокутник розміром 1x1
	aabbs := []AABB[float64, Vec2[float64]]{
		{Upper: Vec2[float64]{1, 1}, Lower: Vec2[float64]{0, 0}},     // (0,0) -> (1,1)
		{Upper: Vec2[float64]{2, 1}, Lower: Vec2[float64]{1, 0}},     // (1,0) -> (2,1)
		{Upper: Vec2[float64]{11, 1}, Lower: Vec2[float64]{10, 0}},   // (10,0) -> (11,1)
		{Upper: Vec2[float64]{12, 1}, Lower: Vec2[float64]{11, 0}},   // (11,0) -> (12,1)
		{Upper: Vec2[float64]{101, 1}, Lower: Vec2[float64]{100, 0}}, // (100,0) -> (101,1)
		{Upper: Vec2[float64]{102, 1}, Lower: Vec2[float64]{101, 0}}, // (101,0) -> (102,1)
		{Upper: Vec2[float64]{111, 1}, Lower: Vec2[float64]{110, 0}}, // (110,0) -> (111,1)
		{Upper: Vec2[float64]{112, 1}, Lower: Vec2[float64]{111, 0}}, // (111,0) -> (112,1)
		{Upper: Vec2[float64]{1, 1}, Lower: Vec2[float64]{-1, -1}},   // (-1,-1) -> (1,1)
	}

	// Створюємо дерево і додаємо всі AABB
	var bvh Tree[float64, AABB[float64, Vec2[float64]], int]
	for i, aabb := range aabbs {
		bvh.Insert(aabb, i)
		// Виводимо дерево після кожного додавання
		t.Log(bvh)
	}

	// Шукаємо всі AABB, що містять точку (0.5, 0.5)
	bvh.Find(TouchPoint[Vec2[float64], AABB[float64, Vec2[float64]]](Vec2[float64]{0.5, 0.5}), func(n *Node[float64, AABB[float64, Vec2[float64]], int]) bool {
		t.Logf("find! %v", n.Value)
		return true
	})
}

// TestTree2_Find_vec тестує пошук об'єктів в дереві
// Ми створюємо кілька AABB і шукаємо ті, що перетинаються з точками та іншими AABB
func TestTree2_Find_vec(t *testing.T) {
	// Створюємо типи-аліаси для зручності
	type Vec2d = Vec2[float64]
	type AABBVec2d = AABB[float64, Vec2d]
	type TreeAABBVec2di = Tree[float64, AABBVec2d, int]

	// Створюємо тестові AABB - чотири прямокутники, що перекриваються
	aabbs := []AABBVec2d{
		{Upper: Vec2d{2, 2}, Lower: Vec2d{-1, -1}}, // Великий прямокутник
		{Upper: Vec2d{2, 1}, Lower: Vec2d{-1, -2}}, // Нижній прямокутник
		{Upper: Vec2d{1, 1}, Lower: Vec2d{-2, -2}}, // Лівий прямокутник
		{Upper: Vec2d{1, 2}, Lower: Vec2d{-2, -1}}, // Верхній прямокутник
	}

	// Створюємо дерево і додаємо всі AABB
	var bvh TreeAABBVec2di
	for i, aabb := range aabbs {
		bvh.Insert(aabb, i)
		t.Log(bvh)
	}

	// Допоміжна функція для пошуку
	find := func(test func(bound AABBVec2d) bool) []int {
		var result []int
		bvh.Find(test, func(n *Node[float64, AABBVec2d, int]) bool {
			result = append(result, n.Value)
			return true
		})
		return result
	}

	// Тестуємо пошук точок
	t.Log(find(TouchPoint[Vec2d, AABBVec2d](Vec2d{0, 0})))     // Центр
	t.Log(find(TouchPoint[Vec2d, AABBVec2d](Vec2d{1.5, 0})))   // Правий край
	t.Log(find(TouchPoint[Vec2d, AABBVec2d](Vec2d{1.5, 1.5}))) // Правий верхній кут
	t.Log(find(TouchPoint[Vec2d, AABBVec2d](Vec2d{-1.5, 0})))  // Лівий край

	// Тестуємо пошук перетинів з іншими AABB
	t.Log(find(TouchBound[AABBVec2d](AABBVec2d{Upper: Vec2d{1, 1}, Lower: Vec2d{-1, -1}})))          // Центральний квадрат
	t.Log(find(TouchBound[AABBVec2d](AABBVec2d{Upper: Vec2d{1, 1}, Lower: Vec2d{1.5, 1.5}})))        // Зовнішній квадрат
	t.Log(find(TouchBound[AABBVec2d](AABBVec2d{Upper: Vec2d{-1.5, 0.5}, Lower: Vec2d{-2.5, -0.5}}))) // Лівий квадрат
}

// BenchmarkTree_Insert тестує швидкість додавання в дерево
func BenchmarkTree_Insert(b *testing.B) {
	type Vec2d = Vec2[float64]
	type AABBVec2d = AABB[float64, Vec2d]
	type TreeAABBVec2da = Tree[float64, AABBVec2d, any]

	const size = 25 // Розмір AABB

	// Генеруємо випадкові AABB для тесту
	aabbs := make([]AABBVec2d, b.N)
	poses := make([]Vec2d, b.N)
	for i := range aabbs {
		// Випадкова позиція в діапазоні 0-10000
		poses[i] = Vec2d{rand.Float64() * 1e4, rand.Float64() * 1e4}
		// AABB розміром 50x50 навколо позиції
		aabbs[i] = AABBVec2d{
			Upper: Vec2d{poses[i][0] + size, poses[i][0] + size},
			Lower: Vec2d{poses[i][0] - size, poses[i][0] - size},
		}
	}
	b.ResetTimer() // Починаємо заміри

	// Додаємо всі AABB в дерево
	var bvh TreeAABBVec2da
	for _, v := range aabbs {
		bvh.Insert(v, nil)
	}
}

// BenchmarkTree2_Find_random тестує швидкість пошуку в дереві
func BenchmarkTree2_Find_random(b *testing.B) {
	type Vec2d = Vec2[float64]
	type AABBVec2d = AABB[float64, Vec2d]
	type TreeAABBVec2da = Tree[float64, AABBVec2d, any]

	const size = 25 // Розмір AABB

	// Генеруємо випадкові AABB та позиції для тесту
	aabbs := make([]AABBVec2d, b.N)
	poses := make([]Vec2d, b.N)
	for i := range aabbs {
		poses[i] = Vec2d{rand.Float64() * 1e4, rand.Float64() * 1e4}
		aabbs[i] = AABBVec2d{
			Upper: Vec2d{poses[i][0] + size, poses[i][0] + size},
			Lower: Vec2d{poses[i][0] - size, poses[i][0] - size},
		}
	}

	// Створюємо дерево і додаємо всі AABB
	var bvh TreeAABBVec2da
	for _, v := range aabbs {
		bvh.Insert(v, nil)
	}
	b.ResetTimer() // Починаємо заміри

	// Шукаємо AABB для кожної позиції
	for _, v := range poses {
		bvh.Find(TouchPoint[Vec2d, AABBVec2d](v), func(n *Node[float64, AABBVec2d, any]) bool { return true })
	}
}

// BenchmarkTree2_Delete_random тестує швидкість видалення з дерева
func BenchmarkTree2_Delete_random(b *testing.B) {
	const size = 25 // Розмір AABB

	// Генеруємо випадкові AABB для тесту
	aabbs := make([]AABB[float64, Vec2[float64]], b.N)
	poses := make([]Vec2[float64], b.N)
	nodes := make([]*Node[float64, AABB[float64, Vec2[float64]], any], b.N)
	for i := range aabbs {
		poses[i] = Vec2[float64]{rand.Float64() * 1e4, rand.Float64() * 1e4}
		aabbs[i] = AABB[float64, Vec2[float64]]{
			Upper: Vec2[float64]{poses[i][0] + size, poses[i][0] + size},
			Lower: Vec2[float64]{poses[i][0] - size, poses[i][0] - size},
		}
	}
	b.ResetTimer() // Починаємо заміри

	// Створюємо дерево і додаємо всі AABB
	var bvh Tree[float64, AABB[float64, Vec2[float64]], any]
	for i, v := range aabbs {
		nodes[i] = bvh.Insert(v, nil)
	}

	b.StopTimer() // Зупиняємо заміри
	// Перемішуємо вузли для випадкового порядку видалення
	rand.Shuffle(b.N, func(i, j int) {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	})
	b.StartTimer() // Відновлюємо заміри

	// Видаляємо всі вузли
	for _, v := range nodes {
		bvh.Delete(v)
	}
}
