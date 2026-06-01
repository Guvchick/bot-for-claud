package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (a *App) maintenanceEnabled() bool {
	return a.db.SettingBool("maintenance_enabled", false)
}

func (a *App) statsText() string {
	stats, err := a.db.Stats()
	if err != nil {
		return "📊 Не удалось получить статистику: <code>" + esc(err.Error()) + "</code>"
	}
	conversion := 0.0
	if stats.UsersApproved > 0 {
		conversion = float64(stats.PaidApprovedUsers) / float64(stats.UsersApproved) * 100
	}
	return fmt.Sprintf(
		"📊 <b>Статистика</b>\n<code>━━━━━━━━━━━━━━━━━━━━</code>\n\n"+
			"👥 Пользователи: <b>%d</b>\n📝 Заявки: <b>%d</b>\n✅ Активные: <b>%d</b>\n❌ Отклоненные: <b>%d</b>\n🔴 Отключенные: <b>%d</b>\n⭐ Премиум: <b>%d</b>\n\n"+
			"💾 Выдано квоты: <b>%d GB</b>\n\n"+
			"💳 Платежей: <b>%d</b>\n✅ Подтверждено: <b>%d</b>\n💰 RUB: <b>%d</b>\n💎 Платящих пользователей: <b>%d</b>\n🆓 Бесплатных активных: <b>%d</b>\n📈 Конверсия free→paid: <b>%.1f%%</b>\n⭐ Купили премиум: <b>%d</b>\n💾 Купили место: <b>%d</b>\n\n"+
			"🎟 Промокодов: <b>%d</b>\n🎁 Использований: <b>%d</b>",
		stats.UsersTotal, stats.UsersRequested, stats.UsersApproved, stats.UsersRejected, stats.UsersDisabled, stats.SupportersActive,
		stats.QuotaTotalGB, stats.PaymentsTotal, stats.PaymentsConfirmed, stats.PaymentsRub, stats.PaidApprovedUsers, stats.FreeApprovedUsers, conversion, stats.PremiumBuyers, stats.StorageBuyers,
		stats.PromoCodesTotal, stats.PromoUsesTotal,
	)
}

func (a *App) commerceText() string {
	return fmt.Sprintf(
		"💾 <b>Продажи и триал</b>\n<code>━━━━━━━━━━━━━━━━━━━━</code>\n\n"+
			"Продажа доп. GB: <b>%s</b>\nПакет места: <b>%d GB</b>\nЦена пакета: <b>%d RUB</b>\nСкидка премиум: <b>%d%%</b>\n\n"+
			"Триал: <b>%s</b>\nКвота триала: <b>%d GB</b>\nДней премиума: <b>%d</b>",
		mapBool(a.storageSalesEnabled(), "включена", "выключена"), a.storagePackGB(), a.storagePackPrice(), a.premiumDiscount(),
		mapBool(a.trialEnabled(), "включен", "выключен"), a.trialQuotaGB(), a.trialPremiumDays(),
	)
}

func (a *App) storagePackGB() int    { return a.db.SettingInt("storage_pack_gb", 10) }
func (a *App) storagePackPrice() int { return a.db.SettingInt("storage_pack_price_rub", 100) }
func (a *App) premiumDiscount() int  { return a.db.SettingInt("premium_discount_percent", 0) }
func (a *App) storageSalesEnabled() bool {
	return a.db.SettingBool("storage_sales_enabled", true)
}
func (a *App) trialEnabled() bool    { return a.db.SettingBool("trial_enabled", false) }
func (a *App) trialQuotaGB() int     { return a.db.SettingInt("trial_quota_gb", a.cfg.DefaultQuotaGB) }
func (a *App) trialPremiumDays() int { return a.db.SettingInt("trial_premium_days", 0) }

func (a *App) approvalQuotaGB() int {
	if a.trialEnabled() && a.trialQuotaGB() > a.cfg.DefaultQuotaGB {
		return a.trialQuotaGB()
	}
	return a.cfg.DefaultQuotaGB
}

func (a *App) discountedStoragePrice(user *User) int {
	price := a.storagePackPrice()
	if isPremium(user) {
		discount := a.premiumDiscount()
		if discount > 0 && discount < 100 {
			price = price * (100 - discount) / 100
		}
	}
	if price < 1 {
		price = 1
	}
	return price
}

