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
	"time"
)

func runWebServer(listenAddr string, configPath string) error {
	configDir := filepath.Dir(configPath)
	subMgr := GetSubscriptionsManager(configPath)
	watchdog := GetWatchdogService(configDir, subMgr)

	// Seed with existing config outbounds if no subscriptions exist
	if len(subMgr.GetAll()) == 0 {
		obs := getActiveOutboundsObjects(configPath)
		if len(obs) > 0 {
			nodes := make([]*NodeInfo, 0, len(obs))
			for i, o := range obs {
				tag, _ := o["tag"].(string)
				tp, _ := o["type"].(string)
				srv, _ := o["server"].(string)
				var port int
				switch v := o["server_port"].(type) {
				case float64:
					port = int(v)
				case int:
					port = v
				}
				nodes = append(nodes, &NodeInfo{
					Tag:           tag,
					Type:          tp,
					Server:        srv,
					ServerPort:    port,
					EnabledInPool: true,
					IsActive:      i == 0,
					Status:        "unknown",
					RawOutbound:   o,
				})
			}
			subMgr.mu.Lock()
			subMgr.Subscriptions = append(subMgr.Subscriptions, &Subscription{
				ID:            "sub-1",
				Name:          "Основная подписка",
				URL:           "Импортировано из config.json",
				Enabled:       true,
				LastUpdated:   time.Now(),
				ProxyPort:     10809,
				KeeneticIface: "Proxy1",
				Nodes:         nodes,
			})
			data, _ := json.MarshalIndent(subMgr.Subscriptions, "", "  ")
			_ = os.WriteFile(subMgr.filePath, data, 0644)
			subMgr.mu.Unlock()
		}
	}

	// Start background watchdog
	watchdog.Start()

	// 1. Root HTML UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmlPage))
	})

	// 2. System Status API
	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		pid := getSingboxPID()
		wdStatus := watchdog.GetStatus()
		activeNode, _ := wdStatus["active_node"].(string)

		st := map[string]interface{}{
			"running":           pid > 0,
			"pid":               pid,
			"version":           "1.14.4-lx.25",
			"config_path":       configPath,
			"subscriptions_len": len(subMgr.GetAll()),
			"active_node":       activeNode,
			"ndms_proxy_active": checkNDMSProxyN(0),
			"mixed_port":        7890,
			"socks_port":        10808,
			"clash_port":        9090,
			"web_port":          9095,
		}
		json.NewEncoder(w).Encode(st)
	})

	// 3. Subscriptions APIs
	http.HandleFunc("/api/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		subs := subMgr.GetAll()
		json.NewEncoder(w).Encode(subs)
	})

	http.HandleFunc("/api/subscriptions/add", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
			return
		}
		link := strings.TrimSpace(req.URL)
		if link == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Ссылка или текст подписки пуст"})
			return
		}
		sub, err := subMgr.AddSubscription(req.Name, link)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		msg := fmt.Sprintf("Подписка '%s' успешно добавлена! Загружено %d узлов.", sub.Name, len(sub.Nodes))
		for _, n := range sub.Nodes {
			tagLower := strings.ToLower(n.Tag)
			if strings.Contains(tagLower, "лимит") || strings.Contains(tagLower, "привязать") || strings.Contains(tagLower, "обновите") {
				msg = fmt.Sprintf("⚠️ Подписка добавлена (%d узлов), но сервер сообщает: '%s'. Укажите HWID вашего телефона во вкладке «Профиль & HWID» или сбросьте привязку в боте провайдера.", len(sub.Nodes), n.Tag)
				break
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": msg,
			"sub_id":  sub.ID,
		})
	})

	http.HandleFunc("/api/subscriptions/refresh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
			return
		}
		if err := subMgr.RefreshSubscription(req.ID); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Подписка успешно обновлена!"})
	})

	http.HandleFunc("/api/subscriptions/delete", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
			return
		}
		if err := subMgr.DeleteSubscription(req.ID); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Подписка удалена"})
	})

	http.HandleFunc("/api/subscriptions/toggle_node", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SubID   string `json:"sub_id"`
			Tag     string `json:"tag"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
			return
		}
		if err := subMgr.ToggleNodePool(req.SubID, req.Tag, req.Enabled); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	})

	http.HandleFunc("/api/subscriptions/select_all", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SubID   string `json:"sub_id"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
			return
		}
		if err := subMgr.SetAllNodesInPool(req.SubID, req.Enabled); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	})

	http.HandleFunc("/api/subscriptions/set_active", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SubID string `json:"sub_id"`
			Tag   string `json:"tag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
			return
		}
		if err := subMgr.SetActiveNode(req.SubID, req.Tag); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Активный узел переключен на: %s", req.Tag),
		})
	})

	// 4. Watchdog APIs
	http.HandleFunc("/api/watchdog", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := watchdog.GetStatus()
		json.NewEncoder(w).Encode(status)
	})

	http.HandleFunc("/api/watchdog/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		var newCfg WatchdogConfig
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
			return
		}
		if err := watchdog.UpdateConfig(newCfg); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Настройки ватчдога сохранены"})
	})

	http.HandleFunc("/api/watchdog/check_now", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		go func() {
			watchdog.CheckActiveNode()
		}()
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Проверка запущена в фоне"})
	})

	// 5. Multi-Proxy Interface APIs (Proxy0..Proxy4)
	http.HandleFunc("/api/interfaces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var ifaces []map[string]interface{}
		// Proxy0 (Master)
		ifaces = append(ifaces, map[string]interface{}{
			"index":       0,
			"name":        "Proxy0",
			"description": "LxBox Master (Failover Pool)",
			"port":        10808,
			"active":      checkNDMSProxyN(0),
		})
		// Proxy1..Proxy5 for Subscriptions
		subs := subMgr.GetAll()
		for i := 1; i <= 5; i++ {
			subName := "Не назначена"
			if i <= len(subs) {
				subName = subs[i-1].Name
			}
			ifaces = append(ifaces, map[string]interface{}{
				"index":       i,
				"name":        fmt.Sprintf("Proxy%d", i),
				"description": fmt.Sprintf("LxBox Sub-%d (%s)", i, subName),
				"port":        10808 + i,
				"active":      checkNDMSProxyN(i),
			})
		}
		json.NewEncoder(w).Encode(ifaces)
	})

	http.HandleFunc("/api/interfaces/manage", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, `{"success":false,"message":"POST required"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Index  int    `json:"index"`
			Action string `json:"action"` // "enable" | "disable"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
			return
		}
		port := 10808 + req.Index
		desc := fmt.Sprintf("LxBox-%d", req.Index)
		if req.Index == 0 {
			desc = "LxBox-Master"
		}

		if req.Action == "enable" {
			if err := configureNDMSProxyN(req.Index, desc, port, true); err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Интерфейс Proxy%d успешно создан в KeeneticOS!", req.Index),
			})
		} else {
			if err := configureNDMSProxyN(req.Index, desc, port, false); err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Интерфейс Proxy%d удален из KeeneticOS.", req.Index),
			})
		}
	})

	// Legacy single import support
	http.HandleFunc("/api/import", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req struct {
			Link string `json:"link"`
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &req)
		link := strings.TrimSpace(req.Link)
		if link == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Ссылка пуста"})
			return
		}
		sub, err := subMgr.AddSubscription("Импорт", link)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Импортировано %d узлов!", len(sub.Nodes)),
			"count":   len(sub.Nodes),
		})
	})

	// System Actions
	http.HandleFunc("/api/action", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req struct {
			Action string `json:"action"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		switch req.Action {
		case "restart":
			_ = restartSingboxService()
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Служба перезапущена"})
		case "stop":
			exec.Command("/opt/etc/init.d/S99lxbox", "stop").Run()
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Служба остановлена"})
		case "start":
			exec.Command("/opt/etc/init.d/S99lxbox", "start").Run()
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Служба запущена"})
		case "check":
			out, err := exec.Command("sing-box", "check", "-c", configPath).CombinedOutput()
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": string(out)})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Конфигурация sing-box корректна!"})
			}
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Неизвестное действие"})
		}
	})

	// 6. Device Profile & HWID APIs
	http.HandleFunc("/api/device_profile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			profile := GetDeviceProfile()
			json.NewEncoder(w).Encode(profile)
			return
		}
		if r.Method == http.MethodPost {
			var req struct {
				RawHeaders string `json:"raw_headers"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Invalid JSON"})
				return
			}
			p, err := UpdateDeviceProfile(req.RawHeaders)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
				return
			}
			LogEvent("info", fmt.Sprintf("Обновлен профиль устройства: HWID=%s (%s %s)", p.HWID, p.DeviceOS, p.DeviceModel))
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Профиль устройства сохранен! Активный HWID: %s", p.HWID),
				"profile": p,
			})
			return
		}
		http.Error(w, `{"success":false,"message":"GET or POST required"}`, http.StatusMethodNotAllowed)
	})

	http.HandleFunc("/api/device_profile/presets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"android":     PresetLxBoxAndroid,
			"ios":         PresetHappIOS,
			"windows":     PresetWindows,
			"random_hwid": GenerateRandomHWID(),
		})
	})

	fmt.Printf("[+] LxBox Web & Watchdog Server listening on http://%s\n", listenAddr)
	server := &http.Server{
		Addr:         listenAddr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return server.ListenAndServe()
}

