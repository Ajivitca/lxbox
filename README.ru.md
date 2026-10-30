# LxBox для роутеров Keenetic (Entware)

<p align="center">
  <img src="https://raw.githubusercontent.com/Leadaxe/LxBox/main/docs/badges/get-it-on-github.png" alt="LxBox Logo" height="70"/>
</p>

<p align="center">
  <b>Порт ядра LxBox (<a href="https://github.com/Leadaxe/sing-box-lx">sing-box-lx</a>) с встроенным веб-интерфейсом MetaCubeXD и CLI-менеджером для роутеров Keenetic на базе Entware</b>
</p>

<p align="center">
  <a href="https://github.com/Ajivitca/lxbox/releases"><img src="https://img.shields.io/github/v/release/Ajivitca/lxbox?color=blue&label=%D0%A0%D0%B5%D0%BB%D0%B8%D0%B7" alt="Release"/></a>
  <a href="https://github.com/Ajivitca/lxbox/blob/main/LICENSE"><img src="https://img.shields.io/badge/%D0%9B%D0%B8%D1%86%D0%B5%D0%BD%D0%B7%D0%B8%D1%8F-GPLv3-green.svg" alt="License"/></a>
  <a href="https://entware.net/"><img src="https://img.shields.io/badge/Keenetic-Entware-orange" alt="Entware"/></a>
  <a href="#-совместимость-с-роутерами-keenetic"><img src="https://img.shields.io/badge/%D0%90%D1%80%D1%85%D0%B8%D1%82%D0%B5%D0%BA%D1%82%D1%83%D1%80%D0%B0-mipsle%20%7C%20arm64%20%7C%20armv7-purple" alt="Arch"/></a>
</p>

---

## 📖 О проекте

