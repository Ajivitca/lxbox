package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	webMu sync.Mutex
)

// StatusResponse represents current system and proxy status
type StatusResponse struct {
	Running         bool     `json:"running"`
	Pid             int      `json:"pid"`
	Uptime          string   `json:"uptime,omitempty"`
	Version         string   `json:"version"`
	ConfigPath      string   `json:"config_path"`
	Outbounds       []string `json:"outbounds"`
	SubscriptionURL string   `json:"subscription_url,omitempty"`
	NDMSProxyActive bool     `json:"ndms_proxy_active"`
	MixedPort       int      `json:"mixed_port"`
	SocksPort       int      `json:"socks_port"`
	ClashPort       int      `json:"clash_port"`
	WebPort         int      `json:"web_port"`
}

type ImportRequest struct {
	Link string `json:"link"`
}

type ImportResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Count   int      `json:"count"`
	Nodes   []string `json:"nodes"`
}

type ActionRequest struct {
	Action string `json:"action"` // "restart", "stop", "start", "update_sub", "check"
}

type ActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func getSingboxPID() int {
	pidFile := "/opt/var/run/lxbox.pid"
	if data, err := os.ReadFile(pidFile); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			if err := exec.Command("kill", "-0", strconv.Itoa(pid)).Run(); err == nil {
				return pid
			}
		}
	}
	out, err := exec.Command("pidof", "sing-box").Output()
	if err == nil {
		parts := strings.Fields(string(out))
		if len(parts) > 0 {
			if pid, err := strconv.Atoi(parts[0]); err == nil {
				return pid
			}
		}
	}
	return 0
}

func checkNDMSProxy() bool {
	if _, err := exec.LookPath("ndmc"); err != nil {
		return false
	}
	out, err := exec.Command("ndmc", "-c", "show interface Proxy0").Output()
	if err != nil {
		return false
	}
	s := string(out)
	return strings.Contains(s, "Proxy0") && !strings.Contains(s, "unable to find")
}

func configureNDMSProxy(enable bool) error {
	if _, err := exec.LookPath("ndmc"); err != nil {
		return fmt.Errorf("ndmc utility not found (KeeneticOS CLI required)")
	}
	if enable {
		cmds := []string{
			"interface Proxy0",
			"interface Proxy0 description LxBox",
			"interface Proxy0 proxy proto socks5",
			"interface Proxy0 proxy upstream 127.0.0.1 10808",
			"interface Proxy0 proxy socks5-udp",
			"interface Proxy0 ip global auto",
			"interface Proxy0 up",
			"system configuration save",
		}
		for _, c := range cmds {
			if out, err := exec.Command("ndmc", "-c", c).CombinedOutput(); err != nil {
				return fmt.Errorf("ndmc command failed '%s': %s (%w)", c, string(out), err)
			}
		}
	} else {
		cmds := []string{
			"no interface Proxy0",
			"system configuration save",
		}
		for _, c := range cmds {
			exec.Command("ndmc", "-c", c).Run()
		}
	}
	return nil
}

func restartSingboxService() error {
	initScript := "/opt/etc/init.d/S99lxbox"
	if _, err := os.Stat(initScript); err == nil {
		return exec.Command(initScript, "restart").Run()
	}
	pid := getSingboxPID()
	if pid > 0 {
		exec.Command("kill", "-HUP", strconv.Itoa(pid)).Run()
	}
	return nil
}

func getActiveOutbounds(configPath string) []string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	obsRaw, ok := cfg["outbounds"].([]interface{})
	if !ok {
		return nil
	}
	var res []string
	for _, o := range obsRaw {
		if om, ok := o.(map[string]interface{}); ok {
			tag, _ := om["tag"].(string)
			tp, _ := om["type"].(string)
			if tag != "" && tp != "direct" && tp != "block" && tp != "dns" && tp != "selector" {
				srv, _ := om["server"].(string)
				port := om["server_port"]
				if srv != "" {
					res = append(res, fmt.Sprintf("%s (%s -> %s:%v)", tag, tp, srv, port))
				} else {
					res = append(res, fmt.Sprintf("%s (%s)", tag, tp))
				}
			}
		}
	}
	return res
}

