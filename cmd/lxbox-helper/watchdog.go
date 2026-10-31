package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// WatchdogConfig holds configurable parameters for the health check service
type WatchdogConfig struct {
	Enabled           bool     `json:"enabled"`
	ActiveIntervalSec int      `json:"active_interval_sec"` // Check active node every N seconds (e.g. 30s)
	PoolIntervalSec   int      `json:"pool_interval_sec"`   // Check candidate pool every N seconds (e.g. 300s)
	TestURLs          []string `json:"test_urls"`           // Target URLs: ["https://instagram.com", "https://x.com", "https://grok.ai"]
	PingTarget        string   `json:"ping_target"`         // e.g. "9.9.9.9:53"
	MaxFailures       int      `json:"max_failures"`        // Max failures before failover (e.g. 3)
	RetryPauseSec     int      `json:"retry_pause_sec"`     // Pause between retries (e.g. 3s)
	FreezeHours       int      `json:"freeze_hours"`        // Hours to freeze failed server (e.g. 2h)
	ProxyAddr         string   `json:"proxy_addr"`          // Local proxy address, default "127.0.0.1:7890"
	ClashAPI          string   `json:"clash_api"`           // Clash API address, default "http://127.0.0.1:9090"
}

func DefaultWatchdogConfig() WatchdogConfig {
	return WatchdogConfig{
		Enabled:           true,
		ActiveIntervalSec: 30,
		PoolIntervalSec:   300,
		TestURLs: []string{
			"https://instagram.com",
			"https://x.com",
			"https://grok.ai",
		},
		PingTarget:    "9.9.9.9:53",
		MaxFailures:   3,
		RetryPauseSec: 3,
		FreezeHours:   2,
		ProxyAddr:     "127.0.0.1:7890",
		ClashAPI:      "http://127.0.0.1:9090",
	}
}

// WatchdogLogEntry represents a recorded event
type WatchdogLogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"` // "info", "warn", "error", "success"
	Message   string `json:"message"`
}

// WatchdogService runs the background watchdog loop
type WatchdogService struct {
	mu         sync.RWMutex
	cfg        WatchdogConfig
	cfgPath    string
	subMgr     *SubscriptionsManager
	stopCh     chan struct{}
	running    bool
	logs       []WatchdogLogEntry
	activeFail int
}

var (
	globalWatchdog   *WatchdogService
	watchdogInitOnce sync.Once
)

func GetWatchdogService(configDir string, subMgr *SubscriptionsManager) *WatchdogService {
	watchdogInitOnce.Do(func() {
		cfgFile := filepath.Join(configDir, "watchdog.json")
		cfg := DefaultWatchdogConfig()
		if data, err := os.ReadFile(cfgFile); err == nil {
			_ = json.Unmarshal(data, &cfg)
		}
		globalWatchdog = &WatchdogService{
			cfg:     cfg,
			cfgPath: cfgFile,
			subMgr:  subMgr,
			logs:    make([]WatchdogLogEntry, 0),
		}
	})
	return globalWatchdog
}

func (w *WatchdogService) log(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	entry := WatchdogLogEntry{
		Timestamp: time.Now().Format("15:04:05"),
		Level:     level,
		Message:   msg,
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logs = append(w.logs, entry)
	if len(w.logs) > 60 {
		w.logs = w.logs[len(w.logs)-60:]
	}
	fmt.Printf("[Watchdog %s] %s\n", strings.ToUpper(level), msg)
}

// LogEvent writes a log message to the active watchdog event stream
func LogEvent(level, format string, args ...interface{}) {
	if globalWatchdog != nil {
		globalWatchdog.log(level, format, args...)
	} else {
		fmt.Printf("[%s] %s\n", strings.ToUpper(level), fmt.Sprintf(format, args...))
	}
}

func (w *WatchdogService) GetStatus() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Find current active node
	activeTag := "none"
	activeStatus := "unknown"
	activeLatency := 0
	subs := w.subMgr.GetAll()
	for _, s := range subs {
		for _, n := range s.Nodes {
			if n.IsActive {
				activeTag = n.Tag
				activeStatus = n.Status
				activeLatency = n.LastLatencyMs
				break
			}
		}
	}

	return map[string]interface{}{
		"enabled":             w.cfg.Enabled,
		"running":             w.running,
		"active_node":         activeTag,
		"active_status":       activeStatus,
		"active_latency":      activeLatency,
		"active_fail_count":   w.activeFail,
		"config":              w.cfg,
		"logs":                w.logs,
	}
}

