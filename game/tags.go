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

// Йоу, чат! Зараз розберемо як працюють теги в майнкрафті!
// Теги - це спосіб групувати блоки, предмети або інші речі за певними ознаками

package game

import (
	// io для запису даних
	"io"
	// пакет для роботи з пакетами майнкрафту
	pk "github.com/Tnze/go-mc/net/packet"
)

// Tag представляє групу елементів з спільними властивостями
// T може бути або int32 або int - це ID елементів
// Наприклад: тег "minecraft:logs" містить всі види деревини
type Tag[T ~int32 | ~int] struct {
	// Назва тегу (наприклад "minecraft:logs")
	Name string
	// Мапа значень тегу
	// Ключ - назва елементу (наприклад "minecraft:oak_log")
	// Значення - список ID цього елементу
	Values map[string][]T
}

// WriteTo записує тег в бінарний формат для відправки клієнту
// Реалізує інтерфейс io.WriterTo
func (t Tag[T]) WriteTo(w io.Writer) (n int64, err error) {
	// Записуємо назву тегу
	n1, err := pk.Identifier(t.Name).WriteTo(w)
	if err != nil {
		return n1, err
	}

	// Записуємо кількість елементів в мапі
	n2, err := pk.VarInt(len(t.Values)).WriteTo(w)
	if err != nil {
		return n1 + n2, err
	}

	// Для кожного елементу в мапі:
	for k, v := range t.Values {
		// Записуємо назву елементу
		n3, err := pk.Identifier(k).WriteTo(w)
		n += n3
		if err != nil {
			return n + n1 + n2, err
		}

		// Записуємо кількість ID цього елементу
		n4, err := pk.VarInt(len(v)).WriteTo(w)
		n += n4
		if err != nil {
			return n + n1 + n2, err
		}

		// Записуємо кожен ID
		for _, v := range v {
			n5, err := pk.VarInt(v).WriteTo(w)
			n += n5
			if err != nil {
				return n + n1 + n2, err
			}
		}
	}
	return n + n1 + n2, err
}

// defaultTags - стандартні теги які відправляються клієнту
// Поки що тут тільки рідини (вода і лава)
var defaultTags = []pk.FieldEncoder{
	Tag[int32]{
		Name: "minecraft:fluid",
		Values: map[string][]int32{
			// Вода: стояча (1) і текуча (2)
			"minecraft:water": {1, 2},
			// Лава: стояча (3) і текуча (4)
			"minecraft:lava": {3, 4},
		},
	},
}
