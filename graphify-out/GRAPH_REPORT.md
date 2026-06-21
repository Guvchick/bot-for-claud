# Graph Report - .  (2026-06-21)

## Corpus Check
- Corpus is ~25,784 words - fits in a single context window. You may not need a graph.

## Summary
- 557 nodes · 1244 edges · 29 communities (27 shown, 2 thin omitted)
- Extraction: 80% EXTRACTED · 20% INFERRED · 0% AMBIGUOUS · INFERRED: 252 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Admin Operations & Broadcast|Admin Operations & Broadcast]]
- [[_COMMUNITY_Database & Encryption Layer|Database & Encryption Layer]]
- [[_COMMUNITY_Telegram API Types|Telegram API Types]]
- [[_COMMUNITY_Payment Provider Config|Payment Provider Config]]
- [[_COMMUNITY_Content Store & Quota Cache|Content Store & Quota Cache]]
- [[_COMMUNITY_Infrastructure & Deployment|Infrastructure & Deployment]]
- [[_COMMUNITY_Bot Event Loop & Routing|Bot Event Loop & Routing]]
- [[_COMMUNITY_CryptoBot & Cloud Clients|CryptoBot & Cloud Clients]]
- [[_COMMUNITY_Admin Extras & Promo Codes|Admin Extras & Promo Codes]]
- [[_COMMUNITY_Data Models & Bot Stats|Data Models & Bot Stats]]
- [[_COMMUNITY_Account & External Payments|Account & External Payments]]
- [[_COMMUNITY_File Upload & Storage Text|File Upload & Storage Text]]
- [[_COMMUNITY_Redis State Store|Redis State Store]]
- [[_COMMUNITY_Background Jobs & Loops|Background Jobs & Loops]]
- [[_COMMUNITY_Upload Batch Management|Upload Batch Management]]
- [[_COMMUNITY_Nextcloud API Client|Nextcloud API Client]]
- [[_COMMUNITY_Payment Webhooks & Fulfillment|Payment Webhooks & Fulfillment]]
- [[_COMMUNITY_Priority Upload Queue|Priority Upload Queue]]
- [[_COMMUNITY_Sticker Store|Sticker Store]]
- [[_COMMUNITY_Notifications & Error Recovery|Notifications & Error Recovery]]
- [[_COMMUNITY_Content Rendering|Content Rendering]]
- [[_COMMUNITY_Module Root|Module Root]]

## God Nodes (most connected - your core abstractions)
1. `esc()` - 51 edges
2. `DB` - 41 edges
3. `App` - 35 edges
4. `Telegram` - 35 edges
5. `now()` - 34 edges
6. `keyboard()` - 33 edges
7. `InlineKeyboardMarkup` - 32 edges
8. `App` - 20 edges
9. `Context` - 18 edges
10. `CallbackQuery` - 18 edges

## Surprising Connections (you probably didn't know these)
- `loadConfig()` --calls--> `parseAdminIDs()`  [INFERRED]
  botgo/config.go → botgo/utils.go
- `telegram-nextcloud-bot Docker Service` --implements--> `Telegram Nextcloud Bot`  [INFERRED]
  docker-compose.yml → README.md
- `postgres Docker Service` --implements--> `PostgreSQL Storage`  [INFERRED]
  docker-compose.yml → README.md
- `redis Docker Service` --implements--> `Redis Temporary State`  [INFERRED]
  docker-compose.yml → README.md
- `telegram-bot-api Docker Service (tdlight)` --implements--> `Local Telegram Bot API (tdlight)`  [INFERRED]
  docker-compose.yml → README.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Payment -> Premium -> Webhook Grant Flow** — readme_payment_providers, readme_webhook_server, readme_premium_feature, readme_file_upload_queue [INFERRED 0.85]
- **Core Bot Infrastructure Services** — docker_compose_bot_service, docker_compose_postgres_service, docker_compose_redis_service [EXTRACTED 1.00]
- **File Upload via WebDAV requiring stored Nextcloud credentials** — readme_file_upload_queue, readme_webdav_upload, readme_password_encryption, readme_nextcloud_integration [EXTRACTED 0.95]

## Communities (29 total, 2 thin omitted)

### Community 0 - "Admin Operations & Broadcast"
Cohesion: 0.09
Nodes (26): shortText(), App, CallbackQuery, InlineKeyboardMarkup, Message, State, User, Message (+18 more)

### Community 1 - "Database & Encryption Layer"
Cohesion: 0.10
Nodes (14): AEAD, DB, boolToInt(), Config, Payment, User, NewDB(), now() (+6 more)

### Community 2 - "Telegram API Types"
Cohesion: 0.07
Nodes (30): BotFile, CallbackQuery, Chat, InlineKeyboardButton, InlineKeyboardMarkup, LabeledPrice, Message, MessageEntity (+22 more)

### Community 3 - "Payment Provider Config"
Cohesion: 0.12
Nodes (34): backAdminKeyboard(), backupKeyboard(), buyStorageKeyboard(), commerceKeyboard(), contentKeyboard(), contentListKeyboard(), deleteConfirmKeyboard(), donateKeyboard() (+26 more)

### Community 4 - "Content Store & Quota Cache"
Cohesion: 0.09
Nodes (21): contentDefault(), contentKeyExists(), contentTitle(), Message, RWMutex, messageHTMLWithCustomEmoji(), NewContentStore(), renderTemplate() (+13 more)