Приложение **[LxBox](https://github.com/Leadaxe/LxBox)** завоевало популярность на Android благодаря всеядности (поддержка зашифрованных строк Happ, VLESS Reality, XHTTP, AmneziaWG 2.0/3.x, MASQUE, Post-Quantum VLESS) и удобному интерфейсу.

Данный проект переносит эти возможности непосредственно на **роутеры Keenetic**:
1. **Ядро `sing-box-lx`:** Используется модифицированное ядро [Leadaxe/sing-box-lx](https://github.com/Leadaxe/sing-box-lx), в котором собраны все современные расширения (AmneziaWG, нативный транспорт XHTTP из Xray, постквантовая криптография `mlkem768x25519plus`, Cloudflare WARP over MASQUE).
2. **Встроенный веб-интерфейс MetaCubeXD:** Полноценный Web Dashboard (доступен по адресу `http://192.168.1.1:9090/ui`) для переключения серверов, замера пинга и мониторинга трафика.
3. **Парсер ссылок (`lxbox import`):** Позволяет прямо в консоли роутера скормить зашифрованную строку Happ (`happ://crypt...`), стандартную ссылку `vless://`, `hysteria2://`, `awg://` или ссылку на подписку.
4. **Совместимость с AWG Manager:** Бесшовная работа рядом с [hoaxisr/awg-manager](https://github.com/hoaxisr/awg-manager) без конфликтов портов и маршрутизации.

---

## 🖥 Совместимость с роутерами Keenetic

Инсталлятор автоматически определяет процессор и архитектуру устройства:

| Модель Keenetic | Процессор / Архитектура | Пакетная база Entware | Статус |
|---|---|---|---|
| **Hopper (KN-3810)** | EcoNet EN7528DU (MIPS 1004Kc, 900MHz, little-endian) | `mipselsf-k3.4` (`mipsle`) |  **Протестировано** |
| **Hopper DSL (KN-3610)** | EcoNet MIPS (little-endian) | `mipselsf-k3.4` (`mipsle`) |  Поддерживается |
| **Hopper SE (KN-3811)** | MediaTek MT7981B (Cortex-A53, 1.3GHz) | `aarch64-k3.10` (`arm64`) |  Поддерживается |
| **Titan (KN-1811)** | MediaTek MT7986A (Cortex-A53, 2.0GHz) | `aarch64-k3.10` (`arm64`) |  Поддерживается |
| **Hero (KN-1012)** | MediaTek MT7981B (Cortex-A53, 1.3GHz) | `aarch64-k3.10` (`arm64`) |  Поддерживается |
| **Speedster, Viva, Giant, Hero 4G** (KN-3010, KN-1910, KN-2610, KN-2310) | MediaTek MT7621AT (MIPS 1004Kc, 880MHz) | `mipsel-3.4` / `mipselsf-k3.4` |  Поддерживается |
| **Hero (KN-1010), Titan (KN-1810)** | MediaTek MT7621A | `mipsel-3.4` |  Поддерживается |

> [!NOTE]
> Все бинарники собраны статически (`CGO_ENABLED=0`), не зависят от внешних библиотек и сжаты UPX, благодаря чему занимают минимум места на накопителе Entware.

---

## 🚀 Установка

### Предварительные требования:
На роутере должен быть установлен накопитель с **Entware** (USB-флешка, внешний диск или встроенный накопитель).  
*Проверить можно наличием команды `opkg` по SSH.*

### Установка в одну команду:
Подключитесь к роутеру по SSH (например, через PuTTY, Termius или терминал macOS/Linux: `ssh root@192.168.1.1 -p 222`) и выполните:

```sh
curl -sSL https://raw.githubusercontent.com/Ajivitca/lxbox/main/install.sh | sh
```

*(Если `curl` не установлен в вашей системе, воспользуйтесь встроенным `wget`)*:
```sh
wget -O - https://raw.githubusercontent.com/Ajivitca/lxbox/main/install.sh | sh
```

### Что делает инсталлятор:
1. Автоматически проверяет среду Entware (`/opt`).
2. Распознаёт архитектуру роутера (`mipsle`, `arm64`, `armv7`).
3. Загружает оптимизированные бинарники ядра `sing-box` и утилиты `lxbox-helper`.
4. Разворачивает веб-панель **MetaCubeXD** в `/opt/etc/lxbox/ui`.
5. Устанавливает базовый шаблон конфигурации `/opt/etc/lxbox/config.json` (если файла нет; существующий конфиг сохраняется).
6. Регистрирует службу автозапуска `/opt/etc/init.d/S99lxbox` и запускает её.

---

## 📥 Как скормить строку от Happ / VLESS / Подписку

На Android в приложении LxBox вы просто вставляете ссылку — на роутере это делается через CLI-команду `lxbox import`.

### Примеры использования:

```sh
# 1. Зашифрованная строка Happ (автоматическая RSA-расшифровка)
lxbox import 'happ://crypt3/...'

# 2. Ссылка VLESS Reality или XHTTP
lxbox import 'vless://b831381d-6324-4d53-ad4f-8cda48b30811@1.2.3.4:443?security=reality&sni=yahoo.com&pbk=abcdef123456&sid=1234&type=tcp#MyNode'

# 3. Ссылка на подписку любого VPN-провайдера (Base64 / JSON)
lxbox import 'https://sub.provider.com/api/v1/client/subscribe?token=secret123'

# 4. Протокол Hysteria 2
lxbox import 'hysteria2://password@server.com:443?sni=server.com&insecure=1#FastHy2'

# 5. AmneziaWG / WireGuard
lxbox import 'awg://...'
```

### Как это работает под капотом:
1. Утилита `lxbox-helper` распознаёт формат строки.
2. Для ссылок `happ://crypt...` производится расшифровка RSA PKCS#1 v1.5 встроенными ключами клиента Happ.
3. Если получен URL подписки — утилита скачивает его, декодирует Base64 и извлекает все ноды.
4. Ссылка разбирается на параметры: `UUID`, `SNI`, `PBK` (Reality public key), `SID` (short id), `Transport`, `Flow`.
5. Нода добавляется в секцию `outbounds` файла `/opt/etc/lxbox/config.json` и привязывается к селектору `select`.
6. Производится проверка синтаксиса через `sing-box check`.
7. Служба `S99lxbox` автоматически перезапускается.

---

## 🌐 Веб-интерфейсы LxBox

В LxBox для Keenetic встроены два взаимодополняющих веб-интерфейса:

### 1. Веб-центр импорта и управления (`:9095`)
👉 **`http://192.168.1.1:9095`**
* **Импорт в браузере:** Вставьте зашифрованную строку Happ (`happ://crypt...`), VLESS Reality, Hysteria2, WireGuard или URL подписки со смартфона или ПК без открытия SSH-консоли!
* **Обновление подписок:** Кнопка обновления текущей подписки в один клик.
* **Управление интерфейсом KeeneticOS (`Proxy0`):** Включение/отключение системного интерфейса для HydraRoute Neo прямо из браузера.
* **Быстрое управление службой:** Перезапуск, проверка конфигурации, остановка и запуск.
* **Быстрый переход:** Кнопки прямого перехода в MetaCubeXD и AWG Manager.

### 2. Панель MetaCubeXD (`:9090/ui`)
👉 **`http://192.168.1.1:9090/ui`**
* **Выбор активной ноды:** В один клик переключайте исходящий сервер в группе `select`.
* **Тестирование задержки (Ping):** Замер скорости отклика до каждого сервера.
* **Графики в реальном времени:** Мониторинг входящего и исходящего трафика (КБ/с, МБ/с).
* **Мониторинг сессий:** Список активных TCP/UDP соединений и DNS-запросов.

---

## 🤝 Совместная работа с AWG Manager и HydraRoute Neo

Частый вопрос: *«Можно ли добавить LxBox как плагин в AWG Manager или заменить им встроенный sing-box?»*

### ⚠️ Почему НЕ нужно заменять встроенный sing-box в AWG Manager:
1. **Жесткий пин версий:** AWG Manager строго привязан к конкретной сборке своего форка (`hoaxisr/amnezia-box`) с проверкой SHA256-хэша (`RequiredVersion = 1.15.0-alpha.13-awgm.35`). При любом обновлении AWG Manager просто перезапишет бинарник своим файлом.
2. **Отсутствие Happ:** Парсер AWG Manager (`internal/singbox/vlink`) не поддерживает расшифровку зашифрованных строк Happ (`happ://crypt...`) ключами RSA PKCS#1 v1.5, а также не имеет оптимизаций XHTTP и квантовой криптографии из `sing-box-lx`.
3. **Монолитный интерфейс:** В AWG Manager нет внешней системы плагинов — его фронтенд скомпилирован в единый исполняемый файл.

### ✅ Рекомендуемое решение: Интеграция через интерфейс `Proxy0` (Zero-Collision)
Вместо вмешательства в код AWG Manager мы используем штатный механизм сетевых интерфейсов **KeeneticOS** и **HydraRoute Neo**:

```
 ┌────────────────────────────────────────────────────────────────────────┐
 │                           KeeneticOS                                   │
 │                                                                        │
 │  ┌───────────────────────┐             ┌─────────────────────────────┐ │
 │  │      AWG Manager      │             │        LxBox Daemon         │ │
 │  │    (Port 2222: Web)   │             │   (Port 9095: Web Import)   │ │
 │  │   Туннели WireGuard   │             │   (Port 9090: MetaCubeXD)   │ │
 │  └───────────┬───────────┘             └──────────────┬──────────────┘ │
 │              │ nwg0 / awg0                            │ SOCKS5 :10808  │
 │              ▼                                        ▼                │
 │       [Интерфейс AWG]                      [Интерфейс Proxy0]          │
 │              │                                        │                │
 │              └───────────────────┬────────────────────┘                │
 │                                  ▼                                     │
 │                   HydraRoute Neo / Политики Keenetic                   │
 │                (Раздельная маршрутизация по доменам)                   │
 └────────────────────────────────────────────────────────────────────────┘
```

#### Как это настроить в 2 шага:

1. **Создайте системный интерфейс `Proxy0`:**  
   В веб-интерфейсе `http://192.168.1.1:9095` нажмите кнопку **«Включить Proxy0»** или выполните в SSH:
   ```sh
   lxbox iface add
   ```
   KeeneticOS зарегистрирует локальный интерфейс `Proxy0`, привязанный к SOCKS5-порту LxBox (`127.0.0.1:10808`).

2. **Используйте `Proxy0` в HydraRoute Neo или AWG Manager:**
   * **В AWG Manager / HydraRoute Neo:** Откройте списки маршрутизации (домены Discord, YouTube, блокировки) и выберите целью интерфейс **`Proxy0`** или политику с `Proxy0`.
   * **В веб-интерфейсе Keenetic («Приоритеты подключений»):** Интерфейс `Proxy0` (LxBox) отображается в списке интернет-подключений — его можно перетащить в любую политику или назначить конкретным домашним устройствам.

**Главное преимущество:**  
Вы можете обновлять AWG Manager, HydraRoute Neo и LxBox независимо друг от друга в любой момент — ничего не сломается и не перезапишется!

---

## 🛠 Консольное управление (`lxbox`)

В консоли роутера доступна единая утилита управления:

```sh
lxbox
```
Выводит статус, IP, PID службы, потребление памяти и активные порты.

| Команда | Описание |
|---|---|
| `lxbox import '<ссылка>'` | Импортировать Happ, VLESS, Hy2, AWG или подписку |
| `lxbox iface [add\|del]` | Включить или удалить системный интерфейс Proxy0 в KeeneticOS |
| `lxbox status` | Показать статус службы, PID, память, порты и Proxy0 |
| `lxbox web` | Показать прямые ссылки на Web-интерфейсы (:9095, :9090, :2222) |
| `lxbox test` | Экспресс-проверка доступности и задержки через прокси |
| `lxbox log` | Просмотр логов в реальном времени (`tail -f`) |
| `lxbox check` | Валидация синтаксиса конфигурации `config.json` |
| `lxbox restart` | Перезапустить службу LxBox |
| `lxbox stop` | Остановить службу |
| `lxbox start` | Запустить службу |
| `lxbox uninstall` | Запустить деинсталляцию |

---

## 🗑 Удаление

Если вы решите удалить LxBox, выполните команду:

```sh
curl -sSL https://raw.githubusercontent.com/Ajivitca/lxbox/main/uninstall.sh | sh
```
или прямо на роутере:
```sh
lxbox uninstall
```

**Деинсталлятор:**
* Корректно остановит процессы `sing-box`.
* Удалит службу автозапуска `/opt/etc/init.d/S99lxbox`.
* Удалит исполняемые файлы `/opt/sbin/sing-box`, `/opt/bin/lxbox`, `/opt/bin/lxbox-helper`.
* Создаст архив с резервной копией ваших настроек в `/opt/etc/lxbox_backup_*.tar.gz` и очистит каталог `/opt/etc/lxbox`.

---

## ⚙ Сборка из исходников

Если вы хотите собрать бинарники самостоятельно:

```sh
git clone https://github.com/Ajivitca/lxbox.git
cd lxbox

# Сборка под Keenetic Hopper (MIPSLE soft-float):
./build.sh mipsle

# Сборка под Keenetic Titan / Hero (ARM64):
./build.sh arm64

# Сборка под ARMv7:
./build.sh armv7
```

---

## 📄 Лицензия

Проект распространяется под свободной лицензией **GNU General Public License v3.0 (GPLv3)**.  
Ядро основано на [Leadaxe/sing-box-lx](https://github.com/Leadaxe/sing-box-lx) и [SagerNet/sing-box](https://github.com/SagerNet/sing-box).  
Веб-интерфейс [MetaCubeX/metacubexd](https://github.com/MetaCubeX/metacubexd).