func checkNDMSProxyN(index int) bool {
	if _, err := exec.LookPath("ndmc"); err != nil {
		return false
	}
	ifaceName := fmt.Sprintf("Proxy%d", index)
	out, err := exec.Command("ndmc", "-c", "show interface "+ifaceName).Output()
	if err != nil {
		return false
	}
	s := string(out)
	return strings.Contains(s, ifaceName) && !strings.Contains(s, "unable to find")
}

func configureNDMSProxyN(index int, desc string, port int, enable bool) error {
	if _, err := exec.LookPath("ndmc"); err != nil {
		return fmt.Errorf("утилита ndmc не найдена (требуется роутер Keenetic)")
	}
	ifaceName := fmt.Sprintf("Proxy%d", index)
	if enable {
		cmds := []string{
			fmt.Sprintf("interface %s", ifaceName),
			fmt.Sprintf("interface %s description %s", ifaceName, desc),
			fmt.Sprintf("interface %s proxy proto socks5", ifaceName),
			fmt.Sprintf("interface %s proxy upstream 127.0.0.1 %d", ifaceName, port),
			fmt.Sprintf("interface %s proxy socks5-udp", ifaceName),
			fmt.Sprintf("interface %s ip global auto", ifaceName),
			fmt.Sprintf("interface %s up", ifaceName),
			"system configuration save",
		}
		for _, c := range cmds {
			if out, err := exec.Command("ndmc", "-c", c).CombinedOutput(); err != nil {
				return fmt.Errorf("ndmc '%s' ошибка: %s", c, string(out))
			}
		}
	} else {
		cmds := []string{
			fmt.Sprintf("no interface %s", ifaceName),
			"system configuration save",
		}
		for _, c := range cmds {
			exec.Command("ndmc", "-c", c).Run()
		}
	}
	return nil
}

