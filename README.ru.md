# LxBox для роутеров Keenetic (Entware)

<p align="center">
  <b>Порт ядра LxBox (<a href="https://github.com/Leadaxe/sing-box-lx">sing-box-lx</a>) с веб-интерфейсом MetaCubeXD и CLI-утилитой для роутеров Keenetic</b>
</p>

---

## ⚡ О проекте

**LxBox для Keenetic** — это готовое решение для роутеров Keenetic под управлением Entware, основанное на ядре **[Leadaxe/sing-box-lx](https://github.com/Leadaxe/sing-box-lx)** (форке sing-box, который используется в Android-клиенте [LxBox](https://github.com/Leadaxe/LxBox)).

### Возможности:
* **Поддержка современных протоколов:** VLESS + REALITY, XHTTP, AmneziaWG (AWG 1.x / 2.0 / 3.x), Hysteria 2, Cloudflare WARP over MASQUE, Trojan, Shadowsocks, WireGuard.
* **Импорт ссылок из Happ и других клиентов:** Утилита `lxbox import` «ест» строки Happ (`happ://crypt...`), стандартные ссылки `vless://`, `hysteria2://`, `awg://`, а также ссылки на подписки провайдеров (`https://...`).
* **Встроенный Web UI (MetaCubeXD):** Локальная панель управления доступна прямо в браузере по адресу `http://192.168.1.1:9090/ui` — мониторинг скорости, пинг узлов, переключение серверов в один клик.
* **Совместимость с Keenetic Hopper (KN-3810):** Скомпилировано под MIPS Little-Endian (`mipsle`, soft-float / `mipselsf-k3.4`) для процессора EcoNet EN7528DU / MT7621. Также поддержаны модели на `aarch64` (Titan KN-1811, Hero KN-1012, Hopper SE KN-3811) и `armv7`.
* **Работа в связке с AWG Manager:** Гармонично сосуществует с [hoaxisr/awg-manager](https://github.com/hoaxisr/awg-manager) без конфликтов портов и сетевых таблиц.

---

## 🚀 Быстрая установка

Подключитесь к роутеру Keenetic по SSH и выполните одну команду:

```sh
curl -sSL https://raw.githubusercontent.com/Ajivitca/lxbox/main/install.sh | sh
```

*(или через `wget`, если `curl` не установлен)*:
```sh
wget -O - https://raw.githubusercontent.com/Ajivitca/lxbox/main/install.sh | sh
```

Скрипт автоматически:
1. Определит архитектуру процессора вашего роутера (`mipsle` для Hopper KN-3810, `arm64` для Titan и т.д.).
2. Установит оптимизированный бинарник `sing-box` в `/opt/sbin/sing-box`.
3. Установит CLI-утилиту `/opt/bin/lxbox` и `/opt/bin/lxbox-helper`.
4. Развернёт веб-интерфейс MetaCubeXD в `/opt/etc/lxbox/ui`.
5. Создаст службу автозапуска `/opt/etc/init.d/S99lxbox` и запустит её.

---

## 📥 Как скормить строку от Happ / VLESS / Подписку

В консоли роутера достаточно выполнить:

```sh
# 1. Зашифрованная строка Happ
lxbox import 'happ://crypt3/...'

# 2. VLESS + Reality / XHTTP ссылка
lxbox import 'vless://uuid@host:443?security=reality&sni=example.com&pbk=...#MyNode'

# 3. Ссылка на подписку провайдера (Base64 / JSON)
lxbox import 'https://my-vpn-provider.com/api/v1/client/subscribe?token=...'

# 4. AmneziaWG / WireGuard
lxbox import 'awg://...'
```

После импорта утилита автоматически проверит конфигурацию и перезапустит службу.

---

## 🌐 Веб-интерфейс (Web UI)

Откройте в браузере любого устройства в домашней сети:

👉 **[http://192.168.1.1:9090/ui](http://192.168.1.1:9090/ui)**  
*(или замените `192.168.1.1` на локальный IP вашего роутера)*

В веб-панели можно:
- Видеть статус и переключать активные ноды.
- Замерять пинг (задержку) до каждого сервера.
- Наблюдать графики скорости входящего и исходящего трафика в реальном времени.
- Просматривать активные соединения.

---

## 🤝 Работа в связке с AWG Manager

Если на роутере уже установлен или планируется [AWG Manager](https://github.com/hoaxisr/awg-manager):

1. **Разделение портов (нет конфликтов):**
   - AWG Manager Web UI: `http://192.168.1.1:2222`
   - LxBox Web UI: `http://192.168.1.1:9090/ui`
   - Порты прокси LxBox: SOCKS5/HTTP `7890`, SOCKS5 `10808`

2. **Сценарии совместного использования:**
   - **Сценарий А (Политики маршрутизации KeeneticOS):**  
     AWG Manager поднимает туннели AmneziaWG (например, `nwg0`), а LxBox предоставляет SOCKS5/HTTP и TUN для протоколов VLESS/Reality/XHTTP. В веб-интерфейсе Keenetic («Сетевые правила» → «Приоритеты подключений») можно направить нужные устройства или домены в AWG, а заблокированные DPI ресурсы — через LxBox.
   - **Сценарий Б (Цепочка / Detour):**  
     Можно направить исходящий трафик sing-box через интерфейс AWG Manager. Для этого в `/opt/etc/lxbox/config.json` в outbound достаточно указать:
     ```json
     "bind_interface": "nwg0"
     ```
   - **Сценарий В (Прямой AmneziaWG в LxBox):**  
     Ядро `sing-box-lx` имеет нативную поддержку `type: "amneziawg"` прямо внутри ядра.

---

## 🛠 Управление через CLI (`lxbox`)

| Команда | Описание |
|---|---|
| `lxbox status` | Показать статус службы, PID, использование памяти и порты |
| `lxbox import '<ссылка>'` | Импортировать Happ, VLESS, Hy2 или подписку |
| `lxbox web` | Показать ссылку на Web Dashboard |
| `lxbox test` | Проверить работу и задержку через локальный прокси |
| `lxbox log` | Просмотр лога в реальном времени (`tail -f`) |
| `lxbox check` | Проверить синтаксис конфигурации |
| `lxbox restart` | Перезапустить службу LxBox |
| `lxbox stop` | Остановить службу |
| `lxbox start` | Запустить службу |
| `lxbox uninstall` | Полное удаление LxBox |

---

## 🗑 Удаление

Чтобы полностью удалить LxBox с роутера (с автоматическим сохранением резервной копии конфига):

```sh
curl -sSL https://raw.githubusercontent.com/Ajivitca/lxbox/main/uninstall.sh | sh
```
или прямо из CLI:
```sh
lxbox uninstall
```
