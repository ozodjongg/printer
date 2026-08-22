package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"printer/home_printer"

	tele "gopkg.in/telebot.v3"
)

type Config struct {
	BotToken     string  `json:"bot_token"`
	Password     string  `json:"password"`
	AllowedUsers []int64 `json:"allowed_users"`
	DownloadDir  string  `json:"download_dir"`
}

var (
	cfg          *Config
	cfgMutex     sync.RWMutex
	allowedMap   = make(map[int64]bool)
	userSessions = make(map[int64]*home_printer.PrintOptions)
	userStates   = make(map[int64]string) // Foydalanuvchi holati (masalan: awaiting_pages)
	sessionMutex sync.Mutex
)

func getAppDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(execPath)
}

func getConfigPath() string {
	return filepath.Join(getAppDir(), "telegram.json")
}

func loadOrSetupConfig() (*Config, error) {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
		var c Config
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		return &c, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("⚙️ Konfiguratsiya sozlanmoqda...")

	fmt.Print("👉 Telegram Bot Token: ")
	tokenInput, _ := reader.ReadString('\n')

	fmt.Print("👉 Bot uchun yangi parol o'ylab toping: ")
	passInput, _ := reader.ReadString('\n')

	defaultDownloadDir := filepath.Join(getAppDir(), "downloads")

	c := Config{
		BotToken:     strings.TrimSpace(tokenInput),
		Password:     strings.TrimSpace(passInput),
		AllowedUsers: []int64{},
		DownloadDir:  defaultDownloadDir,
	}

	if err := saveConfig(&c); err != nil {
		return nil, err
	}

	fmt.Printf("✅ telegram.json saqlandi: %s\n\n", configPath)
	return &c, nil
}

func saveConfig(c *Config) error {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0644)
}

func buildMenu(opt *home_printer.PrintOptions) *tele.ReplyMarkup {
	m := &tele.ReplyMarkup{}

	btn1Up := m.Data("📄 1-in-1", "opt_1up")
	btn2Up := m.Data("📑 2-in-1", "opt_2up")
	btnBooklet := m.Data("📖 Kitobcha", "opt_booklet")

	colorText := "⚫ Oq-qora"
	if opt.Color == "color" {
		colorText = "🎨 Rangli"
	}
	btnColor := m.Data(colorText, "toggle_color")

	duplexText := "1️⃣ Bir tomonlama"
	if opt.Duplex == "duplex" {
		duplexText = "2️⃣ Ikki tomonlama"
	}
	btnDuplex := m.Data(duplexText, "toggle_duplex")

	pagesText := "Hammasi"
	if opt.PageRange != "" {
		pagesText = opt.PageRange
	}
	btnPages := m.Data(fmt.Sprintf("📑 Sahifalar: %s", pagesText), "set_pages")

	btnMinus := m.Data("➖", "copy_minus")
	btnPlus := m.Data("➕", "copy_plus")
	btnPrint := m.Data("🖨 CHOP ETISH", "opt_print")

	m.Inline(
		m.Row(m.Data(fmt.Sprintf("Format: %s", strings.ToUpper(opt.Mode)), "noop")),
		m.Row(btn1Up, btn2Up, btnBooklet),
		m.Row(btnPages),
		m.Row(btnColor, btnDuplex),
		m.Row(btnMinus, m.Data(fmt.Sprintf("Nusxa: %d ta", opt.Copies), "noop"), btnPlus),
		m.Row(btnPrint),
	)
	return m
}

