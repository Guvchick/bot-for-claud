package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"
)

func main() {
	loadDotEnv(".env")
	cfg := loadConfig()
	logWriter := configureLogging(cfg)
	uploadTimeout := time.Duration(cfg.UploadTimeoutMinutes) * time.Minute
	redis, err := NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis config: %v", err)
	}
	db, err := NewDB(cfg)
	if err != nil {
		log.Fatalf("postgres config: %v", err)
	}

	app := &App{
		cfg: cfg,
		tg: &Telegram{
			token:           cfg.BotToken,
			apiURL:          cfg.TelegramAPIBaseURL + "/bot" + cfg.BotToken + "/",
			fileURL:         cfg.TelegramFileBaseURL + "/bot" + cfg.BotToken + "/",
			localMode:       cfg.TelegramLocalMode,
			localPathPrefix: cfg.TelegramLocalPathPrefix,
			botPathPrefix:   cfg.TelegramBotPathPrefix,
			client:          &http.Client{Timeout: 90 * time.Second},
			downloadClient:  &http.Client{Timeout: uploadTimeout},
		},
		db: db,
		nc: &Nextcloud{
			baseURL:      strings.TrimRight(cfg.NextcloudInternalURL, "/"),
			username:     cfg.NextcloudAdminUser,
			password:     cfg.NextcloudAdminPassword,
			client:       &http.Client{Timeout: 90 * time.Second},
			uploadClient: &http.Client{Timeout: uploadTimeout},
			chunkSize:    int64(cfg.UploadChunkSizeMB) * 1024 * 1024,
		},
		states:    NewStateStore(redis),
		uploads:   NewUploadQueue(),
		batches:   NewUploadBatchManager(),
		quota:     NewQuotaCache(time.Duration(cfg.QuotaCacheSeconds) * time.Second),
		stickers:  NewStickerStore(cfg.StickerStoreFile),
		content:   NewContentStore(cfg.ContentStoreFile),
		banners:   NewBannerStore(cfg),
		logWriter: logWriter,
	}
	if err := app.stickers.Load(); err != nil {
		log.Printf("sticker store load failed: %v", err)
	}
	if err := app.content.Load(); err != nil {
		log.Printf("content store load failed: %v", err)
	}
	if err := app.banners.Load(); err != nil {
		log.Printf("banner cache load failed: %v", err)
	}
	if cfg.PlategaEnabled && cfg.PlategaMerchantID != "" && cfg.PlategaSecret != "" {
		app.platega = &Platega{
			merchantID: cfg.PlategaMerchantID,
			secret:     cfg.PlategaSecret,
			baseURL:    cfg.PlategaBaseURL,
			client:     &http.Client{Timeout: 30 * time.Second},
		}
	}
	if cfg.PallyEnabled && cfg.PallyToken != "" && cfg.PallyShopID != "" {
		app.pally = &Pally{
			token:   cfg.PallyToken,
			shopID:  cfg.PallyShopID,
			baseURL: cfg.PallyBaseURL,
			client:  &http.Client{Timeout: 30 * time.Second},
		}
	}
	if cfg.CryptoBotEnabled && cfg.CryptoBotToken != "" {
		app.cryptoBot = &CryptoBotPay{
			token:   cfg.CryptoBotToken,
			baseURL: cfg.CryptoBotBaseURL,
			client:  &http.Client{Timeout: 30 * time.Second},
		}
	}
	if cfg.HeleketEnabled && cfg.HeleketMerchantID != "" && cfg.HeleketAPIKey != "" {
		app.heleket = &Heleket{
			merchantID: cfg.HeleketMerchantID,
			apiKey:     cfg.HeleketAPIKey,
			baseURL:    cfg.HeleketBaseURL,
			client:     &http.Client{Timeout: 30 * time.Second},
		}
	}

	if err := app.db.Init(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	app.setupLogForwarding()
	app.printStartupBanner()
	app.logRuntimeHints()
	app.notifyStartup()
	app.warmBanners()

	for i := 0; i < cfg.UploadWorkers; i++ {
		workerID := i + 1
		app.safeGo(fmt.Sprintf("upload-worker-%d", workerID), func() { app.uploadWorker(workerID) })
	}
	app.safeGo("auto-backup", app.autoBackupLoop)
	app.safeGo("nextcloud-sync", app.nextcloudSyncLoop)
	app.safeGo("premium-expiration", app.premiumExpirationLoop)
	app.safeGo("webhook-server", app.startWebhookServer)
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			log.Printf("panic in poll loop: %v\n%s", r, string(stack))
			app.notifyCrash("poll-loop", r, stack)
		}
	}()
	app.poll()
}

