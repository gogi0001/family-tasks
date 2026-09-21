# Family Tasks

Простой семейный трекер задач. Один Go-сервер, веб-клиент в браузере,
опциональная Android-обёртка через WebView и push-уведомления через
self-hosted [ntfy](https://ntfy.sh).

## Возможности

- Семьи с invite-кодами: создание, вход по коду, роли `owner` / `member`.
- Задачи: заголовок, описание, исполнитель, срок (`dueAt`), статус.
- Три статуса: `todo` → `in_progress` → `done`.
- Персональные цвета: аватарка в шапке, цветная точка в чипах, цвет активного
  статуса — того, кто его последним менял.
- Фильтры и сортировка: быстрые кнопки («Все», «Мои», «Просроченные»),
  подробная панель (статус, автор, исполнитель, диапазон дат создания).
- Push-уведомления через ntfy: о новой задаче, о смене статуса, за час до срока.
- Deep link из уведомления сразу в нужную задачу.
- Индикатор связи с сервером, мягкая деградация при offline.
- Хранение в SQLite с авто-миграциями.

## Состав репозитория

```
cmd/server/          точка входа сервера
internal/api/        HTTP-хендлеры и middleware
internal/config/     парсинг CLI-флагов и переменных окружения
internal/models/     доменные типы (Task, User, Family)
internal/notify/     отправка уведомлений в ntfy
internal/reminder/   фоновый раннер напоминаний о сроке
internal/storage/    SQLite + миграции (embed FS)
web/                 статический клиент (ES-модули, без сборщика)
android/             Android-обёртка (WebView, Kotlin)
```

## Аргументы командной строки

Все параметры можно задать флагом или переменной окружения. Приоритет:
**флаг → переменная окружения → дефолт**.

| Флаг                 | Env                 | По умолчанию    | Описание                                                                            |
| -------------------- | ------------------- | --------------- | ----------------------------------------------------------------------------------- |
| `-addr`              | `ADDR`              | `:8787`         | Адрес и порт HTTP-сервера                                                           |
| `-web-dir`           | `WEB_DIR`           | автопоиск       | Путь к папке `web/`. Если пусто — ищется в `./web`, `../../web`, рядом с бинарником |
| `-db`                | `DB_PATH`           | `data/tasks.db` | Путь к файлу SQLite                                                                 |
| `-ntfy-url`          | `NTFY_URL`          | пусто           | Базовый URL ntfy, например `http://127.0.0.1:7070`. Пусто = уведомления выключены   |
| `-ntfy-topic`        | `NTFY_TOPIC`        | пусто           | Топик ntfy, на который подписаны телефоны                                           |
| `-ntfy-click`        | `NTFY_CLICK`        | пусто           | Deep link для тапа по уведомлению, например `familytasks://open`                    |
| `-reminder-interval` | `REMINDER_INTERVAL` | `5m`            | Как часто проверять приближающиеся сроки                                            |
| `-reminder-window`   | `REMINDER_WINDOW`   | `1h`            | За сколько до срока отправлять напоминание                                          |
| `-log-level`         | `LOG_LEVEL`         | `info`          | `debug` \| `info` \| `warn` \| `error`                                              |
| `-log-format`        | `LOG_FORMAT`        | `text`          | `text` \| `json`                                                                    |

Примеры:

```bash
# минимальный запуск
./server

# с уведомлениями и кастомным портом
./server -addr :9000 -ntfy-url http://127.0.0.1:7070 -ntfy-topic family-tasks-home

# через переменные окружения
NTFY_URL=http://127.0.0.1:7070 NTFY_TOPIC=family-tasks-home ./server

# справка
./server --help
```

## Быстрый старт (разработка)

Требования: Go 1.22+.

```bash
git clone https://github.com/gogi0001/family-tasks.git
cd family-tasks

go build -o cmd/server/server ./cmd/server
./cmd/server/server
```

Сервер стартует на `http://localhost:8787/`. Откройте в браузере.

Файл БД появится в `data/tasks.db` автоматически, миграции применятся при
первом запуске.

## Развёртывание на сервере (Ubuntu)

### 1. Подготовка

```bash
sudo mkdir -p /opt
cd /opt
sudo git clone https://github.com/gogi0001/family-tasks.git
sudo chown -R $USER:$USER /opt/family-tasks
cd /opt/family-tasks
```

Требуется Go:

```bash
sudo apt install -y golang-go
# или свежая версия с https://go.dev/dl/
```

### 2. Сборка

```bash
cd /opt/family-tasks
go build -o cmd/server/server ./cmd/server
```

### 3. Systemd unit

Файл `/etc/systemd/system/family-tasks.service`:

```ini
[Unit]
Description=Family Tasks server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/opt/family-tasks
ExecStart=/opt/family-tasks/cmd/server/server -addr :8787 -web-dir /opt/family-tasks/web -db /opt/family-tasks/data/tasks.db -ntfy-url http://127.0.0.1:7070 -ntfy-topic family-tasks-home -ntfy-click familytasks://open -log-level info
Restart=on-failure
RestartSec=3s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=family-tasks

[Install]
WantedBy=multi-user.target
```

Запуск:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now family-tasks
sudo systemctl status family-tasks
sudo journalctl -u family-tasks -f
```

### 4. Обновление

```bash
cd /opt/family-tasks
sudo systemctl stop family-tasks
git pull
go build -o cmd/server/server ./cmd/server
sudo systemctl start family-tasks
```

Миграции применяются автоматически при старте. Перед обновлением в проде
делайте бэкап БД (см. ниже).

### 5. Бэкап

Скрипт `/etc/cron.daily/family-tasks-backup`:

```bash
#!/bin/sh
set -e
DB=/opt/family-tasks/data/tasks.db
BACKUP_DIR=/opt/family-tasks/data/backups
mkdir -p "$BACKUP_DIR"
STAMP=$(date +%Y%m%d)
sqlite3 "$DB" ".backup '$BACKUP_DIR/tasks-$STAMP.db'"
find "$BACKUP_DIR" -name 'tasks-*.db' -mtime +14 -delete
```

```bash
sudo chmod +x /etc/cron.daily/family-tasks-backup
sudo apt install -y sqlite3
```

## Сторонний сервис: ntfy

[ntfy](https://ntfy.sh) — self-hosted push-сервер. Одна бинарка на Go, никаких
зависимостей от Google/Firebase. Работает в локальной сети.

### Установка ntfy на тот же Ubuntu-сервер

```bash
sudo apt install -y ntfy
id _ntfy  # пользователь создаётся пакетом
```

### Конфигурация `/etc/ntfy/server.yml`

```yaml
base-url: "http://192.168.1.10:7070"
listen-http: ":7070"
cache-file: "/var/cache/ntfy/cache.db"
cache-duration: "12h"
log-level: info
```

Замените `192.168.1.10` на IP сервера в локальной сети. Порт `7070` —
произвольный, лишь бы не конфликтовал с нашим сервером (`8787`).

### Каталог кэша

```bash
sudo mkdir -p /var/cache/ntfy
sudo chown _ntfy:_ntfy /var/cache/ntfy
sudo chmod 750 /var/cache/ntfy
```

> **Внимание:** имя пользователя может быть `_ntfy` (Debian-пакет) или
> `ntfy`. Проверьте через `grep User= /usr/lib/systemd/system/ntfy.service`.

### Запуск

```bash
sudo systemctl enable --now ntfy
curl -s http://127.0.0.1:7070/v1/health
# {"healthy":true}
```

### Подключение телефонов

1. Установите **ntfy** из [F-Droid](https://f-droid.org/packages/io.heckel.ntfy/)
   или Google Play.
2. **Settings → Default server** → `http://192.168.1.10:7070`.
3. **+ → Subscribe to topic** → `family-tasks-home` (или другой ваш топик).
4. Проверка с сервера:

   ```bash
   curl -d "Проверка связи" http://192.168.1.10:7070/family-tasks-home
   ```

   На телефоне должно появиться уведомление.

### Файрвол (если используется UFW)

```bash
sudo ufw allow from 192.168.1.0/24 to any port 8787 proto tcp
sudo ufw allow from 192.168.1.0/24 to any port 7070 proto tcp
sudo ufw enable
```

## Android-обёртка

Каталог `android/` содержит WebView-приложение на Kotlin.

Сборка:

```bash
cd android
./gradlew assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

Или откройте `android/` в Android Studio и запустите **Run → app**.

### Настройка

1. Первый запуск: приложение откроет экран настроек.
2. **Адрес сервера**:
   - **Эмулятор:** `http://10.0.2.2:8787` — псевдоним хост-машины.
   - **Реальный телефон:** `http://<IP-сервера>:8787`. Узнать IP:
     `hostname -I` на сервере.
3. Сохранить. Клиент загрузится.
4. Сменить адрес можно в любой момент через **⋮ → Настройки** — перезагрузка
   происходит на лету, без перезапуска приложения.

### Deep link

Приложение регистрирует схему `familytasks://`. Когда пользователь тапает по
push-уведомлению, ntfy открывает ссылку `familytasks://open?task=<id>`, Android
передаёт её в приложение, а веб-клиент скроллит к нужной задаче и подсвечивает
её. Если приложение уже открыто — поднимется на передний план.

## HTTPS и доступ извне

Сейчас предполагается работа в локальной сети по HTTP. Если потребуется
доступ из мобильного интернета — поставьте reverse proxy (Caddy, Traefik) с
автоматическим HTTPS перед нашим сервером и ntfy, откройте порты наружу
(или используйте WireGuard-туннель до дома).

## Лицензия

Не указана.
