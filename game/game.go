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

package game

import (
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"FlowyCore/client"
	"FlowyCore/world"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/data/packetid"
	"github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/save"
	"github.com/Tnze/go-mc/server"
	"github.com/Tnze/go-mc/yggdrasil/user"
)

type Game struct {
	log *zap.Logger

	config     Config
	serverInfo *server.PingInfo

	playerProvider world.PlayerProvider
	overworld      *world.World

	globalChat globalChat
	*playerList
}

func NewGame(log *zap.Logger, config Config, pingList *server.PlayerList, serverInfo *server.PingInfo) *Game {
	// providers
	overworld, err := createWorld(log, filepath.Join(".", config.LevelName), &config)
	if err != nil {
		log.Fatal("cannot load overworld", zap.Error(err))
	}
	playerProvider := world.NewPlayerProvider(filepath.Join(".", config.LevelName, "playerdata"))

	// keepalive
	keepAlive := server.NewKeepAlive()
	pl := playerList{pingList: pingList, keepAlive: keepAlive}
	keepAlive.AddPlayerDelayUpdateHandler(func(c server.KeepAliveClient, latency time.Duration) {
		pl.updateLatency(c.(*client.Client), latency)
	})
	go keepAlive.Run(context.TODO())

	return &Game{
		log: log.Named("game"),

		config:     config,
		serverInfo: serverInfo,

		playerProvider: playerProvider,
		overworld:      overworld,

		globalChat: globalChat{
			log:           log.Named("chat"),
			players:       &pl,
			chatTypeCodec: &world.NetworkCodec.ChatType,
		},
		playerList: &pl,
	}
}

// Йоу, чат! Зараз розберемо як створюється світ в майнкрафті!
// createWorld створює новий світ або завантажує існуючий
func createWorld(logger *zap.Logger, path string, config *Config) (*world.World, error) {
	// Відкриваємо файл level.dat - це головний файл світу
	// Тут зберігається вся базова інформація - спавн, сід, час, погода
	f, err := os.Open(filepath.Join(path, "level.dat"))
	// Якщо файл не знайдено - повертаємо помилку
	if err != nil {
		return nil, err
	}

	// level.dat зжатий через gzip для економії місця
	// Створюємо читач який розпакує дані
	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	// Читаємо NBT дані з файлу
	// NBT - це формат в якому майн зберігає всі дані
	lv, err := save.ReadLevel(r)
	if err != nil {
		return nil, err
	}

	// Створюємо новий світ (точніше вимір - overworld)
	overworld := world.New(
		// Додаємо до логера префікс "overworld"
		logger.Named("overworld"),
		// Створюємо провайдер який буде читати чанки з папки region
		// ChunkLoadingLimiter обмежує скільки чанків можна загрузити одночасно
		world.NewProvider(filepath.Join(path, "region"), config.ChunkLoadingLimiter.Limiter()),
		// Налаштування світу:
		world.Config{
			// На яку відстань гравці бачать світ (в чанках)
			ViewDistance: config.ViewDistance,
			// Кут повороту на спавні
			SpawnAngle: lv.Data.SpawnAngle,
			// Координати точки спавну
			SpawnPosition: [3]int32{lv.Data.SpawnX, lv.Data.SpawnY, lv.Data.SpawnZ},
		},
	)
	return overworld, nil
}