// printStartupBanner writes a clean, human-readable summary straight to stdout
// (bypassing the leveled logger so it stays unprefixed), then logs a one-line
// machine-readable summary through the logger.
func (a *App) printStartupBanner() {
	mode := "public Telegram Bot API"
	if a.cfg.TelegramLocalMode {
		mode = "local Bot API (--local)"
	}
	group := "—"
	if a.cfg.LogGroupID != 0 {
		group = fmt.Sprintf("%d", a.cfg.LogGroupID)
	}
	banners := 0
	if a.banners != nil {
		banners = len(a.banners.Keys())
	}
	color := colorEnabled()
	paint := func(code, s string) string {
		if !color {
			return s
		}
		return "\x1b[" + code + "m" + s + "\x1b[0m"
	}
	const (
		accent = "38;5;44"   // cyan border
		title  = "1;38;5;81" // bold light cyan
		key    = "38;5;244"  // dim labels
		val    = "38;5;231"  // bright values
		rule   = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	)
	bar := paint(accent, "  ┃")
	row := func(label, value string) string {
		return bar + "  " + paint(key, fmt.Sprintf("%-11s", label)) + paint(val, value)
	}
	fmt.Println()
	fmt.Println(paint(accent, "  ┏"+rule))
	fmt.Println(bar + "  " + paint(title, "☁️  NextCloud × Telegram Bot"))
	fmt.Println(paint(accent, "  ┣"+rule))
	fmt.Println(row("Nextcloud", a.cfg.NextcloudURL))
	fmt.Println(row("Telegram", mode))
	fmt.Println(row("Workers", fmt.Sprintf("%d", a.cfg.UploadWorkers)))
	fmt.Println(row("Max upload", fmt.Sprintf("%d MB · chunk %d MB", a.cfg.TelegramMaxDownloadMB, a.cfg.UploadChunkSizeMB)))
	fmt.Println(row("Log group", group))
	fmt.Println(row("Banners", fmt.Sprintf("%d configured", banners)))
	fmt.Println(row("Webhook", a.cfg.WebhookListenAddr))
	fmt.Println(paint(accent, "  ┗"+rule))
	fmt.Println()
	log.Printf("bot started: nextcloud=%s telegram_local_mode=%v workers=%d max_upload_mb=%d chunk_mb=%d log_group=%v banners=%d",
		a.cfg.NextcloudURL, a.cfg.TelegramLocalMode, a.cfg.UploadWorkers, a.cfg.TelegramMaxDownloadMB, a.cfg.UploadChunkSizeMB, a.cfg.LogGroupID != 0, banners)
}

func (a *App) logRuntimeHints() {
	if a.cfg.TelegramLocalMode {
		log.Printf("telegram local mode enabled: api=%s file_api=%s local_path_prefix=%s bot_path_prefix=%s max_download_mb=%d", a.cfg.TelegramAPIBaseURL, a.cfg.TelegramFileBaseURL, a.cfg.TelegramLocalPathPrefix, a.cfg.TelegramBotPathPrefix, a.cfg.TelegramMaxDownloadMB)
		if strings.Contains(a.cfg.TelegramAPIBaseURL, "telegram-bot-api") {
			log.Printf("telegram local mode hint: docker compose must run profile telegram-local; set COMPOSE_PROFILES=telegram-local if service name is telegram-bot-api")
		}
	} else if a.cfg.TelegramMaxDownloadMB > 20 {
		log.Printf("telegram public api warning: TELEGRAM_MAX_DOWNLOAD_MB=%d but TELEGRAM_LOCAL_MODE=false; public Bot API may still reject large files", a.cfg.TelegramMaxDownloadMB)
	}
	if a.cfg.PlategaEnabled && a.cfg.PlategaCallbackURL == "" {
		log.Printf("platega warning: PLATEGA_CALLBACK_URL is empty; payments can still be checked manually, but automatic webhook fulfillment will not work")
	}
	if a.cfg.CryptoBotEnabled && strings.Trim(a.cfg.CryptoBotWebhookPath, "/") == "" {
		log.Printf("cryptobot warning: CRYPTOBOT_WEBHOOK_PATH must be a secret path for webhook verification")
	}
	if a.cfg.HeleketEnabled && a.publicWebhookURL(a.cfg.HeleketWebhookPath) == "" {
		log.Printf("heleket warning: set PUBLIC_WEBHOOK_BASE_URL or PLATEGA_CALLBACK_URL so bot can pass Heleket url_callback")
	}
}