func runWebServer(listenAddr string, configPath string) error {
	subFile := filepath.Join(filepath.Dir(configPath), "subscription.txt")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmlPage))
	})

	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pid := getSingboxPID()
		subURL := ""
		if data, err := os.ReadFile(subFile); err == nil {
			subURL = strings.TrimSpace(string(data))
		}
		st := StatusResponse{
			Running:         pid > 0,
			Pid:             pid,
			Version:         "1.14.4-lx.25",
			ConfigPath:      configPath,
			Outbounds:       getActiveOutbounds(configPath),
			SubscriptionURL: subURL,
			NDMSProxyActive: checkNDMSProxy(),
			MixedPort:       7890,
			SocksPort:       10808,
			ClashPort:       9090,
			WebPort:         9095,
		}
		json.NewEncoder(w).Encode(st)
	})

	http.HandleFunc("/api/import", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		webMu.Lock()
		defer webMu.Unlock()

		var req ImportRequest
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			json.NewEncoder(w).Encode(ImportResponse{Success: false, Message: "Failed to read request"})
			return
		}
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			json.NewEncoder(w).Encode(ImportResponse{Success: false, Message: "Invalid JSON payload"})
			return
		}
		link := strings.TrimSpace(req.Link)
		if link == "" {
			json.NewEncoder(w).Encode(ImportResponse{Success: false, Message: "Link is empty"})
			return
		}

		// Save subscription URL if remote HTTP
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			_ = os.WriteFile(subFile, []byte(link), 0644)
		}

		obs, err := parseLink(link)
		if err != nil {
			json.NewEncoder(w).Encode(ImportResponse{Success: false, Message: fmt.Sprintf("Parse error: %v", err)})
			return
		}

		if err := updateConfig(configPath, obs); err != nil {
			json.NewEncoder(w).Encode(ImportResponse{Success: false, Message: fmt.Sprintf("Config update error: %v", err)})
			return
		}

		// Validate with sing-box check
		if _, err := exec.LookPath("sing-box"); err == nil {
			cmd := exec.Command("sing-box", "check", "-c", configPath)
			if out, err := cmd.CombinedOutput(); err != nil {
				json.NewEncoder(w).Encode(ImportResponse{
					Success: false,
					Message: fmt.Sprintf("Config validation failed: %s", string(out)),
				})
				return
			}
		}

		// Restart sing-box
		_ = restartSingboxService()

		var nodeNames []string
		for _, o := range obs {
			tag, _ := o["tag"].(string)
			nodeNames = append(nodeNames, tag)
		}

		json.NewEncoder(w).Encode(ImportResponse{
			Success: true,
			Count:   len(obs),
			Nodes:   nodeNames,
			Message: fmt.Sprintf("Успешно импортировано узлов: %d. Сервис sing-box перезапущен.", len(obs)),
		})
	})

	http.HandleFunc("/api/action", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		webMu.Lock()
		defer webMu.Unlock()

		var req ActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: "Invalid JSON"})
			return
		}

		switch req.Action {
		case "restart":
			if err := restartSingboxService(); err != nil {
				json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: fmt.Sprintf("Restart failed: %v", err)})
				return
			}
			json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: "Сервис sing-box успешно перезапущен"})

		case "stop":
			initScript := "/opt/etc/init.d/S99lxbox"
			exec.Command(initScript, "stop").Run()
			json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: "Сервис sing-box остановлен"})

		case "start":
			initScript := "/opt/etc/init.d/S99lxbox"
			exec.Command(initScript, "start").Run()
			json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: "Сервис sing-box запущен"})

		case "update_sub":
			data, err := os.ReadFile(subFile)
			if err != nil || len(strings.TrimSpace(string(data))) == 0 {
				json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: "Сохраненный URL подписки не найден. Сначала импортируйте ссылку."})
				return
			}
			urlStr := strings.TrimSpace(string(data))
			obs, err := parseLink(urlStr)
			if err != nil {
				json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: fmt.Sprintf("Ошибка обновления подписки: %v", err)})
				return
			}
			if err := updateConfig(configPath, obs); err != nil {
				json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: fmt.Sprintf("Ошибка записи: %v", err)})
				return
			}
			_ = restartSingboxService()
			json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: fmt.Sprintf("Подписка обновлена! Загружено %d узлов.", len(obs))})

		case "check":
			if _, err := exec.LookPath("sing-box"); err != nil {
				json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: "sing-box binary not found on PATH"})
				return
			}
			out, err := exec.Command("sing-box", "check", "-c", configPath).CombinedOutput()
			if err != nil {
				json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: fmt.Sprintf("Ошибка проверки:\n%s", string(out))})
				return
			}
			json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: "Конфигурация sing-box корректна!"})

		default:
			json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: "Неизвестное действие"})
		}
	})

	http.HandleFunc("/api/interface", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		webMu.Lock()
		defer webMu.Unlock()

		var req ActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: "Invalid JSON"})
			return
		}

		if req.Action == "enable" {
			if err := configureNDMSProxy(true); err != nil {
				json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: fmt.Sprintf("Ошибка создания Proxy0: %v", err)})
				return
			}
			json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: "Интерфейс Proxy0 успешно создан и привязан к LxBox (127.0.0.1:10808)!"})
		} else if req.Action == "disable" {
			if err := configureNDMSProxy(false); err != nil {
				json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: fmt.Sprintf("Ошибка удаления Proxy0: %v", err)})
				return
			}
			json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: "Интерфейс Proxy0 удален из KeeneticOS"})
		} else {
			active := checkNDMSProxy()
			json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: fmt.Sprintf("Статус Proxy0: %v", active)})
		}
	})

	fmt.Printf("[+] LxBox Web Management Server listening on http://%s\n", listenAddr)
	server := &http.Server{
		Addr:         listenAddr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return server.ListenAndServe()
}