func getActiveOutboundsObjects(configPath string) []map[string]interface{} {
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
	var res []map[string]interface{}
	for _, o := range obsRaw {
		if om, ok := o.(map[string]interface{}); ok {
			tp, _ := om["type"].(string)
			tag, _ := om["tag"].(string)
			if tag != "" && tp != "direct" && tp != "block" && tp != "dns" && tp != "selector" && !strings.HasSuffix(tp, "-select") {
				res = append(res, om)
			}
		}
	}
	return res
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

const htmlPage = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>LxBox for Keenetic — Subscriptions, Watchdog & Multi-Proxy</title>
  <style>
    :root {
      --bg: #0b0f19;
      --card-bg: #111827;
      --card-alt: #162032;
      --border: #1f2937;
      --text: #f9fafb;
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
      padding: 16px;
    }
    .container { max-width: 1100px; margin: 0 auto; }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding-bottom: 16px;
      border-bottom: 1px solid var(--border);
      margin-bottom: 20px;
      flex-wrap: wrap;
      gap: 12px;
    }
    .logo-group { display: flex; align-items: center; gap: 12px; }
    .logo-badge {
      background: linear-gradient(135deg, var(--primary), var(--accent));
      color: white;
      font-weight: 900;
      font-size: 22px;
      padding: 6px 14px;
      border-radius: 10px;
      box-shadow: 0 4px 12px rgba(6, 182, 212, 0.3);
    }
    h1 { font-size: 20px; font-weight: 700; }
    .subtitle { font-size: 13px; color: var(--text-muted); }
    .nav-links { display: flex; gap: 8px; flex-wrap: wrap; }
    .nav-btn {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 12px;
      background: #1f2937;
      color: var(--text);
      text-decoration: none;
      border-radius: 8px;
      font-size: 13px;
      font-weight: 500;
      border: 1px solid var(--border);
      transition: all 0.2s;
    }
    .nav-btn:hover { background: #374151; border-color: var(--primary); }
    .nav-btn.primary { background: rgba(6, 182, 212, 0.15); color: var(--primary); border-color: rgba(6, 182, 212, 0.4); }

    /* Tabs */
    .tabs { display: flex; gap: 8px; margin-bottom: 20px; border-bottom: 1px solid var(--border); padding-bottom: 8px; flex-wrap: wrap; }
    .tab-btn {
      padding: 10px 18px;
      background: transparent;
      border: none;
      color: var(--text-muted);
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
      border-radius: 8px;
      transition: all 0.2s;
      display: flex;
      align-items: center;
      gap: 6px;
    }
    .tab-btn:hover { color: var(--text); background: #1f2937; }
    .tab-btn.active { color: var(--primary); background: rgba(6, 182, 212, 0.15); border: 1px solid rgba(6, 182, 212, 0.3); }

    .tab-content { display: none; }
    .tab-content.active { display: block; }

    .card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px;
      margin-bottom: 20px;
      box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.2);
    }
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 14px;
      flex-wrap: wrap;
      gap: 10px;
    }
    .card-title {
      font-size: 16px;
      font-weight: 600;
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

    input[type="text"], input[type="number"], textarea, select {
      width: 100%;
      background: #090d16;
      border: 1px solid var(--border);
      border-radius: 8px;
      color: var(--text);
      padding: 10px 12px;
      font-size: 13px;
      font-family: inherit;
      margin-bottom: 10px;
      transition: border 0.2s;
    }
    textarea { font-family: monospace; resize: vertical; min-height: 80px; }
    input:focus, textarea:focus { outline: none; border-color: var(--primary); }

    button {
      cursor: pointer;
      padding: 8px 16px;
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
    .btn-secondary { background: #1f2937; color: var(--text); border: 1px solid var(--border); }
    .btn-secondary:hover { background: #374151; }
    .btn-success { background: var(--success); color: #022c22; }
    .btn-danger { background: rgba(239, 68, 68, 0.2); color: var(--danger); border: 1px solid rgba(239, 68, 68, 0.3); }
    .btn-danger:hover { background: rgba(239, 68, 68, 0.3); }

    .sub-card {
      background: var(--card-alt);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 16px;
      margin-bottom: 16px;
    }
    .sub-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 10px;
      flex-wrap: wrap;
      gap: 8px;
    }
    .sub-title { font-size: 15px; font-weight: 600; color: var(--primary); }
    .sub-meta { font-size: 12px; color: var(--text-muted); }

    .node-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 13px;
      margin-top: 10px;
    }
    .node-table th {
      text-align: left;
      padding: 8px;
      color: var(--text-muted);
      border-bottom: 1px solid var(--border);
      font-size: 12px;
    }
    .node-table td {
      padding: 8px;
      border-bottom: 1px solid #1f2937;
      vertical-align: middle;
    }
    .node-table tr:hover { background: rgba(255,255,255,0.02); }

    .badge {
      display: inline-block;
      padding: 2px 8px;
      border-radius: 12px;
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
    }
    .badge.ok { background: rgba(16, 185, 129, 0.2); color: var(--success); }
    .badge.frozen { background: rgba(6, 182, 212, 0.2); color: var(--primary); }
    .badge.dead { background: rgba(239, 68, 68, 0.2); color: var(--danger); }
    .badge.unknown { background: rgba(156, 163, 175, 0.2); color: var(--text-muted); }

    .log-box {
      background: #070a12;
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px;
      font-family: monospace;
      font-size: 12px;
      max-height: 250px;
      overflow-y: auto;
      color: #94a3b8;
    }
    .log-line { margin-bottom: 4px; display: flex; gap: 8px; }
    .log-time { color: #64748b; }
    .log-info { color: #38bdf8; }
    .log-warn { color: #fbbf24; }
    .log-error { color: #f87171; }
    .log-success { color: #4ade80; }

    .grid-2 { display: grid; grid-template-columns: 1fr; gap: 20px; }
    @media(min-width: 800px) {
      .grid-2 { grid-template-columns: 1fr 1fr; }
    }

    .alert {
      padding: 10px 14px;
      border-radius: 8px;
      font-size: 13px;
      margin-bottom: 12px;
      display: none;
    }
    .alert.success { background: rgba(16, 185, 129, 0.15); color: var(--success); border: 1px solid rgba(16, 185, 129, 0.3); }
    .alert.error { background: rgba(239, 68, 68, 0.15); color: var(--danger); border: 1px solid rgba(239, 68, 68, 0.3); }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div class="logo-group">
        <div class="logo-badge">Lx</div>
        <div>
          <h1>LxBox for Keenetic</h1>
          <div class="subtitle">Управление подписками (до 5), Ватчдог с автопереключением и Мульти-прокси</div>
        </div>
      </div>
      <div class="nav-links">
        <a id="clashUiLink" href="/ui" target="_blank" class="nav-btn primary">📊 MetaCubeXD (9090)</a>
        <a id="awgLink" href="#" target="_blank" class="nav-btn">🛡️ AWG Manager (2222)</a>
        <a id="knLink" href="#" target="_blank" class="nav-btn">⚙️ Keenetic</a>
      </div>
    </header>

    <div class="tabs">
      <button class="tab-btn active" onclick="switchTab('subsTab', this)">📑 Подписки & Узлы (<span id="subsCountBadge">0</span>/5)</button>
      <button class="tab-btn" onclick="switchTab('watchdogTab', this)">🐕 Ватчдог & Мониторинг</button>
      <button class="tab-btn" onclick="switchTab('ifacesTab', this)">🔌 Сетевые интерфейсы (Proxy0..4)</button>
      <button class="tab-btn" onclick="switchTab('hwidTab', this)">📱 Профиль & HWID</button>
      <button class="tab-btn" onclick="switchTab('statusTab', this)">⚙️ Ядро & Служба</button>
    </div>

    <!-- Alert Banner -->
    <div id="globalAlert" class="alert"></div>

    <!-- TAB 1: Subscriptions -->
    <div id="subsTab" class="tab-content active">
      <div class="card">
        <div class="card-header">
          <div class="card-title">
            <span>➕ Добавить подписку (поддерживается до 5 подписок)</span>
          </div>
        </div>
        <div style="display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 10px;">
          <input type="text" id="newSubName" placeholder="Название подписки (например: Happ Основной, VLESS Нидерланды)" style="flex: 1; min-width: 200px; margin-bottom: 0;">
          <button class="btn-primary" onclick="addSubscription()">📥 Добавить подписку</button>
        </div>
        <textarea id="newSubURL" placeholder="Вставьте ссылку любого формата:
• happ://crypt... (зашифрованная строка Happ)
• https://provider.com/sub?... (удаленный URL подписки)
• vless://... / hysteria2://... / awg://..."></textarea>
      </div>

      <div id="subscriptionsList">
        <!-- Rendered Subscriptions Cards -->
      </div>
    </div>

    <!-- TAB 2: Watchdog -->
    <div id="watchdogTab" class="tab-content">
      <div class="grid-2">
        <div class="card">
          <div class="card-header">
            <div class="card-title"><span>🐕 Состояние Ватчдога</span></div>
            <button class="btn-primary" onclick="checkActiveNow()">⚡ Проверить сейчас</button>
          </div>
          <table style="width: 100%; border-collapse: collapse; font-size: 13px;">
            <tr><td style="padding: 8px 0; color: var(--text-muted);">Активный узел:</td><td id="wdActiveNode" style="font-weight: 600; font-family: monospace;">—</td></tr>
            <tr><td style="padding: 8px 0; color: var(--text-muted);">Состояние активного:</td><td id="wdActiveStatus"><span class="badge unknown">Неизвестно</span></td></tr>
            <tr><td style="padding: 8px 0; color: var(--text-muted);">Задержка (Latency):</td><td id="wdActiveLatency">— ms</td></tr>
            <tr><td style="padding: 8px 0; color: var(--text-muted);">Неудач подряд:</td><td id="wdActiveFails">0</td></tr>
          </table>

          <div style="margin-top: 16px; padding: 12px; background: #070a12; border-radius: 8px; font-size: 12px; color: var(--text-muted); line-height: 1.6;">
            <b>Логика работы Ватчдога:</b><br/>
            1. Каждые <code>N</code> сек проверяет доступность целевых HTTP-страниц через активный туннель.<br/>
            2. При сбое делает до <code>M</code> повторов с паузой.<br/>
            3. Если сервер мертв — <b>замораживает его на N часов</b> и ищет замену среди узлов, отмеченных чекбоксами в пуле.<br/>
            4. <b>Перед переключением</b> Ватчдог проверяет кандидата на жив/мертв, и переключает только на живой сервер!
          </div>
        </div>

        <div class="card">
          <div class="card-title"><span>⚙️ Настройки проверки & Заморозки</span></div>
          <label style="font-size: 12px; color: var(--text-muted);">Интервал проверки активного узла (сек):</label>
          <input type="number" id="wdActiveInterval" min="5" max="3600" value="30">

          <label style="font-size: 12px; color: var(--text-muted);">Интервал мониторинга всего пула (сек):</label>
          <input type="number" id="wdPoolInterval" min="30" max="86400" value="300">

          <label style="font-size: 12px; color: var(--text-muted);">Количество неудач до переключения:</label>
          <input type="number" id="wdMaxFails" min="1" max="10" value="3">

          <label style="font-size: 12px; color: var(--text-muted);">Пауза между повторами (сек):</label>
          <input type="number" id="wdRetryPause" min="1" max="60" value="3">

          <label style="font-size: 12px; color: var(--text-muted);">Заморозка упавшего сервера (часов):</label>
          <input type="number" id="wdFreezeHours" min="1" max="72" value="2">

          <label style="font-size: 12px; color: var(--text-muted);">Целевые адреса для проверки (по одному на строку):</label>
          <textarea id="wdTargets" style="height: 70px;">https://instagram.com
https://x.com
https://grok.ai</textarea>

          <button class="btn-success" onclick="saveWatchdogConfig()">💾 Сохранить параметры</button>
        </div>
      </div>

      <div class="card">
        <div class="card-title"><span>📜 Журнал событий Ватчдога (Live Log)</span></div>
        <div id="wdLogBox" class="log-box">
          <div style="color: #64748b;">Ожидание событий ватчдога...</div>
        </div>
      </div>
    </div>

    <!-- TAB 3: Interfaces -->
    <div id="ifacesTab" class="tab-content">
      <div class="card">
        <div class="card-title"><span>🔌 Сетевые интерфейсы KeeneticOS (Proxy0 .. Proxy4)</span></div>
        <p style="font-size: 13px; color: var(--text-muted); margin-bottom: 16px;">
          Вы можете привязать каждую подписку к отдельному интерфейсу KeeneticOS. Это позволяет направлять разные домашние устройства (ПК, ТВ, телефоны) через разные подписки, либо использовать <b>Proxy0</b> в <b>HydraRoute Neo</b> для автоматического обхода блокировок.
        </p>

        <table class="node-table">
          <thead>
            <tr>
              <th>Интерфейс</th>
              <th>Назначение</th>
              <th>Локальный SOCKS5 порт</th>
              <th>Статус в KeeneticOS</th>
              <th>Действие</th>
            </tr>
          </thead>
          <tbody id="ifacesTableBody">
            <!-- Rendered Interfaces -->
          </tbody>
        </table>
      </div>
    </div>

    <!-- TAB 5: Device Profile & HWID -->
    <div id="hwidTab" class="tab-content">
      <div class="grid-2">
        <div class="card">
          <div class="card-header">
            <div class="card-title"><span>📱 Настройка заголовков & HWID</span></div>
          </div>
          <p style="font-size: 13px; color: var(--text-muted); margin-bottom: 12px;">
            Укажите заголовки клиента, передаваемые при скачивании и обновлении подписок. Поддерживаются разделители <code>:</code> и стили с дефисом (<code>X-HWID</code>) или подчеркиванием (<code>X_Device_OS</code>).
          </p>

          <div style="margin-bottom: 12px; display: flex; gap: 6px; flex-wrap: wrap;">
            <span style="font-size: 12px; color: var(--text-muted); align-self: center; margin-right: 4px;">Шаблоны:</span>
            <button class="btn-secondary" style="font-size: 12px; padding: 4px 10px;" onclick="applyHwidPreset('android')">📱 LxBox Android</button>
            <button class="btn-secondary" style="font-size: 12px; padding: 4px 10px;" onclick="applyHwidPreset('ios')">🍏 Happ iOS (iPhone 15)</button>
            <button class="btn-secondary" style="font-size: 12px; padding: 4px 10px;" onclick="applyHwidPreset('windows')">💻 Windows 11</button>
            <button class="btn-secondary" style="font-size: 12px; padding: 4px 10px; color: var(--accent); border-color: rgba(139,92,246,0.4);" onclick="generateRandomHwid()">🎲 Случайный HWID</button>
          </div>

          <textarea id="hwidHeadersInput" style="height: 190px; font-family: monospace; font-size: 12px; line-height: 1.4;" placeholder="User-Agent: LxBox/2.25.18/android&#10;X-Device-OS: Android&#10;X-HWID: e6ce21f5efaa40b99711058bd35ba999"></textarea>

          <div style="display: flex; gap: 8px; flex-wrap: wrap;">
            <button class="btn-success" onclick="saveDeviceProfile()">💾 Сохранить заголовки</button>
            <button class="btn-secondary" onclick="applyHwidPreset('android')">🔄 Сбросить к Android</button>
          </div>
        </div>

        <div class="card">
          <div class="card-header">
            <div class="card-title"><span>🔍 Распознанный профиль клиента</span></div>
            <span class="badge ok" id="profileCountBadge">0 заголовков</span>
          </div>

          <table style="width: 100%; border-collapse: collapse; font-size: 13px;">
            <tr><td style="padding: 6px 0; color: var(--text-muted); width: 140px;">Активный X-HWID:</td><td id="dispProfileHWID" style="font-family: monospace; font-weight: 600; color: var(--primary); word-break: break-all;">—</td></tr>
            <tr><td style="padding: 6px 0; color: var(--text-muted);">ОС устройства:</td><td id="dispProfileOS" style="font-weight: 500;">—</td></tr>
            <tr><td style="padding: 6px 0; color: var(--text-muted);">Модель:</td><td id="dispProfileModel">—</td></tr>
            <tr><td style="padding: 6px 0; color: var(--text-muted);">Локаль (Язык):</td><td id="dispProfileLocale">—</td></tr>
            <tr><td style="padding: 6px 0; color: var(--text-muted);">Версия ОС:</td><td id="dispProfileVerOS">—</td></tr>
            <tr><td style="padding: 6px 0; color: var(--text-muted);">Версия приложения:</td><td id="dispProfileAppVer">—</td></tr>
            <tr><td style="padding: 6px 0; color: var(--text-muted);">User-Agent:</td><td id="dispProfileUA" style="font-family: monospace; font-size: 12px; word-break: break-all;">—</td></tr>
            <tr><td style="padding: 6px 0; color: var(--text-muted);">Обновлено:</td><td id="dispProfileUpdated" style="font-size: 12px; color: var(--text-muted);">—</td></tr>
          </table>

          <div style="margin-top: 14px; padding: 10px; background: #070a12; border-radius: 8px; font-size: 12px; color: var(--text-muted); line-height: 1.5;">
            <b>ℹ️ Мониторинг передачи HWID:</b><br/>
            При каждом запросе подписки в лог ватчдога записывается событие:
            <code style="color: #38bdf8; display: block; margin-top: 4px;">Передача X-HWID: &lt;ваш-hwid&gt; (OS: ..., Model: ...)</code>
            Вы можете проконтролировать передачу во вкладке «Ватчдог & Мониторинг» или по команде <code>lxbox log</code>.
          </div>
        </div>
      </div>
    </div>

    <!-- TAB 4: Core & Service Status -->
    <div id="statusTab" class="tab-content">
      <div class="card">
        <div class="card-header">
          <div class="card-title"><span>Состояние процесса sing-box-lx</span></div>
          <div id="coreStatusPill" class="status-pill stopped"><div class="status-dot"></div><span id="coreStatusText">STOPPED</span></div>
        </div>
        <table style="width: 100%; border-collapse: collapse; font-size: 13px;">
          <tr><td style="padding: 8px 0; color: var(--text-muted);">Версия ядра:</td><td id="coreVersion">1.14.4-lx.25</td></tr>
          <tr><td style="padding: 8px 0; color: var(--text-muted);">PID процесса:</td><td id="corePid">—</td></tr>
          <tr><td style="padding: 8px 0; color: var(--text-muted);">Mixed Port:</td><td>7890 (SOCKS5/HTTP)</td></tr>
          <tr><td style="padding: 8px 0; color: var(--text-muted);">Master SOCKS5 Port:</td><td>10808 (Failover Pool)</td></tr>
          <tr><td style="padding: 8px 0; color: var(--text-muted);">Clash REST API:</td><td>9090</td></tr>
          <tr><td style="padding: 8px 0; color: var(--text-muted);">Конфигурация:</td><td id="coreCfgPath" style="font-family: monospace;">—</td></tr>
        </table>
        <div style="margin-top: 16px; display: flex; gap: 8px; flex-wrap: wrap;">
          <button class="btn-secondary" onclick="doCoreAction('restart')">🔄 Перезапустить ядро</button>
          <button class="btn-secondary" onclick="doCoreAction('start')">▶️ Запустить</button>
          <button class="btn-secondary" onclick="doCoreAction('stop')">⏹ Остановить</button>
          <button class="btn-secondary" onclick="doCoreAction('check')">🩺 Проверить config.json</button>
        </div>
      </div>
    </div>
  </div>

  <script>
    const host = window.location.hostname || "192.168.1.1";
    document.getElementById("clashUiLink").href = "http://" + host + ":9090/ui/?hostname=" + host + "&port=9090";
    document.getElementById("awgLink").href = "http://" + host + ":2222";
    document.getElementById("knLink").href = "http://" + host;

    function showAlert(msg, isSuccess) {
      const el = document.getElementById("globalAlert");
      el.className = "alert " + (isSuccess ? "success" : "error");
      el.innerText = msg;
      el.style.display = "block";
      setTimeout(() => { el.style.display = "none"; }, 6000);
    }

    function switchTab(tabId, btn) {
      document.querySelectorAll(".tab-content").forEach(el => el.classList.remove("active"));
      document.querySelectorAll(".tab-btn").forEach(el => el.classList.remove("active"));
      document.getElementById(tabId).classList.add("active");
      btn.classList.add("active");
      if (tabId === "watchdogTab") loadWatchdog();
      if (tabId === "ifacesTab") loadInterfaces();
      if (tabId === "subsTab") loadSubscriptions();
      if (tabId === "hwidTab") loadDeviceProfile();
    }

    async function loadStatus() {
      try {
        const res = await fetch("/api/status");
        const data = await res.json();
        const pill = document.getElementById("coreStatusPill");
        const text = document.getElementById("coreStatusText");
        if (data.running) {
          pill.className = "status-pill running";
          text.innerText = "RUNNING";
          document.getElementById("corePid").innerText = data.pid;
        } else {
          pill.className = "status-pill stopped";
          text.innerText = "STOPPED";
          document.getElementById("corePid").innerText = "—";
        }
        document.getElementById("coreVersion").innerText = data.version;
        document.getElementById("coreCfgPath").innerText = data.config_path;
        document.getElementById("subsCountBadge").innerText = data.subscriptions_len;
      } catch(e) { console.error(e); }
    }

    async function loadSubscriptions() {
      try {
        const res = await fetch("/api/subscriptions");
        const subs = await res.json();
        document.getElementById("subsCountBadge").innerText = subs.length;
        const container = document.getElementById("subscriptionsList");
        container.innerHTML = "";

        if (subs.length === 0) {
          container.innerHTML = '<div style="text-align: center; color: var(--text-muted); padding: 30px;">Нет добавленных подписок. Вставьте ссылку выше, чтобы добавить первую.</div>';
          return;
        }

        subs.forEach((s, idx) => {
          const card = document.createElement("div");
          card.className = "sub-card";

          let poolCheckedCount = 0;
          s.nodes.forEach(n => { if (n.enabled_in_pool) poolCheckedCount++; });

          let rows = "";
          (s.nodes || []).forEach(n => {
            const chk = n.enabled_in_pool ? "checked" : "";
            const actStyle = n.is_active ? "background: var(--primary); color: #042f2e;" : "";
            const actLabel = n.is_active ? "★ Активен" : "Сделать акт.";
            const lat = n.last_latency_ms > 0 ? n.last_latency_ms + " ms" : "—";
            rows += "<tr>" +
              "<td><input type='checkbox' " + chk + " onchange='togglePool(\"" + s.id + "\", \"" + escapeHTML(n.tag) + "\", this.checked)'></td>" +
              "<td><button class='btn-secondary' style='padding: 2px 8px; font-size: 11px; " + actStyle + "' onclick='setActiveNode(\"" + s.id + "\", \"" + escapeHTML(n.tag) + "\")'>" + actLabel + "</button></td>" +
              "<td style='font-family: monospace; font-weight: 500;'>" + escapeHTML(n.tag) + "</td>" +
              "<td><span style='font-size: 11px; text-transform: uppercase; color: var(--accent);'>" + escapeHTML(n.type) + "</span></td>" +
              "<td style='font-size: 12px; color: var(--text-muted);'>" + escapeHTML(n.server) + ":" + n.server_port + "</td>" +
              "<td>" + renderStatusBadge(n) + "</td>" +
              "<td style='font-family: monospace;'>" + lat + "</td>" +
              "</tr>";
          });

          card.innerHTML = 
            "<div class='sub-header'>" +
              "<div>" +
                "<span class='sub-title'>" + escapeHTML(s.name) + "</span>" +
                "<span style='font-size: 11px; background: rgba(6,182,212,0.15); color: var(--primary); padding: 2px 6px; border-radius: 6px; margin-left: 6px;'>" +
                  "Port: " + s.proxy_port + " (Keenetic: " + s.keenetic_iface + ")" +
                "</span>" +
                "<div class='sub-meta'>Узлов: " + s.nodes.length + " (В пуле: " + poolCheckedCount + ") • Обновлено: " + new Date(s.last_updated).toLocaleString() + "</div>" +
              "</div>" +
              "<div style='display: flex; gap: 6px;'>" +
                "<button class='btn-secondary' onclick='refreshSub(\"" + s.id + "\")'>🔄 Обновить</button>" +
                "<button class='btn-danger' onclick='deleteSub(\"" + s.id + "\")'>🗑️ Удалить</button>" +
              "</div>" +
            "</div>" +
            "<div style='margin-bottom: 8px; display: flex; gap: 8px;'>" +
              "<button class='btn-secondary' style='font-size: 11px; padding: 4px 8px;' onclick='selectAllNodes(\"" + s.id + "\", true)'>Отметить все в пул</button>" +
              "<button class='btn-secondary' style='font-size: 11px; padding: 4px 8px;' onclick='selectAllNodes(\"" + s.id + "\", false)'>Снять все</button>" +
            "</div>" +
            "<div style='max-height: 280px; overflow-y: auto;'>" +
              "<table class='node-table'>" +
                "<thead><tr>" +
                  "<th style='width: 40px;'>Пул</th><th>Активен</th><th>Тег / Имя</th><th>Тип</th><th>Адрес</th><th>Статус</th><th>Задержка</th>" +
                "</tr></thead>" +
                "<tbody>" + rows + "</tbody>" +
              "</table>" +
            "</div>";
          container.appendChild(card);
        });
      } catch(e) { console.error(e); }
    }

    function renderStatusBadge(node) {
      if (node.status === "ok") return '<span class="badge ok">Здоров</span>';
      if (node.status === "frozen") {
        const until = new Date(node.frozen_until).toLocaleTimeString();
        return '<span class="badge frozen" title="Заморожен до ' + until + '">❄️ Заморожен</span>';
      }
      if (node.status === "dead") return '<span class="badge dead">Недоступен</span>';
      return '<span class="badge unknown">Не проверен</span>';
    }

    async function addSubscription() {
      const name = document.getElementById("newSubName").value.trim();
      const url = document.getElementById("newSubURL").value.trim();
      if (!url) { showAlert("Вставьте ссылку или текст подписки!", false); return; }
      try {
        const res = await fetch("/api/subscriptions/add", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name, url })
        });
        const data = await res.json();
        showAlert(data.message, data.success);
        if (data.success) {
          document.getElementById("newSubName").value = "";
          document.getElementById("newSubURL").value = "";
          loadSubscriptions();
        }
      } catch(e) { showAlert("Ошибка сети: " + e.message, false); }
    }

    async function refreshSub(id) {
      try {
        const res = await fetch("/api/subscriptions/refresh", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ id })
        });
        const data = await res.json();
        showAlert(data.message, data.success);
        loadSubscriptions();
      } catch(e) { showAlert("Ошибка сети: " + e.message, false); }
    }

    async function deleteSub(id) {
      if (!confirm("Вы уверены, что хотите удалить эту подписку?")) return;
      try {
        const res = await fetch("/api/subscriptions/delete", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ id })
        });
        const data = await res.json();
        showAlert(data.message, data.success);
        loadSubscriptions();
      } catch(e) { showAlert("Ошибка сети: " + e.message, false); }
    }

    async function togglePool(subId, tag, enabled) {
      await fetch("/api/subscriptions/toggle_node", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sub_id: subId, tag, enabled })
      });
    }

    async function selectAllNodes(subId, enabled) {
      await fetch("/api/subscriptions/select_all", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sub_id: subId, enabled })
      });
      loadSubscriptions();
    }

    async function setActiveNode(subId, tag) {
      try {
        const res = await fetch("/api/subscriptions/set_active", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ sub_id: subId, tag })
        });
        const data = await res.json();
        showAlert(data.message, data.success);
        loadSubscriptions();
      } catch(e) { showAlert(e.message, false); }
    }

    async function loadWatchdog() {
      try {
        const res = await fetch("/api/watchdog");
        const data = await res.json();
        document.getElementById("wdActiveNode").innerText = data.active_node;
        document.getElementById("wdActiveStatus").innerHTML = renderStatusBadge({ status: data.active_status });
        document.getElementById("wdActiveLatency").innerText = (data.active_latency > 0 ? data.active_latency + " ms" : "—");
        document.getElementById("wdActiveFails").innerText = data.active_fail_count;

        if (data.config) {
          document.getElementById("wdActiveInterval").value = data.config.active_interval_sec;
          document.getElementById("wdPoolInterval").value = data.config.pool_interval_sec;
          document.getElementById("wdMaxFails").value = data.config.max_failures;
          document.getElementById("wdRetryPause").value = data.config.retry_pause_sec;
          document.getElementById("wdFreezeHours").value = data.config.freeze_hours;
          document.getElementById("wdTargets").value = (data.config.test_urls || []).join("\n");
        }

        const logBox = document.getElementById("wdLogBox");
        if (data.logs && data.logs.length > 0) {
          logBox.innerHTML = data.logs.map(l =>
            '<div class="log-line"><span class="log-time">[' + l.timestamp + ']</span> <span class="log-' + l.level + '">' + escapeHTML(l.message) + '</span></div>'
          ).join('');
          logBox.scrollTop = logBox.scrollHeight;
        }
      } catch(e) { console.error(e); }
    }

    async function saveWatchdogConfig() {
      const activeSec = parseInt(document.getElementById("wdActiveInterval").value) || 30;
      const poolSec = parseInt(document.getElementById("wdPoolInterval").value) || 300;
      const maxFails = parseInt(document.getElementById("wdMaxFails").value) || 3;
      const retryPause = parseInt(document.getElementById("wdRetryPause").value) || 3;
      const freezeH = parseInt(document.getElementById("wdFreezeHours").value) || 2;
      const urls = document.getElementById("wdTargets").value.split("\n").map(s => s.trim()).filter(s => s.length > 0);

      try {
        const res = await fetch("/api/watchdog/config", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            enabled: true,
            active_interval_sec: activeSec,
            pool_interval_sec: poolSec,
            max_failures: maxFails,
            retry_pause_sec: retryPause,
            freeze_hours: freezeH,
            test_urls: urls,
            proxy_addr: "127.0.0.1:7890",
            clash_api: "http://127.0.0.1:9090"
          })
        });
        const data = await res.json();
        showAlert(data.message, data.success);
      } catch(e) { showAlert("Ошибка: " + e.message, false); }
    }

    async function checkActiveNow() {
      showAlert("Запущена внеплановая проверка активного узла...", true);
      await fetch("/api/watchdog/check_now", { method: "POST" });
      setTimeout(loadWatchdog, 2500);
    }

    async function loadInterfaces() {
      try {
        const res = await fetch("/api/interfaces");
        const list = await res.json();
        const tbody = document.getElementById("ifacesTableBody");
        tbody.innerHTML = list.map(item => {
          const badge = item.active
            ? '<span class="badge ok">● Включен в Keenetic</span>'
            : '<span class="badge unknown">○ Не создан</span>';
          const btn = item.active
            ? '<button class="btn-danger" style="padding: 4px 10px;" onclick="manageIface(' + item.index + ', \'disable\')">Отключить</button>'
            : '<button class="btn-success" style="padding: 4px 10px;" onclick="manageIface(' + item.index + ', \'enable\')">Включить</button>';
          return '<tr>' +
            '<td style="font-weight: 600; color: var(--primary);">' + escapeHTML(item.name) + '</td>' +
            '<td>' + escapeHTML(item.description) + '</td>' +
            '<td style="font-family: monospace;">127.0.0.1:' + item.port + '</td>' +
            '<td>' + badge + '</td>' +
            '<td>' + btn + '</td>' +
            '</tr>';
        }).join('');
      } catch(e) { console.error(e); }
    }

    async function manageIface(index, action) {
      try {
        const res = await fetch("/api/interfaces/manage", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ index, action })
        });
        const data = await res.json();
        showAlert(data.message, data.success);
        loadInterfaces();
      } catch(e) { showAlert(e.message, false); }
    }

    async function doCoreAction(action) {
      try {
        const res = await fetch("/api/action", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ action })
        });
        const data = await res.json();
        showAlert(data.message, data.success);
        loadStatus();
      } catch(e) { showAlert(e.message, false); }
    }

    const HWID_PRESETS = {
      android: "User-Agent: LxBox/2.25.18/android\nX-Device-OS: Android\nX-HWID: e6ce21f5efaa40b99711058bd35ba999\nX-Ver-OS: 14\nX-Device-Model: Pixel 8\nX-Device-Locale: ru\nX-App-Version: 2.25.18",
      ios: "User-Agent: Happ/4.6.0/ios/1792157899\nX-Device-OS: iOS\nX-HWID: 4e03f677f4766c43\nX-Device-Locale: ru\nX-Ver-OS: 18.1\nX-App-Version: 4.6.0\nX-Device-Model: iPhone 15 Pro",
      windows: "X-HWID: e6ce21f5efaa40b99711058bd35ba999\nX_Device_OS: Windows\nX_Ver_OS: 11_10.0.26200\nX_Device_model: hsan_x86_64\nX_Device_Locale: RU"
    };

    async function loadDeviceProfile() {
      try {
        const res = await fetch("/api/device_profile");
        const p = await res.json();
        if (p) {
          const textarea = document.getElementById("hwidHeadersInput");
          if (!textarea.value || document.activeElement !== textarea) {
            textarea.value = p.raw_headers || "";
          }
          document.getElementById("dispProfileHWID").innerText = p.hwid || "—";
          document.getElementById("dispProfileOS").innerText = p.device_os || "—";
          document.getElementById("dispProfileModel").innerText = p.device_model || "—";
          document.getElementById("dispProfileLocale").innerText = p.device_locale || "—";
          document.getElementById("dispProfileVerOS").innerText = p.ver_os || "—";
          document.getElementById("dispProfileAppVer").innerText = p.app_version || "—";
          document.getElementById("dispProfileUA").innerText = p.user_agent || "—";
          document.getElementById("dispProfileUpdated").innerText = p.last_updated ? new Date(p.last_updated).toLocaleString() : "—";
          const count = Object.keys(p.headers || {}).length;
          document.getElementById("profileCountBadge").innerText = count + " заголовков";
        }
      } catch(e) { console.error(e); }
    }

    async function saveDeviceProfile() {
      const raw = document.getElementById("hwidHeadersInput").value.trim();
      if (!raw) {
        showAlert("Заголовки не могут быть пустыми!", false);
        return;
      }
      try {
        const res = await fetch("/api/device_profile", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ raw_headers: raw })
        });
        const data = await res.json();
        showAlert(data.message, data.success);
        loadDeviceProfile();
      } catch(e) { showAlert("Ошибка: " + e.message, false); }
    }

    function applyHwidPreset(type) {
      if (HWID_PRESETS[type]) {
        document.getElementById("hwidHeadersInput").value = HWID_PRESETS[type];
      }
    }

    function generateRandomHwid() {
      const hex = Array.from(crypto.getRandomValues(new Uint8Array(16)))
        .map(b => b.toString(16).padStart(2, '0')).join('');
      const el = document.getElementById("hwidHeadersInput");
      let val = el.value;
      const regex = /^(X[-_]HWID:\s*)(.*)$/im;
      if (regex.test(val)) {
        el.value = val.replace(regex, "$1" + hex);
      } else {
        el.value = "X-HWID: " + hex + "\n" + val;
      }
    }

    function escapeHTML(str) {
      if (!str) return '';
      return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
    }

    loadStatus();
    loadSubscriptions();
    loadDeviceProfile();
    setInterval(loadStatus, 5000);
    setInterval(() => {
      const activeTab = document.querySelector(".tab-content.active");
      if (activeTab && activeTab.id === "watchdogTab") loadWatchdog();
    }, 6000);
  </script>
</body>
</html>
`
