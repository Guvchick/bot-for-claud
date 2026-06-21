package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

func (a *App) notifyAdmins(text string) {
	for adminID := range a.cfg.AdminIDs {
		if _, err := a.tg.SendMessage(adminID, text, nil); err != nil {
			log.Printf("admin notification failed: admin_id=%d err=%v", adminID, err)
		}
	}
}

// notifyGroup sends a message to the configured log group. It is best-effort and
// intentionally silent on failure: logging here would feed the log-forwarder and
// create a notification loop.
func (a *App) notifyGroup(text string) {
	if a.cfg.LogGroupID == 0 {
		return
	}
	_, _ = a.tg.SendMessage(a.cfg.LogGroupID, text, nil)
}

// auditEvent records a structured event: it is logged at EVENT level (kept out of
// the debug stream) and mirrored to the log group as a clean, formatted message.
func (a *App) auditEvent(emoji, title string, lines ...string) {
	log.Printf("event: %s", title)
	if a.cfg.LogGroupID == 0 {
		return
	}
	text := emoji + " <b>" + esc(title) + "</b>"
	if len(lines) > 0 {
		text += "\n\n" + strings.Join(lines, "\n")
	}
	a.notifyGroup(text)
}

// notifyUser delivers a push message only if the user has that notification kind
// enabled. It fails open on DB errors (see DB.NotificationEnabled).
func (a *App) notifyUser(id int64, kind, text string, markup *InlineKeyboardMarkup) {
	if !a.db.NotificationEnabled(id, kind) {
		return
	}
	if _, err := a.tg.SendMessage(id, text, markup); err != nil {
		log.Printf("user notification failed: telegram_id=%d kind=%s err=%v", id, kind, err)
	}
}

// setupLogForwarding streams WARN/ERROR log lines to the log group, throttled and
// non-blocking. The drain goroutine never logs on failure so it cannot feed itself.
func (a *App) setupLogForwarding() {
	if a.cfg.LogGroupID == 0 || !a.cfg.LogGroupForwardErrors || a.logWriter == nil {
		return
	}
	a.logForward = make(chan string, 256)
	a.logWriter.SetForward(func(level int, label, line string) {
		msg := "🪵 <b>" + esc(label) + "</b>\n<code>" + esc(shortText(line, 1500)) + "</code>"
		select {
		case a.logForward <- msg:
		default:
		}
	})
	go func() {
		for msg := range a.logForward {
			_, _ = a.tg.SendMessage(a.cfg.LogGroupID, msg, nil)
			time.Sleep(1200 * time.Millisecond)
		}
	}()
}

func (a *App) notifyStartup() {
	if !a.cfg.NotifyAdminsOnStart {
		return
	}
	text := fmt.Sprintf(
		"🟢 <b>Бот запущен</b>\n\n"+
			"Время: <code>%s</code>\n"+
			"Облако: <code>%s</code>\n"+
			"Telegram API: <code>%s</code>\n"+
			"Local Bot API: <b>%s</b>\n"+
			"Workers: <b>%d</b>\n"+
			"Webhook: <code>%s</code>",
		time.Now().Format("2006-01-02 15:04:05"),
		esc(a.cfg.NextcloudURL),
		esc(a.cfg.TelegramAPIBaseURL),
		mapBool(a.cfg.TelegramLocalMode, "да", "нет"),
		a.cfg.UploadWorkers,
		esc(a.cfg.WebhookListenAddr),
	)
	a.notifyAdmins(text)
	a.notifyGroup(text)
}

func (a *App) notifyCrash(scope string, recovered any, stack []byte) {
	if !a.cfg.NotifyAdminsOnCrash {
		return
	}
	stackText := strings.TrimSpace(string(stack))
	if len([]rune(stackText)) > 2500 {
		stackText = string([]rune(stackText)[:2500]) + "\n..."
	}
	text := fmt.Sprintf(
		"🚨 <b>Краш в боте</b>\n\n"+
			"Место: <code>%s</code>\n"+
			"Ошибка: <code>%s</code>\n\n"+
			"<b>Stack:</b>\n<code>%s</code>",
		esc(scope),
		esc(fmt.Sprint(recovered)),
		esc(stackText),
	)
	a.notifyAdmins(text)
	a.notifyGroup(text)
}

func (a *App) safeGo(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				log.Printf("panic in goroutine %s: %v\n%s", name, r, string(stack))
				a.notifyCrash("goroutine: "+name, r, stack)
			}
		}()
		fn()
	}()
}

func (a *App) recoverHTTP(name string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				stack := debug.Stack()
				log.Printf("panic in webhook %s: %v\n%s", name, recovered, string(stack))
				a.notifyCrash("webhook: "+name, recovered, stack)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		handler(w, r)
	}
}