func main() {
	var err error
	cfg, err = loadOrSetupConfig()
	if err != nil {
		log.Fatalf("Config xatosi: %v", err)
	}

	for _, id := range cfg.AllowedUsers {
		allowedMap[id] = true
	}

	os.MkdirAll(cfg.DownloadDir, os.ModePerm)

	pref := tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	bot.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			if c.Text() != "" && (strings.HasPrefix(c.Text(), "/auth") || c.Text() == "/start") {
				return next(c)
			}
			if !allowedMap[c.Sender().ID] {
				return c.Send("🔒 Botdan foydalanish uchun parolni kiriting:\n`/auth <parol>`", tele.ModeMarkdown)
			}
			return next(c)
		}
	})

	bot.Handle("/start", func(c tele.Context) error {
		if allowedMap[c.Sender().ID] {
			return c.Send("Xush kelibsiz! Chop etish uchun fayl yoki rasm yuboring.")
		}
		return c.Send("Botdan foydalanish uchun parolni kiriting:\n`/auth <parol>`", tele.ModeMarkdown)
	})

	bot.Handle("/auth", func(c tele.Context) error {
		args := c.Args()
		if len(args) == 0 {
			return c.Send("⚠️ Parolni kiriting: `/auth <parol>`", tele.ModeMarkdown)
		}
		if args[0] == cfg.Password {
			userID := c.Sender().ID
			if !allowedMap[userID] {
				allowedMap[userID] = true
				cfg.AllowedUsers = append(cfg.AllowedUsers, userID)
				saveConfig(cfg)
			}
			return c.Send("✅ Parol to'g'ri! Endi botga fayl yoki rasm yuborishingiz mumkin.")
		}
		return c.Send("❌ Parol noto'g'ri!")
	})

	// Sahifa raqamlarini matn ko'rinishida qabul qilish
	bot.Handle(tele.OnText, func(c tele.Context) error {
		userID := c.Sender().ID
		sessionMutex.Lock()
		state := userStates[userID]
		opt := userSessions[userID]
		sessionMutex.Unlock()

		if state == "awaiting_pages" && opt != nil {
			input := strings.TrimSpace(c.Text())
			if strings.ToLower(input) == "all" || input == "0" || input == "-" {
				opt.PageRange = ""
			} else {
				opt.PageRange = input
			}

			sessionMutex.Lock()
			delete(userStates, userID)
			sessionMutex.Unlock()

			c.Send(fmt.Sprintf("✅ Sahifalar o'rnatildi: %s", opt.PageRange))
			return c.Send("⚙️ Parametrlarni tasdiqlang:", buildMenu(opt))
		}

		return c.Send("Fayl yoki rasm yuboring.")
	})

	bot.Handle(tele.OnDocument, func(c tele.Context) error {
		doc := c.Message().Document
		localPath := filepath.Join(cfg.DownloadDir, fmt.Sprintf("%d_%s", time.Now().Unix(), doc.FileName))

		c.Send("📥 Fayl yuklanmoqda...")
		if err := bot.Download(&doc.File, localPath); err != nil {
			return c.Send("❌ Faylni yuklab olishda xatolik!")
		}

		sessionMutex.Lock()
		userSessions[c.Sender().ID] = &home_printer.PrintOptions{
			FilePath:  localPath,
			Mode:      "1up",
			Color:     "monochrome",
			Duplex:    "simplex",
			Copies:    1,
			PageRange: "",
		}
		opt := userSessions[c.Sender().ID]
		sessionMutex.Unlock()

		return c.Send("⚙️ Chop etish parametrlarini sozlang:", buildMenu(opt))
	})

	bot.Handle(tele.OnPhoto, func(c tele.Context) error {
		photo := c.Message().Photo
		localPath := filepath.Join(cfg.DownloadDir, fmt.Sprintf("photo_%d.jpg", time.Now().Unix()))

		c.Send("📥 Rasm yuklanmoqda...")
		if err := bot.Download(&photo.File, localPath); err != nil {
			return c.Send("❌ Rasmni yuklab olishda xatolik!")
		}

		sessionMutex.Lock()
		userSessions[c.Sender().ID] = &home_printer.PrintOptions{
			FilePath:  localPath,
			Mode:      "1up",
			Color:     "monochrome",
			Duplex:    "simplex",
			Copies:    1,
			PageRange: "",
		}
		opt := userSessions[c.Sender().ID]
		sessionMutex.Unlock()

		return c.Send("🖼 Rasm qabul qilindi. Sozlamalarni tanlang:", buildMenu(opt))
	})

	bot.Handle(tele.OnCallback, func(c tele.Context) error {
		userID := c.Sender().ID
		sessionMutex.Lock()
		opt, ok := userSessions[userID]
		sessionMutex.Unlock()

		if !ok || opt == nil {
			return c.Respond(&tele.CallbackResponse{Text: "Seans muddati o'tgan, qaytadan fayl yuboring."})
		}

		data := strings.TrimPrefix(c.Callback().Data, "\t")
		data = strings.TrimSpace(data)

		switch data {
		case "opt_1up":
			opt.Mode = "1up"
		case "opt_2up":
			opt.Mode = "2up"
		case "opt_booklet":
			opt.Mode = "booklet"
		case "set_pages":
			sessionMutex.Lock()
			userStates[userID] = "awaiting_pages"
			sessionMutex.Unlock()
			c.Send("📝 Chop etmoqchi bo'lgan sahifalaringizni yozib yuboring.\n\n*Masalan:* `5-11` yoki `1,3,5-9`\n*(Hammasini chop etish uchun `all` deb yozing)*", tele.ModeMarkdown)
			return c.Respond()
		case "toggle_color":
			if opt.Color == "monochrome" {
				opt.Color = "color"
			} else {
				opt.Color = "monochrome"
			}
		case "toggle_duplex":
			if opt.Duplex == "simplex" {
				opt.Duplex = "duplex"
			} else {
				opt.Duplex = "simplex"
			}
		case "copy_plus":
			opt.Copies++
		case "copy_minus":
			if opt.Copies > 1 {
				opt.Copies--
			}
		case "opt_print":
			c.Send("⏳ Fayl ishlanib, printerga yuborilmoqda...")
			go func(options home_printer.PrintOptions) {
				err := home_printer.Print(options)
				if err != nil {
					bot.Send(c.Sender(), "❌ Chop etishda xatolik: "+err.Error())
				} else {
					bot.Send(c.Sender(), "✅ Fayl muvaffaqiyatli printerga yuborildi!")
				}
				sessionMutex.Lock()
				delete(userSessions, userID)
				delete(userStates, userID)
				sessionMutex.Unlock()
			}(*opt)
			return c.Respond()
		}

		c.Edit("", buildMenu(opt))
		return c.Respond()
	})

	log.Println("🤖 Bot tayyor va ishga tushdi...")
	bot.Start()
}