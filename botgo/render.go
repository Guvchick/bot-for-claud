package main

import "log"

func (a *App) sendContent(chatID int64, key string, vars map[string]string, markup *InlineKeyboardMarkup) (*Message, error) {
	text := a.content.Message(key, vars)
	photo := a.menuPhotoID(key)
	if photo != "" {
		if len([]rune(text)) > 1024 {
			if _, err := a.tg.SendPhoto(chatID, photo, "", nil); err != nil {
				return nil, err
			}
			return a.tg.SendMessage(chatID, text, markup)
		}
		return a.tg.SendPhoto(chatID, photo, text, markup)
	}
	return a.tg.SendMessage(chatID, text, markup)
}

// menuPhotoID resolves the image for a menu/content key: an admin-set content photo
// takes priority, otherwise the cached file_id of a configured banner is used.
func (a *App) menuPhotoID(key string) string {
	if id := a.content.Photo(key); id != "" {
		return id
	}
	if a.banners != nil {
		return a.banners.FileID(bannerKeyFor(key))
	}
	return ""
}

// present renders a screen, attaching a banner/content image when one exists. With
// an image it sends a fresh photo message (deleting the previous one on a callback);
// without an image it edits in place on a callback, or sends a new message.
func (a *App) present(cb *CallbackQuery, chatID int64, key, text string, markup *InlineKeyboardMarkup) {
	if photo := a.menuPhotoID(key); photo != "" {
		if len([]rune(text)) > 1024 {
			_, _ = a.tg.SendPhoto(chatID, photo, "", nil)
			_, _ = a.tg.SendMessage(chatID, text, markup)
		} else {
			_, _ = a.tg.SendPhoto(chatID, photo, text, markup)
		}
		a.deletePrev(cb, chatID)
		return
	}
	if cb != nil {
		a.edit(cb, text, markup)
		return
	}
	_, _ = a.tg.SendMessage(chatID, text, markup)
}

func (a *App) deletePrev(cb *CallbackQuery, chatID int64) {
	if cb != nil && cb.Message != nil {
		_ = a.tg.DeleteMessage(chatID, cb.Message.MessageID)
	}
}

// warmBanners uploads every configured banner once to capture its Telegram file_id,
// so user-facing menus can attach the banner by file_id without re-uploading. Once
// cached on disk, banners are only re-sent when their file changes.
func (a *App) warmBanners() {
	if a.banners == nil {
		return
	}
	keys := a.banners.Keys()
	if len(keys) == 0 {
		return
	}
	warmChat := a.cfg.LogGroupID
	if warmChat == 0 {
		for id := range a.cfg.AdminIDs {
			warmChat = id
			break
		}
	}
	if warmChat == 0 {
		log.Printf("banners configured but no LOG_GROUP_ID or admin to warm them; they load on first use")
		return
	}
	warmed := 0
	for _, key := range keys {
		if a.banners.FileID(key) != "" {
			continue
		}
		msg, err := a.tg.SendPhotoFile(warmChat, a.banners.Path(key), "🖼 Banner loaded: <code>"+esc(key)+"</code>")
		if err != nil {
			log.Printf("banner warm failed: key=%s err=%v", key, err)
			continue
		}
		if id := largestPhotoID(msg); id != "" {
			a.banners.Remember(key, id)
			warmed++
		}
	}
	if warmed > 0 {
		log.Printf("banners ready: %d loaded, %d configured", warmed, len(keys))
	}
}
