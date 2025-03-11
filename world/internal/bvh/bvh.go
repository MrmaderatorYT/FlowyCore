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

// Йоу, чат! Сьогодні ми розберемо як працює BVH дерево!
// BVH (Bounding Volume Hierarchy) - це дерево, де кожен вузол
// містить обмежувальний об'єм (AABB або сферу), який повністю
// містить всі об'єми в його дочірніх вузлах.
// Це дозволяє дуже швидко знаходити об'єкти, що перетинаються!

package bvh

import (
	"container/heap"
	"fmt"

	"golang.org/x/exp/constraints"
)

// Node - вузол BVH дерева
// I - тип для координат (float64)
// B - тип обмежувального об'єму (AABB або Sphere)
// V - тип значення, що зберігається в листі
type Node[I constraints.Float, B interface {
	Union(B) B  // Об'єднання двох об'ємів
	Surface() I // Площа поверхні об'єму
}, V any] struct {
	Box      B                 // Обмежувальний об'єм
	Value    V                 // Значення (тільки в листах)
	parent   *Node[I, B, V]    // Батьківський вузол
	children [2]*Node[I, B, V] // Дочірні вузли (nil для листів)
	isLeaf   bool              // Чи є вузол листом
}

// findAnotherChild повертає інший дочірній вузол (не not)
func (n *Node[I, B, V]) findAnotherChild(not *Node[I, B, V]) *Node[I, B, V] {
	if n.children[0] == not {
		return n.children[1]
	} else if n.children[1] == not {
		return n.children[0]
	}
	panic("unreachable, please make sure the 'not' is the n's child")
}

// findChildPointer повертає вказівник на дочірній вузол
func (n *Node[I, B, V]) findChildPointer(child *Node[I, B, V]) **Node[I, B, V] {
	if n.children[0] == child {
		return &n.children[0]
	} else if n.children[1] == child {
		return &n.children[1]
	}
	panic("unreachable, please make sure the 'not' is the n's child")
}

// each обходить дерево і викликає foreach для кожного вузла, що задовольняє test
func (n *Node[I, B, V]) each(test func(bound B) bool, foreach func(n *Node[I, B, V]) bool) bool {
	if n == nil {
		return true
	}
	if n.isLeaf {
		return !test(n.Box) || foreach(n)
	} else {
		return n.children[0].each(test, foreach) && n.children[1].each(test, foreach)
	}
}

// Tree - BVH дерево
type Tree[I constraints.Float, B interface {
	Union(B) B
	Surface() I
}, V any] struct {
	root *Node[I, B, V] // Корінь дерева
}

// Insert додає новий лист в дерево
// Алгоритм:
// 1. Знаходимо найкращого сусіда для нового листа
// 2. Створюємо новий батьківський вузол
// 3. Оновлюємо обмежувальні об'єми вгору по дереву
func (t *Tree[I, B, V]) Insert(leaf B, value V) (n *Node[I, B, V]) {
	// Створюємо новий лист
	n = &Node[I, B, V]{
		Box:      leaf,
		Value:    value,
		parent:   nil,
		children: [2]*Node[I, B, V]{nil, nil},
		isLeaf:   true,
	}
	// Якщо дерево пусте - новий лист стає коренем
	if t.root == nil {
		t.root = n
		return
	}

	// Етап 1: Шукаємо найкращого сусіда
	sibling := t.root
	bestCost := t.root.Box.Union(leaf).Surface()
	parentTo := &t.root // Вказівник на майбутнього сусіда

	// Черга для пошуку найкращого сусіда
	var queue searchHeap[I, Node[I, B, V]]
	queue.Push(searchItem[I, Node[I, B, V]]{pointer: t.root, parentTo: &t.root})

	leafCost := leaf.Surface()
	for queue.Len() > 0 {
		p := heap.Pop(&queue).(searchItem[I, Node[I, B, V]])
		// Перевіряємо чи поточний вузол кращий за знайдений
		mergeSurface := p.pointer.Box.Union(leaf).Surface()
		deltaCost := mergeSurface - p.pointer.Box.Surface()
		cost := p.inheritedCost + mergeSurface
		if cost <= bestCost {
			bestCost = cost
			sibling = p.pointer
			parentTo = p.parentTo
		}
		// Перевіряємо чи варто дивитись дочірні вузли
		inheritedCost := p.inheritedCost + deltaCost
		if !p.pointer.isLeaf && inheritedCost+leafCost < bestCost {
			heap.Push(&queue, searchItem[I, Node[I, B, V]]{
				pointer:       p.pointer.children[0],
				parentTo:      &p.pointer.children[0],
				inheritedCost: inheritedCost,
			})
			heap.Push(&queue, searchItem[I, Node[I, B, V]]{
				pointer:       p.pointer.children[1],
				parentTo:      &p.pointer.children[1],
				inheritedCost: inheritedCost,
			})
		}
	}

	// Етап 2: Створюємо новий батьківський вузол
	*parentTo = &Node[I, B, V]{
		Box:      sibling.Box.Union(leaf),
		parent:   sibling.parent,
		children: [2]*Node[I, B, V]{sibling, n},
		isLeaf:   false,
	}
	n.parent = *parentTo
	sibling.parent = *parentTo

	// Етап 3: Оновлюємо обмежувальні об'єми вгору по дереву
	for p := *parentTo; p != nil; p = p.parent {
		p.Box = p.children[0].Box.Union(p.children[1].Box)
		t.rotate(p)
	}
	return
}