func (w *WatchdogService) UpdateConfig(newCfg WatchdogConfig) error {
	w.mu.Lock()
	w.cfg = newCfg
	data, err := json.MarshalIndent(w.cfg, "", "  ")
	if err == nil {
		_ = os.WriteFile(w.cfgPath, data, 0644)
	}
	w.mu.Unlock()

	w.log("info", "Параметры ватчдога обновлены (проверка: %dс, заморозка: %dч)", newCfg.ActiveIntervalSec, newCfg.FreezeHours)
	return nil
}

func (w *WatchdogService) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	w.log("info", "Ватчдог запущен (интервал активного узла: %d сек, пул: %d сек)", w.cfg.ActiveIntervalSec, w.cfg.PoolIntervalSec)
	go w.activeLoop()
	go w.poolLoop()
}

func (w *WatchdogService) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	close(w.stopCh)
	w.mu.Unlock()
	w.log("info", "Ватчдог остановлен")
}

// activeLoop periodically checks the currently active node
func (w *WatchdogService) activeLoop() {
	ticker := time.NewTicker(time.Duration(w.cfg.ActiveIntervalSec) * time.Second)
	defer ticker.Stop()

	// Initial delay on startup
	time.Sleep(5 * time.Second)

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.mu.RLock()
			enabled := w.cfg.Enabled
			interval := w.cfg.ActiveIntervalSec
			w.mu.RUnlock()

			if !enabled {
				continue
			}
			ticker.Reset(time.Duration(interval) * time.Second)
			w.CheckActiveNode()
		}
	}
}

// poolLoop periodically checks the health and latency of all candidate nodes
func (w *WatchdogService) poolLoop() {
	ticker := time.NewTicker(time.Duration(w.cfg.PoolIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.mu.RLock()
			enabled := w.cfg.Enabled
			interval := w.cfg.PoolIntervalSec
			w.mu.RUnlock()

			if !enabled {
				continue
			}
			ticker.Reset(time.Duration(interval) * time.Second)
			w.CheckPoolNodes()
		}
	}
}

// testProxyHTTP performs HTTP GET through local proxy to test URLs
func (w *WatchdogService) testProxyHTTP() (bool, int, string) {
	w.mu.RLock()
	proxyAddr := w.cfg.ProxyAddr
	testURLs := w.cfg.TestURLs
	w.mu.RUnlock()

	if len(testURLs) == 0 {
		testURLs = []string{"https://cp.cloudflare.com/generate_204"}
	}

	proxyURL, err := url.Parse("http://" + proxyAddr)
	if err != nil {
		return false, 0, err.Error()
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   7 * time.Second,
	}

	// Try each test URL until one responds
	for _, target := range testURLs {
		start := time.Now()
		resp, err := client.Get(target)
		latency := int(time.Since(start).Milliseconds())

		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode > 0 && resp.StatusCode < 500 {
				return true, latency, target
			}
		}
	}

	return false, 0, "all test urls timed out"
}

