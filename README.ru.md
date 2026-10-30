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

## 🌐 Веб-интерфейс (Web Dashboard)

После установки откройте веб-панель в любом браузере:

👉 **`http://192.168.1.1:9090/ui`**  
*(замените `192.168.1.1` на локальный адрес вашего роутера при необходимости)*

### Возможности панели MetaCubeXD:
* **Выбор активной ноды:** В один клик переключайте исходящий сервер в группе `select`.
* **Тестирование задержки (Ping):** Замер скорости отклика до каждого сервера.
* **Графики в реальном времени:** Мониторинг входящего и исходящего трафика (КБ/с, МБ/с).
* **Мониторинг сессий:** Список активных TCP/UDP соединений и DNS-запросов.

---

## 🤝 Работа в связке с AWG Manager

Если на роутере уже работает [AWG Manager](https://github.com/hoaxisr/awg-manager) или вы планируете его поставить:

### 1. Отсутствие конфликтов портов
* **Веб-интерфейс AWG Manager:** `http://192.168.1.1:2222`
* **Веб-интерфейс LxBox:** `http://192.168.1.1:9090/ui`
* **Прокси-порты LxBox:**
  * Mixed (SOCKS5/HTTP): `7890`
  * SOCKS5: `10808`
  * Clash REST API: `9090`

### 2. Сценарии интеграции

#### Сценарий 1: Разделение по политикам KeeneticOS (Рекомендуется)
AWG Manager управляет туннелями AmneziaWG на уровне сетевых интерфейсов KeeneticOS (`nwg0`, `awg0`).  
В веб-интерфейсе роутера Keenetic (**«Сетевые правила» → «Приоритеты подключений»**):
* Устройства, требующие максимальной скорости (Smart TV, консоли, рабочие ПК), можно направить через интерфейс AWG Manager.
* Устройства или приложения, которым требуются VLESS / Reality / XHTTP для обхода жестких блокировок, настраиваются на прокси-порт LxBox (`192.168.1.1:7890`).

#### Сценарий 2: Цепочка (Detour через AWG Manager)
Можно направить весь исходящий трафик LxBox в туннель AmneziaWG, поднятый AWG Manager'ом:
В `/opt/etc/lxbox/config.json` в свойствах outbound укажите интерфейс:
```json
{
  "type": "vless",
  "tag": "MyNode",
  "bind_interface": "nwg0",
  ...
}
```

#### Сценарий 3: Нативный AmneziaWG прямо в LxBox
Ядро `sing-box-lx` скомпилировано с тегом `with_amneziawg`, поэтому вы можете скармливать конфигурации AmneziaWG прямо в LxBox без сторонних менеджеров.

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
| `lxbox status` | Показать статус службы, PID, память и слушающие порты |
| `lxbox web` | Показать прямую ссылку на Web Dashboard |
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
