# Banners

Drop banner images for the bot menus into this folder, then point the matching
`BANNER_*` variables in `.env` at the filenames.

Each banner is shown as an image above its menu. The bot uploads every image to
Telegram once and reuses the returned `file_id` afterwards (cached in
`BANNER_CACHE_FILE`, default `data/banners.json`), so changing an image and
restarting the bot re-uploads it automatically.

## Menu → variable

| Menu                                  | Variable               |
| ------------------------------------- | ---------------------- |
| Main menu / personal cabinet          | `BANNER_MENU`          |
| Access opened (new account)           | `BANNER_APPROVED`      |
| Support                               | `BANNER_SUPPORT`       |
| Donate                                | `BANNER_DONATE`        |
| Info                                  | `BANNER_INFO`          |
| Notifications                         | `BANNER_NOTIFICATIONS` |
| Admin panel                           | `BANNER_ADMIN`         |
| Language                              | `BANNER_LANGUAGE`      |

## Example

```
BANNER_DIR=banners
BANNER_MENU=main.jpg
BANNER_DONATE=donate.png
BANNER_ADMIN=admin.jpg
```

Paths may be relative to `BANNER_DIR` (as above) or absolute. Supported formats are
the usual Telegram photo formats (JPG, PNG). Keep them within Telegram's photo
limits (max 10 MB, sensible dimensions).

> The bot warms banners at startup by sending each one once to the log group
> (`LOG_GROUP_ID`) or, if unset, to the first admin — that first send is what
> captures the reusable `file_id`.
