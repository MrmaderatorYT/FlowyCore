// Йоу, чат! Сьогодні ми будемо розбирати як створити ядро майнкрафт сервера!
// Це ліцензія AGPL - означає що наш код має бути відкритим, і всі модифікації теж.
// Це важливо для спільноти, щоб всі могли вчитися і покращувати код!

// Пакет main - це точка входу нашої програми, звідси все починається!
package main

import (
	// Імпортуємо наше ігрове ядро - тут вся магія відбувається!
	"FlowyCore/game"
	// flag - це пакет для роботи з командним рядком, будемо використовувати для налаштувань
	"flag"
	// debug дозволяє отримати інформацію про збірку програми
	"runtime/debug"
	// strings потрібен для роботи з текстом, будемо використовувати для форматування помилок
	"strings"

	// toml - крутий формат для конфігів, як JSON але читабельніший
	"github.com/BurntSushi/toml"
	// zap - мегашвидкий логер, набагато швидший за fmt.Printf
	"go.uber.org/zap"

	// chat - бібліотека для роботи з текстом в майнкрафті
	// підтримує кольори, форматування, переклади
	"github.com/Tnze/go-mc/chat"
	// server - основна бібліотека для створення серверу
	"github.com/Tnze/go-mc/server"
)

// isDebug - флаг який можна включити при запуску через -debug
// В дебаг режимі буде більше логів і інформації для розробки
var isDebug = flag.Bool("debug", false, "Enable debug log output")

func main() {
	// Парсимо командний рядок - шукаємо наш флаг -debug
	flag.Parse()

	// Створюємо логер - він буде записувати все що відбувається на сервері
	// В дебаг режимі логи будуть детальніші, але повільніші
	// В продакшені логи оптимізовані для швидкодії
	var logger *zap.Logger
	// Якщо включений дебаг - використовуємо розширені логи
	if *isDebug {
		logger = unwrap(zap.NewDevelopment())
	} else {
		// Інакше - швидкі продакшен логи
		logger = unwrap(zap.NewProduction())
	}

	// defer - це магія Go, цей код виконається коли функція закінчиться
	// Тут ми закриваємо логер, щоб всі логи записались на диск
	defer func(logger *zap.Logger) {
		// Синхронізуємо буфер логів з диском
		if err := logger.Sync(); err != nil {
			// Якщо щось пішло не так - панікуємо
			panic(err)
		}
	}(logger)

	// Пишемо в лог що сервер запустився
	logger.Info("Server start")
	// Виводимо інформацію про версію і налаштування збірки
	printBuildInfo(logger)
	// Коли сервер завершиться - запишемо про це в лог
	defer logger.Info("Server exit")

	// Читаємо налаштування з файлу config.toml
	// Там зберігається порт, максимум гравців, назва серверу і т.д.
	config, err := readConfig()
	// Якщо не змогли прочитати конфіг - пишемо помилку і виходимо
	if err != nil {
		logger.Error("Read config fail", zap.Error(err))
		return
	}

	// PlayerList зберігає список всіх гравців на сервері
	// MaxPlayers - максимальна кількість гравців, за замовчуванням 20
	// В протоколі майнкрафту ID гравця - це 1 байт, тому максимум 255
	playerList := server.NewPlayerList(config.MaxPlayers)

	// ServerInfo - це те, що бачать гравці в списку серверів
	// Тут задається назва, версія, опис, іконка серверу
	serverInfo := server.NewPingInfo(
		// Назва серверу - буде показуватись в списку
		"Go-MC "+server.ProtocolName,
		// Версія протоколу - кожна версія майну має свій номер
		server.ProtocolVersion,
		// MOTD - повідомлення дня, показується під назвою
		chat.Text(config.MessageOfTheDay),
		// Іконка серверу - поки що не використовуємо
		nil,
	)
	// Якщо не вдалося створити ServerInfo - виходимо
	if err != nil {
		logger.Error("Init server info system fail", zap.Error(err))
		return
	}

	// Створюємо сам сервер - це головний об'єкт який все контролює
	s := server.Server{
		// Налаштовуємо логер для серверу
		Logger: zap.NewStdLog(logger),
		// ListPingHandler відповідає за пінг і список гравців
		ListPingHandler: struct {
			*server.PlayerList
			*server.PingInfo
		}{playerList, serverInfo},
		// LoginHandler перевіряє гравців при вході
		LoginHandler: &server.MojangLoginHandler{
			// OnlineMode - перевірка ліцензії
			OnlineMode: config.OnlineMode,
			// EnforceSecureProfile - вимагати безпечний профіль
			EnforceSecureProfile: config.EnforceSecureProfile,
			// Threshold - з якого розміру стискати пакети
			// Пакети більше 256 байт будуть стиснуті для економії трафіку
			Threshold: config.NetworkCompressionThreshold,
			// LoginChecker перевіряє чи можна зайти на сервер
			LoginChecker: playerList,
		},
		// GamePlay - наше ігрове ядро, вся логіка гри тут
		GamePlay: game.NewGame(logger, config, playerList, serverInfo),
	}

	// Запускаємо сервер на вказаному адресі
	// За замовчуванням це 0.0.0.0:25565 - стандартний порт майнкрафту
	logger.Info("Start listening", zap.String("address", config.ListenAddress))
	// Починаємо слухати підключення
	err = s.Listen(config.ListenAddress)
	// Якщо сталася помилка - пишемо в лог
	if err != nil {
		logger.Error("Server listening error", zap.Error(err))
	}
}

// printBuildInfo виводить інформацію про збірку
// Це допомагає знайти проблеми з версіями бібліотек
func printBuildInfo(logger *zap.Logger) {
	binaryInfo, _ := debug.ReadBuildInfo()
	settings := make(map[string]string)
	for _, v := range binaryInfo.Settings {
		settings[v.Key] = v.Value
	}
	logger.Debug("Build info", zap.Any("settings", settings))
}

// readConfig читає конфіг з файлу
// Використовуємо TOML формат - він як INI але потужніший
// Якщо знайдемо невідомі налаштування - повернемо помилку
func readConfig() (game.Config, error) {
	var c game.Config
	meta, err := toml.DecodeFile("config.toml", &c)
	if err != nil {
		return game.Config{}, err
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		var err errUnknownConfig
		for _, key := range undecoded {
			err = append(err, key.String())
		}
		return game.Config{}, err
	}

	return c, nil
}

// errUnknownConfig - це список невідомих налаштувань
// Коли знаходимо щось чого не очікували в конфігу
type errUnknownConfig []string

func (e errUnknownConfig) Error() string {
	return "unknown config keys: [" + strings.Join(e, ", ") + "]"
}

// unwrap - хелпер функція яка спрощує обробку помилок
// Якщо є помилка - відразу панікуємо
func unwrap[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