// Йоу, чат! А тепер розберемо як гравець заходить на сервер!
// AcceptPlayer викликається в окремій горутині коли новий гравець логіниться
func (g *Game) AcceptPlayer(name string, id uuid.UUID, profilePubKey *user.PublicKey, properties []user.Property, protocol int32, conn *net.Conn) {
	// Створюємо логер для цього гравця
	// Додаємо його нік, UUID і версію протоколу щоб легше було дебажити
	logger := g.log.With(
		zap.String("name", name),
		zap.String("uuid", id.String()),
		zap.Int32("protocol", protocol),
	)

	// Пробуємо завантажити дані гравця з файлу
	p, err := g.playerProvider.GetPlayer(name, id, profilePubKey, properties)
	// Якщо файл не знайдено - створюємо нового гравця
	if errors.Is(err, os.ErrNotExist) {
		p = &world.Player{
			// Базові поля Entity - ID, позиція, поворот
			Entity: world.Entity{
				// Генеруємо унікальний ID для цієї сутності
				EntityID: world.NewEntityID(),
				// Початкові координати гравця
				Position: [3]float64{48, 100, 35},
				// Кути повороту (дивиться прямо)
				Rotation: [2]float32{},
			},
			// Нік гравця - показується над головою
			Name: name,
			// UUID - унікальний ID акаунта
			UUID: id,
			// Публічний ключ для шифрування чату
			PubKey: profilePubKey,
			// Properties містять скін та інші дані профілю
			Properties: properties,
			// Gamemode: 0 - виживання, 1 - креатив
			Gamemode: 1,
			// В якому чанку знаходиться гравець
			// Ділимо координати на 16 щоб отримати номер чанка
			ChunkPos: [3]int32{48 >> 4, 64 >> 4, 35 >> 4},
			// Список сутностей які гравець бачить
			EntitiesInView: make(map[int32]*world.Entity),
			// Радіус прогрузки в чанках
			ViewDistance: 10,
		}
		// Якщо сталася інша помилка - логуємо і виходимо
	} else if err != nil {
		logger.Error("Read player data error", zap.Error(err))
		return
	}

	// Створюємо нового клієнта для гравця
	c := client.New(logger, conn, p)

	// Логуємо що гравець зайшов
	logger.Info("Player join", zap.Int32("eid", p.EntityID))
	// Коли функція закінчиться - запишемо що гравець вийшов
	defer logger.Info("Player left")

	// Відправляємо пакети логіну:
	// - Інформація про світ
	c.SendLogin(g.overworld, p)
	// - Налаштування серверу (MOTD, іконка)
	c.SendServerData(g.serverInfo.Description(), g.serverInfo.FavIcon(), g.config.EnforceSecureProfile)

	// Створюємо повідомлення про вхід/вихід гравця
	// Жовтим кольором, як в оригінальному майні
	joinMsg := chat.TranslateMsg("multiplayer.player.joined", chat.Text(p.Name)).SetColor(chat.Yellow)
	leftMsg := chat.TranslateMsg("multiplayer.player.left", chat.Text(p.Name)).SetColor(chat.Yellow)
	// Відправляємо повідомлення всім гравцям
	g.globalChat.broadcastSystemChat(joinMsg, false)
	// Коли гравець вийде - відправимо повідомлення про вихід
	defer g.globalChat.broadcastSystemChat(leftMsg, false)
	// Додаємо обробник чату для цього гравця
	c.AddHandler(packetid.ServerboundChat, g.globalChat.Handle)

	// Додаємо гравця в список гравців (табліст)
	g.playerList.addPlayer(c, p)
	// Коли вийде - видалимо зі списку
	defer g.playerList.removePlayer(c)

	// Телепортуємо гравця на його позицію
	c.SendPlayerPosition(p.Position, p.Rotation)
	// Додаємо гравця в світ (це почне відправку чанків)
	g.overworld.AddPlayer(c, p, g.config.PlayerChunkLoadingLimiter.Limiter())
	// Коли вийде - видалимо зі світу
	defer g.overworld.RemovePlayer(c, p)
	// Відправляємо теги (використовуються для команд)
	c.SendPacket(packetid.ClientboundUpdateTags, pk.Array(defaultTags))
	// Встановлюємо точку спавну
	c.SendSetDefaultSpawnPosition(g.overworld.SpawnPositionAndAngle())

	// Запускаємо головний цикл обробки пакетів
	c.Start()
}

// ChunkPos визначає позицію чанка в світі
// Чанк - це куб 16x16x16 блоків
// Координати чанка отримуємо діленням координат блока на 16 (побітовий зсув >> 4)
// [3]int32{48 >> 4, 64 >> 4, 35 >> 4} означає:
// X = 48/16 = 3 (чанк номер 3 по X)
// Y = 64/16 = 4 (чанк номер 4 по висоті)
// Z = 35/16 = 2 (чанк номер 2 по Z)
// NBT (Named Binary Tag) - це формат даних який використовує Minecraft
// В NBT зберігається вся інформація про блоки, предмети, інвентарі, etc
// Кожен тег має свій тип (byte, short, int, long, float, double, string, list, compound)
// І кожен тег має ім'я, щоб можна було знайти потрібні дані

// Йоу, чат! Зараз розберемо як влаштований світ в майнкрафті!
// Світ поділений на чанки - це кубики 16x16x16 блоків
// Коли гравець переміщується, сервер підгружає нові чанки і вивантажує старі

// Entity - це базовий клас для всіх об'єктів у грі
// Кожен Entity має:
// - Унікальний ID (EntityID)
// - Позицію у світі (Position)
// - Кут повороту (Rotation)
type Entity struct {
	EntityID int32
	Position [3]float64
	Rotation [2]float32
}

// Player - це гравець, який розширює Entity додатковими полями
type Player struct {
	Entity
	// Ім'я гравця - показується над головою і в чаті
	Name string
	// UUID - унікальний ідентифікатор акаунта
	UUID uuid.UUID
	// PubKey - публічний ключ для шифрування чату
	PubKey *user.PublicKey
	// Properties - скін та інші властивості профілю
	Properties []user.Property
	// Gamemode: 0 - виживання, 1 - креатив, 2 - пригоди, 3 - спостерігач
	Gamemode int32

	// ChunkPos - в якому чанку знаходиться гравець
	// Розраховується діленням координат на 16 (або зсувом >> 4)
	// [3]int32{48 >> 4, 64 >> 4, 35 >> 4} означає:
	// X = 48/16 = 3 (третій чанк по X)
	// Y = 64/16 = 4 (четвертий чанк по висоті)
	// Z = 35/16 = 2 (другий чанк по Z)
	ChunkPos [3]int32

	// EntitiesInView - список Entity які гравець бачить
	// Коли Entity виходить з радіусу видимості - видаляємо його з мапи
	EntitiesInView map[int32]*Entity

	// ViewDistance - на яку відстань (в чанках) гравець бачить світ
	// За замовчуванням 10 чанків = 160 блоків
	ViewDistance int32
}