### Community 5 - "Infrastructure & Deployment"
Cohesion: 0.11
Nodes (30): telegram-nextcloud-bot Docker Service, postgres_data Docker Volume, postgres Docker Service, redis_data Docker Volume, redis Docker Service, telegram_bot_api_data Docker Volume, telegram-bot-api Docker Service (tdlight), Admin Panel (+22 more)

### Community 6 - "Bot Event Loop & Routing"
Cohesion: 0.12
Nodes (20): App, Message, main(), configureLogging(), env(), envBool(), envInt(), envIntList() (+12 more)

### Community 7 - "CryptoBot & Cloud Clients"
Cohesion: 0.12
Nodes (12): firstMapString(), Client, CryptoBotPay, Heleket, Pally, Client, Values, marshalJSONNoEscape() (+4 more)

### Community 8 - "Admin Extras & Promo Codes"
Cohesion: 0.14
Nodes (7): formattingText(), App, Message, State, User, parsePromoCreate(), storageMetricRow

### Community 9 - "Data Models & Bot Stats"
Cohesion: 0.10
Nodes (21): BotStats, App, Config, DB, StickerStore, Payment, PromoCode, State (+13 more)

### Community 10 - "Account & External Payments"
Cohesion: 0.19
Nodes (7): App, CallbackQuery, Message, Payment, accountBackKeyboard(), containsInt(), langOf()

### Community 11 - "File Upload & Storage Text"
Cohesion: 0.18
Nodes (9): App, Message, UploadJob, User, firstNonEmpty(), User, isPremium(), premiumUntilText() (+1 more)

### Community 12 - "Redis State Store"
Cohesion: 0.17
Nodes (12): Duration, Mutex, Reader, State, Writer, NewRedisClient(), NewStateStore(), readRedisReply() (+4 more)

### Community 13 - "Background Jobs & Loops"
Cohesion: 0.21
Nodes (11): createDatabaseBackup(), createPublicJSONBackup(), Config, DB, listBackups(), pruneBackups(), restoreDatabaseBackup(), sortByModTime() (+3 more)

### Community 14 - "Upload Batch Management"
Cohesion: 0.23
Nodes (8): App, Mutex, Time, UploadJob, NewUploadBatchManager(), UploadBatch, UploadBatchItem, UploadBatchManager

### Community 15 - "Nextcloud API Client"
Cohesion: 0.23
Nodes (4): Reader, Values, Nextcloud, cleanFilename()

### Community 16 - "Payment Webhooks & Fulfillment"
Cohesion: 0.29
Nodes (7): App, HandlerFunc, paymentLanding(), plategaInt(), plategaString(), Request, ResponseWriter

### Community 17 - "Priority Upload Queue"
Cohesion: 0.23
Nodes (6): Mutex, NewUploadQueue(), UploadHeap, UploadJob, UploadQueue, Cond

### Community 18 - "Sticker Store"
Cohesion: 0.23
Nodes (6): App, RWMutex, NewStickerStore(), StickerStore, stickerStoreFile, StickerValue

### Community 20 - "Content Rendering"
Cohesion: 0.50
Nodes (3): App, InlineKeyboardMarkup, Message

## Knowledge Gaps
- **83 isolated node(s):** `Update`, `Values`, `Reader`, `RWMutex`, `contentStoreFile` (+78 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **2 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `now()` connect `Database & Encryption Layer` to `Admin Operations & Broadcast`, `Content Store & Quota Cache`, `Bot Event Loop & Routing`, `CryptoBot & Cloud Clients`, `Account & External Payments`, `File Upload & Storage Text`, `Background Jobs & Loops`, `Upload Batch Management`, `Payment Webhooks & Fulfillment`, `Sticker Store`, `Notifications & Error Recovery`?**
  _High betweenness centrality (0.217) - this node is a cross-community bridge._
- **Why does `esc()` connect `Admin Operations & Broadcast` to `Payment Provider Config`, `Bot Event Loop & Routing`, `Admin Extras & Promo Codes`, `Account & External Payments`, `File Upload & Storage Text`, `Background Jobs & Loops`, `Upload Batch Management`, `Payment Webhooks & Fulfillment`, `Notifications & Error Recovery`?**
  _High betweenness centrality (0.178) - this node is a cross-community bridge._
- **Why does `main()` connect `Bot Event Loop & Routing` to `Database & Encryption Layer`, `Content Store & Quota Cache`, `Redis State Store`, `Upload Batch Management`, `Priority Upload Queue`, `Sticker Store`?**
  _High betweenness centrality (0.116) - this node is a cross-community bridge._
- **Are the 49 inferred relationships involving `esc()` (e.g. with `.accountSupport()` and `.accountText()`) actually correct?**
  _`esc()` has 49 INFERRED edges - model-reasoned connections that need verification._
- **Are the 17 inferred relationships involving `now()` (e.g. with `.fulfillPayment()` and `.handleSuccessfulPayment()`) actually correct?**
  _`now()` has 17 INFERRED edges - model-reasoned connections that need verification._
- **What connects `Update`, `Values`, `Reader` to the rest of the system?**
  _83 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Admin Operations & Broadcast` be split into smaller, more focused modules?**
  _Cohesion score 0.08911325724319578 - nodes in this community are weakly interconnected._