const htmlPage = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>LxBox for Keenetic — Management & Subscription</title>
  <style>
    :root {
      --bg: #090d16;
      --card-bg: #111827;
      --border: #1f2937;
      --text: #f3f4f6;
      --text-muted: #9ca3af;
      --primary: #06b6d4;
      --primary-hover: #0891b2;
      --accent: #8b5cf6;
      --success: #10b981;
      --danger: #ef4444;
      --warning: #f59e0b;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding: 20px;
    }
    .container { max-width: 960px; margin: 0 auto; }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding-bottom: 20px;
      border-bottom: 1px solid var(--border);
      margin-bottom: 24px;
      flex-wrap: wrap;
      gap: 12px;
    }
    .logo-group { display: flex; align-items: center; gap: 12px; }
    .logo-badge {
      background: linear-gradient(135deg, var(--primary), var(--accent));
      color: white;
      font-weight: 900;
      font-size: 20px;
      padding: 6px 14px;
      border-radius: 10px;
      box-shadow: 0 4px 12px rgba(6, 182, 212, 0.3);
    }
    h1 { font-size: 22px; font-weight: 700; }
    .subtitle { font-size: 13px; color: var(--text-muted); }
    .nav-links { display: flex; gap: 10px; flex-wrap: wrap; }
    .nav-btn {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 14px;
      background: #1e293b;
      color: var(--text);
      text-decoration: none;
      border-radius: 8px;
      font-size: 13px;
      font-weight: 500;
      border: 1px solid var(--border);
      transition: all 0.2s;
    }
    .nav-btn:hover { background: #334155; border-color: var(--primary); }
    .nav-btn.primary { background: rgba(6, 182, 212, 0.15); color: var(--primary); border-color: rgba(6, 182, 212, 0.4); }
    .nav-btn.primary:hover { background: rgba(6, 182, 212, 0.25); }

    .grid { display: grid; grid-template-columns: 1fr; gap: 20px; }
    @media(min-width: 768px) {
      .grid-2 { grid-template-columns: 2fr 1fr; }
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px;
      box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.2);
    }
    .card-title {
      font-size: 16px;
      font-weight: 600;
      margin-bottom: 14px;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 10px;
      border-radius: 20px;
      font-size: 12px;
      font-weight: 600;
    }
    .status-pill.running { background: rgba(16, 185, 129, 0.15); color: var(--success); }
    .status-pill.stopped { background: rgba(239, 68, 68, 0.15); color: var(--danger); }
    .status-dot { width: 8px; height: 8px; border-radius: 50%; }
    .status-pill.running .status-dot { background: var(--success); box-shadow: 0 0 8px var(--success); }
    .status-pill.stopped .status-dot { background: var(--danger); }

    textarea {
      width: 100%;
      height: 110px;
      background: #0f172a;
      border: 1px solid var(--border);
      border-radius: 8px;
      color: var(--text);
      padding: 12px;
      font-size: 13px;
      font-family: monospace;
      resize: vertical;
      margin-bottom: 12px;
      transition: border 0.2s;
    }
    textarea:focus { outline: none; border-color: var(--primary); }
    .btn-row { display: flex; gap: 10px; flex-wrap: wrap; }
    button {
      cursor: pointer;
      padding: 10px 18px;
      font-size: 13px;
      font-weight: 600;
      border-radius: 8px;
      border: none;
      transition: all 0.2s;
      display: inline-flex;
      align-items: center;
      gap: 6px;
    }
    .btn-primary { background: var(--primary); color: #042f2e; }
    .btn-primary:hover { background: var(--primary-hover); }
    .btn-secondary { background: #1e293b; color: var(--text); border: 1px solid var(--border); }
    .btn-secondary:hover { background: #334155; }
    .btn-success { background: var(--success); color: #022c22; }
    .btn-success:hover { background: #059669; }
    .btn-danger { background: rgba(239, 68, 68, 0.2); color: var(--danger); border: 1px solid rgba(239, 68, 68, 0.3); }
    .btn-danger:hover { background: rgba(239, 68, 68, 0.3); }

    .alert {
      padding: 12px 14px;
      border-radius: 8px;
      font-size: 13px;
      margin-top: 14px;
      display: none;
    }
    .alert.success { background: rgba(16, 185, 129, 0.15); color: var(--success); border: 1px solid rgba(16, 185, 129, 0.3); }
    .alert.error { background: rgba(239, 68, 68, 0.15); color: var(--danger); border: 1px solid rgba(239, 68, 68, 0.3); }
    .alert.info { background: rgba(6, 182, 212, 0.15); color: var(--primary); border: 1px solid rgba(6, 182, 212, 0.3); }

    .node-list {
      max-height: 200px;
      overflow-y: auto;
      margin-top: 10px;
      background: #0f172a;
      border-radius: 8px;
      padding: 8px;
      border: 1px solid var(--border);
    }
    .node-item {
      padding: 6px 10px;
      border-bottom: 1px solid #1e293b;
      font-size: 12px;
      font-family: monospace;
      color: #cbd5e1;
      display: flex;
      justify-content: space-between;
    }
    .node-item:last-child { border-bottom: none; }

    .info-table { width: 100%; border-collapse: collapse; font-size: 13px; }
    .info-table td { padding: 8px 0; border-bottom: 1px solid #1e293b; }
    .info-table td:first-child { color: var(--text-muted); width: 45%; }
    .info-table td:last-child { font-weight: 500; text-align: right; font-family: monospace; }

    .help-box {
      font-size: 12px;
      color: var(--text-muted);
      line-height: 1.6;
      margin-top: 10px;
      background: #0b1120;
      padding: 12px;
      border-radius: 8px;
      border-left: 3px solid var(--accent);
    }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div class="logo-group">
        <div class="logo-badge">Lx</div>
        <div>
          <h1>LxBox for Keenetic</h1>
          <div class="subtitle">Управление подписками, ядром sing-box-lx и маршрутизацией</div>
        </div>
      </div>
      <div class="nav-links">
        <a id="clashUiLink" href="/ui" target="_blank" class="nav-btn primary">📊 MetaCubeXD (9090)</a>
        <a id="awgLink" href="#" target="_blank" class="nav-btn">🛡️ AWG Manager (2222)</a>
        <a id="knLink" href="#" target="_blank" class="nav-btn">⚙️ Keenetic</a>
      </div>
    </header>

    <div class="grid grid-2">
      <!-- Main Left: Subscription Import & Integration -->
      <div style="display: flex; flex-direction: column; gap: 20px;">
        <!-- Card 1: Import -->
        <div class="card">
          <div class="card-title">
            <span>📥 Импорт подписки или ссылки</span>
          </div>
          <p style="font-size: 13px; color: var(--text-muted); margin-bottom: 10px;">
            Вставьте сюда строку любого формата. Система автоматически распознает и расшифрует ее:
          </p>
          <textarea id="linkInput" placeholder="Поддерживаются форматы:
• happ://crypt... (зашифрованная строка Happ)
• vless://... / hysteria2://... / awg://...
• https://... (удаленная подписка провайдера)"></textarea>
          <div class="btn-row">
            <button id="importBtn" class="btn-primary" onclick="doImport()">🚀 Импортировать в sing-box</button>
            <button id="updateSubBtn" class="btn-secondary" onclick="doUpdateSub()">🔄 Обновить подписку</button>
          </div>
          <div id="importAlert" class="alert"></div>
          <div id="nodesContainer" style="display: none; margin-top: 14px;">
            <div style="font-size: 12px; font-weight: 600; color: var(--text-muted); margin-bottom: 4px;">ИМПОРТИРОВАННЫЕ УЗЛЫ:</div>
            <div id="nodeList" class="node-list"></div>
          </div>
        </div>

        <!-- Card 2: Keenetic & AWG Manager Integration -->
        <div class="card">
          <div class="card-title">
            <span>🔌 Интеграция с KeeneticOS и AWG Manager</span>
          </div>
          <p style="font-size: 13px; color: var(--text-muted);">
            Создает системный сетевой интерфейс <code>Proxy0</code> в KeeneticOS, привязанный к SOCKS5-порту LxBox (<code>127.0.0.1:10808</code>).
          </p>
          <div class="help-box">
            <b>Как использовать совместно с AWG Manager и HydraRoute Neo:</b><br/>
            1. Включите интерфейс <b>Proxy0</b> кнопкой ниже.<br/>
            2. В веб-интерфейсе <b>AWG Manager</b> (или <b>HydraRoute Neo</b>) в списках маршрутизации доменов (YouTube, Discord и др.) выберите целью <code>Proxy0</code> или политику <code>LxBox</code>.<br/>
            3. Трафик пойдет через активный узел LxBox, а туннели WireGuard в AWG Manager продолжат работать параллельно без помех.
          </div>
          <div style="margin-top: 14px; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px;">
            <div>Статус интерфейса Proxy0: <span id="proxy0Status" style="font-weight: 600;">Проверка...</span></div>
            <div class="btn-row">
              <button id="enableProxyBtn" class="btn-success" onclick="toggleProxy0('enable')">Включить Proxy0</button>
              <button id="disableProxyBtn" class="btn-danger" onclick="toggleProxy0('disable')">Удалить Proxy0</button>
            </div>
          </div>
          <div id="ifaceAlert" class="alert"></div>
        </div>
      </div>

      <!-- Main Right: Status & Actions -->
      <div style="display: flex; flex-direction: column; gap: 20px;">
        <!-- Card 3: Status -->
        <div class="card">
          <div class="card-title" style="justify-content: space-between;">
            <span>Состояние ядра</span>
            <div id="statusPill" class="status-pill stopped">
              <div class="status-dot"></div>
              <span id="statusText">STOPPED</span>
            </div>
          </div>
          <table class="info-table">
            <tr>
              <td>Ядро:</td>
              <td>sing-box-lx</td>
            </tr>
            <tr>
              <td>Версия ядра:</td>
              <td id="sbVersion">1.14.4-lx.25</td>
            </tr>
            <tr>
              <td>PID процесса:</td>
              <td id="sbPid">—</td>
            </tr>
            <tr>
              <td>Mixed Proxy:</td>
              <td id="mixedPort">7890</td>
            </tr>
            <tr>
              <td>SOCKS5 Proxy:</td>
              <td id="socksPort">10808</td>
            </tr>
            <tr>
              <td>Clash API:</td>
              <td id="clashPort">9090</td>
            </tr>
            <tr>
              <td>Узлов в конфиге:</td>
              <td id="outboundsCount">0</td>
            </tr>
          </table>

          <div style="margin-top: 20px; display: flex; flex-direction: column; gap: 8px;">
            <button class="btn-secondary" style="width: 100%; justify-content: center;" onclick="doAction('restart')">🔄 Перезапустить LxBox</button>
            <div class="btn-row" style="width: 100%;">
              <button class="btn-secondary" style="flex: 1; justify-content: center;" onclick="doAction('start')">▶️ Старт</button>
              <button class="btn-secondary" style="flex: 1; justify-content: center;" onclick="doAction('stop')">⏹ Стоп</button>
            </div>
            <button class="btn-secondary" style="width: 100%; justify-content: center;" onclick="doAction('check')">🩺 Проверить конфиг</button>
          </div>
          <div id="actionAlert" class="alert"></div>
        </div>
      </div>
    </div>
  </div>

  <script>
    const host = window.location.hostname || "192.168.1.1";
    document.getElementById("clashUiLink").href = "http://" + host + ":9090/ui";
    document.getElementById("awgLink").href = "http://" + host + ":2222";
    document.getElementById("knLink").href = "http://" + host;

    function showAlert(id, msg, type) {
      const el = document.getElementById(id);
      el.className = "alert " + type;
      el.innerText = msg;
      el.style.display = "block";
      setTimeout(() => { el.style.display = "none"; }, 8000);
    }

    async function loadStatus() {
      try {
        const res = await fetch("/api/status");
        const data = await res.json();
        
        const pill = document.getElementById("statusPill");
        const text = document.getElementById("statusText");
        if (data.running) {
          pill.className = "status-pill running";
          text.innerText = "RUNNING";
          document.getElementById("sbPid").innerText = data.pid;
        } else {
          pill.className = "status-pill stopped";
          text.innerText = "STOPPED";
          document.getElementById("sbPid").innerText = "—";
        }
        document.getElementById("sbVersion").innerText = data.version;
        document.getElementById("mixedPort").innerText = data.mixed_port;
        document.getElementById("socksPort").innerText = data.socks_port;
        document.getElementById("clashPort").innerText = data.clash_port;
        document.getElementById("outboundsCount").innerText = (data.outbounds || []).length;

        const proxyStatus = document.getElementById("proxy0Status");
        if (data.ndms_proxy_active) {
          proxyStatus.innerText = "Включен (Proxy0 готов к маршрутизации)";
          proxyStatus.style.color = "var(--success)";
        } else {
          proxyStatus.innerText = "Не создан (можно включить)";
          proxyStatus.style.color = "var(--text-muted)";
        }

        if (data.outbounds && data.outbounds.length > 0) {
          document.getElementById("nodesContainer").style.display = "block";
          const list = document.getElementById("nodeList");
          list.innerHTML = "";
          data.outbounds.forEach(n => {
            const div = document.createElement("div");
            div.className = "node-item";
            div.innerText = n;
            list.appendChild(div);
          });
        }
      } catch (e) {
        console.error("status load error", e);
      }
    }

    async function doImport() {
      const link = document.getElementById("linkInput").value.trim();
      if (!link) {
        showAlert("importAlert", "Пожалуйста, вставьте ссылку или зашифрованную строку!", "error");
        return;
      }
      const btn = document.getElementById("importBtn");
      btn.disabled = true;
      btn.innerText = "⏳ Обработка...";
      try {
        const res = await fetch("/api/import", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ link })
        });
        const data = await res.json();
        if (data.success) {
          showAlert("importAlert", data.message, "success");
          document.getElementById("linkInput").value = "";
          loadStatus();
        } else {
          showAlert("importAlert", "Ошибка: " + data.message, "error");
        }
      } catch (e) {
        showAlert("importAlert", "Ошибка сети: " + e.message, "error");
      } finally {
        btn.disabled = false;
        btn.innerText = "🚀 Импортировать в sing-box";
      }
    }

    async function doUpdateSub() {
      const btn = document.getElementById("updateSubBtn");
      btn.disabled = true;
      btn.innerText = "⏳ Обновление...";
      try {
        const res = await fetch("/api/action", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ action: "update_sub" })
        });
        const data = await res.json();
        showAlert("importAlert", data.message, data.success ? "success" : "error");
        if (data.success) loadStatus();
      } catch (e) {
        showAlert("importAlert", "Ошибка: " + e.message, "error");
      } finally {
        btn.disabled = false;
        btn.innerText = "🔄 Обновить подписку";
      }
    }

    async function doAction(action) {
      try {
        const res = await fetch("/api/action", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ action })
        });
        const data = await res.json();
        showAlert("actionAlert", data.message, data.success ? "success" : "error");
        loadStatus();
      } catch (e) {
        showAlert("actionAlert", "Ошибка: " + e.message, "error");
      }
    }

    async function toggleProxy0(action) {
      try {
        const res = await fetch("/api/interface", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ action })
        });
        const data = await res.json();
        showAlert("ifaceAlert", data.message, data.success ? "success" : "error");
        loadStatus();
      } catch (e) {
        showAlert("ifaceAlert", "Ошибка: " + e.message, "error");
      }
    }

    loadStatus();
    setInterval(loadStatus, 5000);
  </script>
</body>
</html>
`