func (a *App) parseSettingInput(key, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("empty value")
	}
	switch key {
	case "maintenance_enabled", "trial_enabled", "storage_sales_enabled":
		raw := strings.ToLower(value)
		if raw == "1" || raw == "true" || raw == "on" || raw == "yes" || raw == "да" {
			return "true", nil
		}
		if raw == "0" || raw == "false" || raw == "off" || raw == "no" || raw == "нет" {
			return "false", nil
		}
		return "", fmt.Errorf("use true/false")
	case "storage_pack_gb", "storage_pack_price_rub", "premium_discount_percent", "trial_quota_gb", "trial_premium_days":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return "", fmt.Errorf("use non-negative number")
		}
		if (key == "storage_pack_gb" || key == "storage_pack_price_rub" || key == "trial_quota_gb") && n == 0 {
			return "", fmt.Errorf("value must be greater than zero")
		}
		if key == "premium_discount_percent" && n > 99 {
			return "", fmt.Errorf("discount must be 0..99")
		}
		return strconv.Itoa(n), nil
	default:
		return value, nil
	}
}

type storageMetricRow struct {
	User      User
	Used      int64
	Available int64
	Percent   float64
	Err       error
}

func (a *App) storageMetricsText(fresh bool) string {
	users, err := a.db.ListUsers("approved", 100000, 0)
	if err != nil {
		return "💾 Не удалось получить пользователей: <code>" + esc(err.Error()) + "</code>"
	}
	rows := make([]storageMetricRow, 0, len(users))
	checked := 0
	skipped := 0
	failed := 0
	var totalUsed int64
	var knownUsed int64
	var totalAvailable int64
	knownQuotaUsers := 0
	for _, user := range users {
		if user.NCUserID == nil || user.NCPassword == nil {
			skipped++
			continue
		}
		used, available, err := a.userQuota(&user, fresh)
		row := storageMetricRow{User: user, Used: used, Available: available, Err: err}
		if err != nil {
			failed++
			rows = append(rows, row)
			continue
		}
		checked++
		if used > 0 {
			totalUsed += used
		}
		if available >= 0 {
			knownQuotaUsers++
			if used > 0 {
				knownUsed += used
			}
			totalAvailable += available
			total := used + available
			if total > 0 {
				row.Percent = float64(used) / float64(total) * 100
			}
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Err != nil || rows[j].Err != nil {
			return rows[i].Err == nil
		}
		if rows[i].Percent == rows[j].Percent {
			return rows[i].Used > rows[j].Used
		}
		return rows[i].Percent > rows[j].Percent
	})
	quotaLine := "нет данных"
	if knownQuotaUsers > 0 {
		quotaLine = fmt.Sprintf("%s / %s", formatBytes(knownUsed), formatBytes(knownUsed+totalAvailable))
		if knownQuotaUsers != checked {
			quotaLine += fmt.Sprintf(" (%d/%d пользователей)", knownQuotaUsers, checked)
		}
	}
	mode := "кеш"
	if fresh {
		mode = "обновлено"
	}
	text := fmt.Sprintf(
		"💾 <b>Место пользователей</b>\n<code>━━━━━━━━━━━━━━━━━━━━</code>\n\n"+
			"Режим: <b>%s</b>\nПроверено: <b>%d</b>\nОшибок: <b>%d</b>\nБез WebDAV-пароля: <b>%d</b>\nВсего занято: <b>%s</b>\nИзвестная квота: <b>%s</b>\n\n",
		mode, checked, failed, skipped, formatBytes(totalUsed), quotaLine,
	)
	if len(rows) == 0 {
		return text + "Пока нет одобренных пользователей с данными облака."
	}
	limit := 20
	if len(rows) < limit {
		limit = len(rows)
	}
	text += fmt.Sprintf("<b>Пользователи, топ %d по заполненности</b>\n", limit)
	for i := 0; i < limit; i++ {
		row := rows[i]
		name := displayName(&row.User)
		if row.Err != nil {
			text += fmt.Sprintf("%d. %s · ошибка: <code>%s</code>\n", i+1, name, esc(shortText(row.Err.Error(), 80)))
			continue
		}
		if row.Available >= 0 {
			total := row.Used + row.Available
			text += fmt.Sprintf("%d. %s · <b>%s</b> / <b>%s</b> · свободно <b>%s</b> · %.1f%%\n", i+1, name, formatBytes(row.Used), formatBytes(total), formatBytes(row.Available), row.Percent)
			continue
		}
		text += fmt.Sprintf("%d. %s · занято <b>%s</b> · свободно <b>%s</b>\n", i+1, name, formatBytes(row.Used), formatBytes(row.Available))
	}
	return text
}