func (a *App) publicWebhookURL(path string) string {
	base := strings.TrimSpace(a.cfg.PublicWebhookBaseURL)
	if base == "" {
		base = strings.TrimSpace(a.cfg.PlategaCallbackURL)
	}
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	if path == "" {
		return base
	}
	u.Path = path
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func (a *App) poll() {
	offset := 0
	for {
		updates, err := a.tg.GetUpdates(offset)
		if err != nil {
			log.Printf("getUpdates failed: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			a.handleUpdate(update)
		}
	}
}

func (a *App) handleUpdate(update Update) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			log.Printf("panic while handling update %d: %v\n%s", update.UpdateID, r, string(stack))
			a.notifyCrash(fmt.Sprintf("update:%d", update.UpdateID), r, stack)
		}
	}()
	if update.PreCheckoutQuery != nil {
		a.tg.AnswerPreCheckout(update.PreCheckoutQuery.ID, strings.HasPrefix(update.PreCheckoutQuery.InvoicePayload, "stars_donate:"))
		return
	}
	if update.CallbackQuery != nil {
		a.handleCallback(update.CallbackQuery)
		return
	}
	if update.Message != nil {
		a.handleMessage(update.Message)
	}
}

func (a *App) handleMessage(msg *Message) {
	if msg.From == nil {
		return
	}
	userID := msg.From.ID
	if msg.SuccessfulPayment != nil {
		a.handleSuccessfulPayment(msg)
		return
	}
	if !a.isAdmin(userID) && a.maintenanceEnabled() {
		_, _ = a.sendContent(msg.Chat.ID, "maintenance", nil, nil)
		return
	}
	if st, ok := a.states.Get(userID); ok {
		a.handleStateMessage(msg, st)
		return
	}
	if strings.HasPrefix(msg.Text, "/") {
		a.handleCommand(msg)
		return
	}
	if a.hasUpload(msg) {
		a.handleUpload(msg)
	}
}

func (a *App) handleCommand(msg *Message) {
	parts := strings.Fields(msg.Text)
	command := strings.Split(strings.TrimPrefix(parts[0], "/"), "@")[0]
	switch command {
	case "start":
		a.start(msg)
	case "admin":
		if a.isAdmin(msg.From.ID) {
			a.present(nil, msg.Chat.ID, "admin", a.adminSummary(), adminKeyboard())
		}
	case "health":
		if a.isAdmin(msg.From.ID) {
			a.health(msg.Chat.ID)
		}
	case "sync":
		if a.isAdmin(msg.From.ID) {
			checked, removed, err := a.syncNextcloudUsers()
			if err != nil {
				_, _ = a.tg.SendMessage(msg.Chat.ID, "⚠️ Синхронизация не удалась: <code>"+esc(err.Error())+"</code>", nil)
				return
			}
			_, _ = a.tg.SendMessage(msg.Chat.ID, fmt.Sprintf("🔄 <b>Синхронизация завершена</b>\n\nПроверено: <b>%d</b>\nУдалено: <b>%d</b>", checked, removed), adminKeyboard())
		}
	case "search":
		if a.isAdmin(msg.From.ID) {
			query := strings.TrimSpace(strings.TrimPrefix(msg.Text, parts[0]))
			a.renderSearch(msg.Chat.ID, query)
		}
	case "broadcast":
		if a.isAdmin(msg.From.ID) {
			text := strings.TrimSpace(strings.TrimPrefix(msg.Text, parts[0]))
			a.broadcastText(msg.Chat.ID, text)
		}
	case "setsticker":
		if a.isAdmin(msg.From.ID) {
			a.setStickerStart(msg, parts)
		}
	case "stickers":
		if a.isAdmin(msg.From.ID) {
			_, _ = a.tg.SendMessage(msg.Chat.ID, a.stickersText(), stickersKeyboard(a.stickers, a.cfg.StickerPackURL))
		}
	}
}
