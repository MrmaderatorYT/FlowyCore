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

// Йоу, чат! Сьогодні ми розберемо як працює система завантаження чанків!
// Це дуже важлива частина серверу, яка відповідає за те, які чанки
// потрібно завантажити для гравця, а які можна вивантажити.
// Система використовує круговий алгоритм завантаження, щоб гравець
// завжди бачив світ навколо себе у радіусі видимості.

package world

import (
	"math"
	"sort"

	"golang.org/x/time/rate"
)

// loader відповідає за завантаження чанків навколо певної точки
// Кожен loader має позицію 'pos' та радіус 'r', в межах якого
// будуть завантажуватися чанки
type loader struct {
	loaderSource                       // інтерфейс для отримання позиції та радіусу
	loaded       map[[2]int32]struct{} // мапа завантажених чанків
	loadQueue    [][2]int32            // черга чанків для завантаження
	unloadQueue  [][2]int32            // черга чанків для вивантаження
	limiter      *rate.Limiter         // обмежувач швидкості завантаження
}

// loaderSource - інтерфейс для отримання інформації про точку завантаження
// Зазвичай це гравець або камера спостереження
type loaderSource interface {
	chunkPosition() [2]int32 // повертає позицію в координатах чанків
	chunkRadius() int32      // повертає радіус завантаження в чанках
}

// newLoader створює новий завантажувач чанків
// source - джерело позиції (гравець/камера)
// limiter - обмежувач швидкості завантаження
func newLoader(source loaderSource, limiter *rate.Limiter) (l *loader) {
	l = &loader{
		loaderSource: source,
		loaded:       make(map[[2]int32]struct{}),
		limiter:      limiter,
	}
	l.calcLoadingQueue() // одразу розраховуємо перші чанки для завантаження
	return
}

// calcLoadingQueue розраховує які чанки потрібно завантажити
// Результат зберігається в l.loadQueue, попередня черга очищується
// Чанки додаються по спіралі від центру до краю радіусу
func (l *loader) calcLoadingQueue() {
	l.loadQueue = l.loadQueue[:0]
	for _, v := range loadList[:radiusIdx[l.chunkRadius()]] {
		pos := l.chunkPosition()
		pos[0], pos[1] = pos[0]+v[0], pos[1]+v[1]
		if _, ok := l.loaded[pos]; !ok {
			l.loadQueue = append(l.loadQueue, pos)
		}
	}
}

// calcUnusedChunks розраховує які чанки можна вивантажити
// Чанк вивантажується якщо він знаходиться за межами радіусу видимості
func (l *loader) calcUnusedChunks() {
	l.unloadQueue = l.unloadQueue[:0]
	for chunk := range l.loaded {
		player := l.chunkPosition()
		r := l.chunkRadius()
		if distance2i([2]int32{chunk[0] - player[0], chunk[1] - player[1]}) > float64(r) {
			l.unloadQueue = append(l.unloadQueue, chunk)
		}
	}
}

// loadList містить відносні координати чанків відносно центру (0,0)
// Відсортовані за відстанню від центру - ближчі чанки йдуть першими
var loadList [][2]int32

// radiusIdx[i] містить кількість чанків у loadList для радіусу i
// Використовується для швидкого знаходження потрібної кількості чанків
var radiusIdx []int

// init ініціалізує глобальні змінні loadList та radiusIdx
// Викликається автоматично при старті програми
func init() {
	const maxR int32 = 32 // максимальний радіус завантаження

	// Заповнюємо loadList всіма можливими позиціями чанків
	for x := -maxR; x <= maxR; x++ {
		for z := -maxR; z <= maxR; z++ {
			pos := [2]int32{x, z}
			if distance2i(pos) < float64(maxR) {
				loadList = append(loadList, pos)
			}
		}
	}
	// Сортуємо за відстанню від центру
	sort.Slice(loadList, func(i, j int) bool {
		return distance2i(loadList[i]) < distance2i(loadList[j])
	})

	// Заповнюємо radiusIdx
	radiusIdx = make([]int, maxR+1)
	for i, v := range loadList {
		r := int32(math.Ceil(distance2i(v)))
		if r > maxR {
			break
		}
		radiusIdx[r] = i
	}
}

// distance2i обчислює Евклідову відстань від точки до початку координат
// Використовується для визначення чи чанк входить в радіус завантаження
func distance2i(pos [2]int32) float64 {
	return math.Sqrt(float64(pos[0]*pos[0]) + float64(pos[1]*pos[1]))
}