func (a *App) infoAdminText() string {
	raw := shortText(a.content.messageRaw("info"), 1200)
	return "<b>ℹ️ Инфо-раздел</b>\n<code>━━━━━━━━━━━━━━━━━━━━</code>\n\n" +
		"Пользователь видит этот раздел по кнопке <b>" + esc(a.content.Button("info_ru")) + "</b> / <b>" + esc(a.content.Button("info_en")) + "</b> в своем кабинете.\n\n" +
		"Фото: <b>" + mapBool(a.content.Photo("info") != "", "задано", "нет") + "</b>\n\n" +
		"Текущий текст:\n<code>" + esc(raw) + "</code>"
}

func shortText(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "\n..."
}

func (a *App) promoListText() string {
	promos, err := a.db.ListPromos(20)
	if err != nil {
		return "🎟 Не удалось получить промокоды: <code>" + esc(err.Error()) + "</code>"
	}
	text := "🎟 <b>Промокоды</b>\n<code>━━━━━━━━━━━━━━━━━━━━</code>\n\n"
	if len(promos) == 0 {
		return text + "Пока пусто.\n\nФормат создания: <code>CODE 10 30 100</code>\nгде 10=GB, 30=дней премиума, 100=макс. использований."
	}
	for _, promo := range promos {
		text += fmt.Sprintf("<code>%s</code> · %dGB · %d дней · %d/%d · %s\n", esc(promo.Code), promo.QuotaGB, promo.PremiumDays, promo.UsedCount, promo.MaxUses, mapBool(promo.IsActive == 1, "активен", "выкл"))
	}
	return text
}

func parsePromoCreate(text string) (string, int, int, int, *string, error) {
	parts := strings.Fields(text)
	if len(parts) < 4 {
		return "", 0, 0, 0, nil, fmt.Errorf("format: CODE GB PREMIUM_DAYS MAX_USES [YYYY-MM-DD]")
	}
	quota, err1 := strconv.Atoi(parts[1])
	days, err2 := strconv.Atoi(parts[2])
	uses, err3 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil || err3 != nil || quota < 0 || days < 0 || uses < 0 {
		return "", 0, 0, 0, nil, fmt.Errorf("GB, days and uses must be numbers")
	}
	var expires *string
	if len(parts) >= 5 {
		t, err := time.Parse("2006-01-02", parts[4])
		if err != nil {
			return "", 0, 0, 0, nil, err
		}
		value := t.UTC().Format(time.RFC3339)
		expires = &value
	}
	return strings.ToUpper(parts[0]), quota, days, uses, expires, nil
}

func formattingText() string {
	return "🧾 <b>Форматирование текста</b>\n<code>━━━━━━━━━━━━━━━━━━━━</code>\n\n" +
		"Можно использовать HTML Telegram:\n" +
		"<code>&lt;b&gt;жирный&lt;/b&gt;</code>\n" +
		"<code>&lt;i&gt;курсив&lt;/i&gt;</code>\n" +
		"<code>&lt;u&gt;подчеркнутый&lt;/u&gt;</code>\n" +
		"<code>&lt;s&gt;зачеркнутый&lt;/s&gt;</code>\n" +
		"<code>&lt;code&gt;код&lt;/code&gt;</code>\n" +
		"<code>&lt;a href=\"https://example.com\"&gt;ссылка&lt;/a&gt;</code>\n\n" +
		"Custom emoji можно вставлять прямо в сообщение. Бот сохранит их как <code>&lt;tg-emoji emoji-id=\"...\"&gt;</code>.\n\n" +
		"Доступные плейсхолдеры зависят от сообщения: <code>{login}</code>, <code>{password}</code>, <code>{quota_gb}</code>, <code>{storage}</code>, <code>{cloud_url}</code>, <code>{support_contacts}</code>."
}