// Delete видаляє вузол з дерева
func (t *Tree[I, B, V]) Delete(n *Node[I, B, V]) V {
	if n.parent == nil {
		// Якщо видаляємо корінь - дерево стає пустим
		t.root = nil
		return n.Value
	}
	// Знаходимо брата видаляємого вузла
	sibling := n.parent.findAnotherChild(n)
	grand := n.parent.parent
	if grand == nil {
		// Якщо батько - корінь, брат стає новим коренем
		t.root = sibling
		sibling.parent = nil
	} else {
		// Інакше брат займає місце батька
		p := grand.findChildPointer(n.parent)
		*p = sibling
		sibling.parent = grand
		// Оновлюємо обмежувальні об'єми вгору по дереву
		for p := sibling.parent; p.parent != nil; p = p.parent {
			p.Box = p.children[0].Box.Union(p.children[1].Box)
			t.rotate(p)
		}
	}
	return n.Value
}

// rotate оптимізує дерево, намагаючись зменшити площу обмежувальних об'ємів
func (t *Tree[I, B, V]) rotate(n *Node[I, B, V]) {
	if n.isLeaf || n.parent == nil {
		return
	}
	// Пробуємо поміняти місцями брата та дітей вузла
	sibling := n.parent.findAnotherChild(n)
	current := n.Box.Surface()
	if n.children[1].Box.Union(sibling.Box).Surface() < current {
		// Міняємо місцями першу дитину і брата
		t1 := [2]*Node[I, B, V]{n, n.children[0]}
		t2 := [2]*Node[I, B, V]{sibling, n.children[1]}
		n.parent.children, n.children, n.children[0].parent, sibling.parent = t1, t2, n.parent, n
		n.Box = n.children[0].Box.Union(n.children[1].Box)
	} else if n.children[0].Box.Union(sibling.Box).Surface() < current {
		// Міняємо місцями другу дитину і брата
		t1 := [2]*Node[I, B, V]{n, n.children[1]}
		t2 := [2]*Node[I, B, V]{sibling, n.children[0]}
		n.parent.children, n.children, n.children[1].parent, sibling.parent = t1, t2, n.parent, n
		n.Box = n.children[0].Box.Union(n.children[1].Box)
	}
}

// Find шукає всі вузли, що задовольняють умову test
func (t *Tree[I, B, V]) Find(test func(bound B) bool, foreach func(n *Node[I, B, V]) bool) {
	t.root.each(test, foreach)
}

// String повертає текстове представлення дерева
func (t Tree[I, B, V]) String() string {
	return t.root.String()
}

// String повертає текстове представлення вузла
func (n *Node[I, B, V]) String() string {
	if n.isLeaf {
		return fmt.Sprint(n.Value)
	} else {
		return fmt.Sprintf("{%v, %v}", n.children[0], n.children[1])
	}
}

// TouchPoint створює функцію для пошуку об'ємів, що містять точку
func TouchPoint[Vec any, B interface{ WithIn(Vec) bool }](point Vec) func(bound B) bool {
	return func(bound B) bool {
		return bound.WithIn(point)
	}
}

// TouchBound створює функцію для пошуку об'ємів, що перетинаються з іншим
func TouchBound[B interface{ Touch(B) bool }](other B) func(bound B) bool {
	return func(bound B) bool {
		return bound.Touch(other)
	}
}

// searchHeap - допоміжна структура для пошуку найкращого сусіда
type (
	searchHeap[I constraints.Float, V any] []searchItem[I, V]
	searchItem[I constraints.Float, V any] struct {
		pointer       *V  // Вказівник на вузол
		parentTo      **V // Вказівник на батьківський вказівник
		inheritedCost I   // Накопичена вартість шляху
	}
)

// Реалізація інтерфейсу heap.Interface
func (h searchHeap[I, V]) Len() int           { return len(h) }
func (h searchHeap[I, V]) Less(i, j int) bool { return h[i].inheritedCost < h[j].inheritedCost }
func (h searchHeap[I, V]) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *searchHeap[I, V]) Push(x any)        { *h = append(*h, x.(searchItem[I, V])) }
func (h *searchHeap[I, V]) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