// probeNodeClashDelay probes any specific node through sing-box Clash API
func (w *WatchdogService) probeNodeClashDelay(nodeTag string) (bool, int) {
	w.mu.RLock()
	apiBase := w.cfg.ClashAPI
	w.mu.RUnlock()

	testURL := "https://cp.cloudflare.com/generate_204"
	probeURL := fmt.Sprintf("%s/proxies/%s/delay?timeout=5000&url=%s",
		apiBase, url.PathEscape(nodeTag), url.QueryEscape(testURL))

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Get(probeURL)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var res struct {
			Delay int `json:"delay"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && res.Delay > 0 {
			return true, res.Delay
		}
	}
	return false, 0
}

// CheckActiveNode executes the full check on the active server with retries and failover
func (w *WatchdogService) CheckActiveNode() {
	// Find currently active node
	var activeNode *NodeInfo
	var activeSub *Subscription
	subs := w.subMgr.GetAll()

	for _, s := range subs {
		for _, n := range s.Nodes {
			if n.IsActive {
				activeNode = n
				activeSub = s
				break
			}
		}
		if activeNode != nil {
			break
		}
	}

	if activeNode == nil {
		// No node active; select first available from pool
		w.triggerFailover(nil, nil, "нет активного узла")
		return
	}

	// 1. Probe through local proxy
	ok, latency, target := w.testProxyHTTP()
	if ok {
		w.activeFail = 0
		activeNode.Status = "ok"
		activeNode.LastLatencyMs = latency
		activeNode.LastChecked = time.Now()
		activeNode.FailureCount = 0
		_ = w.subMgr.Save()
		return
	}

	// First attempt failed — retry with pause
	w.activeFail++
	w.log("warn", "Активный узел '%s' не ответил на HTTP-пинг (%s). Попытка %d/%d...",
		activeNode.Tag, target, w.activeFail, w.cfg.MaxFailures)

	if w.activeFail < w.cfg.MaxFailures {
		time.Sleep(time.Duration(w.cfg.RetryPauseSec) * time.Second)
		// Retry once more within this check
		retryOK, retryLat, _ := w.testProxyHTTP()
		if retryOK {
			w.activeFail = 0
			activeNode.Status = "ok"
			activeNode.LastLatencyMs = retryLat
			activeNode.LastChecked = time.Now()
			_ = w.subMgr.Save()
			w.log("info", "Узел '%s' восстановил соединение при повторной проверке (%d ms)", activeNode.Tag, retryLat)
			return
		}
	}

	if w.activeFail >= w.cfg.MaxFailures {
		// Active node is confirmed DEAD
		w.triggerFailover(activeSub, activeNode, "превышен лимит неудачных проверок")
	}
}

// triggerFailover freezes current server and switches to best verified live candidate
func (w *WatchdogService) triggerFailover(failedSub *Subscription, failedNode *NodeInfo, reason string) {
	w.mu.RLock()
	freezeHours := w.cfg.FreezeHours
	w.mu.RUnlock()

	if failedNode != nil {
		failedNode.Status = "frozen"
		failedNode.FailureCount = w.cfg.MaxFailures
		failedNode.FrozenUntil = time.Now().Add(time.Duration(freezeHours) * time.Hour)
		w.log("error", "Узел '%s' признан неработающим (%s). Заморожен на %d ч до %s",
			failedNode.Tag, reason, freezeHours, failedNode.FrozenUntil.Format("15:04:05"))
		_ = w.subMgr.Save()
	}

	w.log("info", "Поиск резервного сервера среди выбранных в пуле...")

	// Collect all candidates with EnabledInPool == true and not frozen
	type Candidate struct {
		sub  *Subscription
		node *NodeInfo
	}
	var candidates []Candidate
	now := time.Now()

	subs := w.subMgr.GetAll()
	for _, s := range subs {
		if !s.Enabled {
			continue
		}
		for _, n := range s.Nodes {
			if failedNode != nil && n.Tag == failedNode.Tag {
				continue
			}
			if n.EnabledInPool && (n.FrozenUntil.IsZero() || n.FrozenUntil.Before(now)) {
				candidates = append(candidates, Candidate{sub: s, node: n})
			}
		}
	}

	if len(candidates) == 0 {
		w.log("error", "ВНИМАНИЕ: Нет доступных кандидатов в пуле (все заморожены или отключены). Сброс заморозки...")
		// Emergency: reset freeze on all nodes
		for _, s := range subs {
			for _, n := range s.Nodes {
				n.FrozenUntil = time.Time{}
				if n.EnabledInPool && (failedNode == nil || n.Tag != failedNode.Tag) {
					candidates = append(candidates, Candidate{sub: s, node: n})
				}
			}
		}
	}

	// Sort candidates by lowest last known latency
	sort.Slice(candidates, func(i, j int) bool {
		l1 := candidates[i].node.LastLatencyMs
		l2 := candidates[j].node.LastLatencyMs
		if l1 <= 0 {
			l1 = 9999
		}
		if l2 <= 0 {
			l2 = 9999
		}
		return l1 < l2
	})

	// "перед переключением сначала проверяем так же на жив/мертв выбранный сервер"
	for _, cand := range candidates {
		w.log("info", "Проверка кандидата '%s' на жив/мертв...", cand.node.Tag)
		alive, lat := w.probeNodeClashDelay(cand.node.Tag)
		if alive {
			// Candidate is ALIVE! Switch!
			w.log("success", "Кандидат '%s' УСПЕШНО проверен (задержка: %d ms). Переключение трафика...",
				cand.node.Tag, lat)

			cand.node.Status = "ok"
			cand.node.LastLatencyMs = lat
			cand.node.LastChecked = time.Now()
			cand.node.FailureCount = 0

			_ = w.subMgr.SetActiveNode(cand.sub.ID, cand.node.Tag)
			w.activeFail = 0

			// Switch via Clash API
			w.switchClashProxy(cand.node.Tag)

			w.log("success", "✔ Переключение завершено! Активный узел: '%s' (%s)", cand.node.Tag, cand.sub.Name)
			return
		}

		// Candidate failed probe
		cand.node.Status = "dead"
		cand.node.FrozenUntil = time.Now().Add(time.Duration(freezeHours) * time.Hour)
		w.log("warn", "Кандидат '%s' недоступен. Заморозка и проверка следующего...", cand.node.Tag)
		_ = w.subMgr.Save()
	}

	w.log("error", "Все кандидаты в пуле оказались недоступны! Перезапуск службы sing-box...")
	_ = restartSingboxService()
}

func (w *WatchdogService) switchClashProxy(nodeTag string) {
	w.mu.RLock()
	apiBase := w.cfg.ClashAPI
	w.mu.RUnlock()

	reqURL := fmt.Sprintf("%s/proxies/select", apiBase)
	body := fmt.Sprintf(`{"name": "%s"}`, nodeTag)
	req, err := http.NewRequest(http.MethodPut, reqURL, strings.NewReader(body))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 3 * time.Second}
		_, _ = client.Do(req)
	}
}

// CheckPoolNodes checks latency and alive/dead state for all pool nodes
func (w *WatchdogService) CheckPoolNodes() {
	now := time.Now()
	subs := w.subMgr.GetAll()

	for _, s := range subs {
		if !s.Enabled {
			continue
		}
		for _, n := range s.Nodes {
			// Unfreeze expired
			if !n.FrozenUntil.IsZero() && n.FrozenUntil.Before(now) {
				n.FrozenUntil = time.Time{}
				if n.Status == "frozen" {
					n.Status = "unknown"
				}
			}

			if !n.EnabledInPool || n.IsActive {
				continue
			}

			alive, lat := w.probeNodeClashDelay(n.Tag)
			if alive {
				n.Status = "ok"
				n.LastLatencyMs = lat
				n.LastChecked = now
			} else {
				if n.Status == "ok" {
					n.Status = "dead"
				}
			}
		}
	}
	_ = w.subMgr.Save()
}