func (a *App) saveContentPhoto(msg *Message, st State) {
	if len(msg.Photo) == 0 {
		_, _ = a.tg.SendMessage(msg.Chat.ID, "Отправьте именно фото. Документ как фото Telegram не подставит в сообщение.", contentEditKeyboard("message", st.Event))
		return
	}
	best := msg.Photo[0]
	for _, photo := range msg.Photo {
		if photo.FileSize > best.FileSize {
			best = photo
		}
	}
	if err := a.content.SetPhoto(st.Event, best.FileID); err != nil {
		_, _ = a.tg.SendMessage(msg.Chat.ID, "Не удалось сохранить фото: <code>"+esc(err.Error())+"</code>", contentKeyboard())
		return
	}
	a.states.Clear(msg.From.ID)
	_, _ = a.tg.SendMessage(msg.Chat.ID, "🖼 Фото сохранено для сообщения <code>"+esc(st.Event)+"</code>.", contentEditKeyboard("message", st.Event))
}

func (a *App) applyPromoMessage(msg *Message) {
	a.states.Clear(msg.From.ID)
	user, err := a.db.GetUser(msg.From.ID)
	if err != nil || user == nil || user.Status != "approved" || user.IsDisabled == 1 {
		_, _ = a.tg.SendMessage(msg.Chat.ID, "Доступ не активен.", nil)
		return
	}
	oldQuota := user.QuotaGB
	promo, err := a.db.ApplyPromo(msg.Text, user)
	if err != nil {
		_, _ = a.tg.SendMessage(msg.Chat.ID, "🎟 Не удалось применить промокод: <code>"+esc(err.Error())+"</code>", a.accountKeyboard(langOf(user)))
		return
	}
	if promo.QuotaGB > 0 && user.NCUserID != nil {
		if err := a.nc.SetQuota(*user.NCUserID, oldQuota+promo.QuotaGB); err != nil {
			_, _ = a.tg.SendMessage(msg.Chat.ID, "⚠️ Промокод записан в базу, но квота облака не обновилась: <code>"+esc(err.Error())+"</code>", a.accountKeyboard(langOf(user)))
			return
		}
		a.quota.Delete(*user.NCUserID)
	}
	updated, _ := a.db.GetUser(msg.From.ID)
	text := fmt.Sprintf("🎟 <b>Промокод применен</b>\n\nДобавлено места: <b>%d GB</b>\nПремиум: <b>%d дней</b>", promo.QuotaGB, promo.PremiumDays)
	_, _ = a.tg.SendMessage(msg.Chat.ID, text, a.accountKeyboard(langOf(updated)))
}

func (a *App) createPromoMessage(msg *Message) {
	code, quota, days, uses, expires, err := parsePromoCreate(msg.Text)
	if err != nil {
		_, _ = a.tg.SendMessage(msg.Chat.ID, "Формат: <code>CODE GB PREMIUM_DAYS MAX_USES [YYYY-MM-DD]</code>\nОшибка: <code>"+esc(err.Error())+"</code>", promosKeyboard())
		return
	}
	if err := a.db.CreatePromo(code, quota, days, uses, expires); err != nil {
		_, _ = a.tg.SendMessage(msg.Chat.ID, "Не удалось создать промокод: <code>"+esc(err.Error())+"</code>", promosKeyboard())
		return
	}
	a.states.Clear(msg.From.ID)
	_, _ = a.tg.SendMessage(msg.Chat.ID, "🎟 Промокод <code>"+esc(code)+"</code> сохранен.\n\n"+a.promoListText(), promosKeyboard())
}

func (a *App) saveSettingMessage(msg *Message, st State) {
	value, err := a.parseSettingInput(st.Event, msg.Text)
	if err != nil {
		_, _ = a.tg.SendMessage(msg.Chat.ID, "Не удалось сохранить настройку: <code>"+esc(err.Error())+"</code>", commerceKeyboard())
		return
	}
	if err := a.db.SetSetting(st.Event, value); err != nil {
		_, _ = a.tg.SendMessage(msg.Chat.ID, "Ошибка базы: <code>"+esc(err.Error())+"</code>", commerceKeyboard())
		return
	}
	a.states.Clear(msg.From.ID)
	_, _ = a.tg.SendMessage(msg.Chat.ID, "✅ Настройка <code>"+esc(st.Event)+"</code> = <code>"+esc(value)+"</code>\n\n"+a.commerceText(), commerceKeyboard())
}
