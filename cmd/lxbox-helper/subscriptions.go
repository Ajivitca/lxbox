package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// NodeInfo represents an outbound server parsed from a subscription
type NodeInfo struct {
	Tag           string                 `json:"tag"`
	Type          string                 `json:"type"`
	Server        string                 `json:"server"`
	ServerPort    int                    `json:"server_port"`
	EnabledInPool bool                   `json:"enabled_in_pool"` // Included in watchdog failover pool
	IsActive      bool                   `json:"is_active"`
	Status        string                 `json:"status"` // "ok", "dead", "frozen", "unknown"
	LastLatencyMs int                    `json:"last_latency_ms"`
	LastChecked   time.Time              `json:"last_checked"`
	FailureCount  int                    `json:"failure_count"`
	FrozenUntil   time.Time              `json:"frozen_until"`
	RawOutbound   map[string]interface{} `json:"raw_outbound"`
}

// Subscription represents one provider subscription or group of links (up to 5 supported)
type Subscription struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	URL           string      `json:"url"` // http URL, happ://, or raw links
	Enabled       bool        `json:"enabled"`
	LastUpdated   time.Time   `json:"last_updated"`
	ProxyPort     int         `json:"proxy_port"`     // SOCKS5 port: 10808 (Master), 10809 (Sub 1), 10810 (Sub 2)...
	KeeneticIface string      `json:"keenetic_iface"` // e.g. "Proxy0", "Proxy1"...
	Nodes         []*NodeInfo `json:"nodes"`
}

// SubscriptionsManager manages persistent storage and config generation for up to 5 subscriptions
type SubscriptionsManager struct {
	mu            sync.RWMutex
	rebuildMu     sync.Mutex
	filePath      string
	configPath    string
	Subscriptions []*Subscription `json:"subscriptions"`
}

var (
	globalSubMgr *SubscriptionsManager
	subMgrOnce   sync.Once
)

func GetSubscriptionsManager(configPath string) *SubscriptionsManager {
	subMgrOnce.Do(func() {
		dir := filepath.Dir(configPath)
		storePath := filepath.Join(dir, "subscriptions.json")
		globalSubMgr = &SubscriptionsManager{
			filePath:      storePath,
			configPath:    configPath,
			Subscriptions: make([]*Subscription, 0),
		}
		_ = globalSubMgr.load()
	})
	return globalSubMgr
}

func (m *SubscriptionsManager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize empty store
			m.Subscriptions = make([]*Subscription, 0)
			return nil
		}
		return err
	}
	var loaded []*Subscription
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	m.Subscriptions = loaded
	return nil
}

func (m *SubscriptionsManager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.MarshalIndent(m.Subscriptions, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, m.filePath)
}

func (m *SubscriptionsManager) GetAll() []*Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]*Subscription, len(m.Subscriptions))
	copy(res, m.Subscriptions)
	return res
}

func (m *SubscriptionsManager) GetByID(id string) *Subscription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.Subscriptions {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// AddSubscription adds a new subscription (max 5 allowed)
func (m *SubscriptionsManager) AddSubscription(name, urlOrContent string) (*Subscription, error) {
	m.mu.Lock()
	if len(m.Subscriptions) >= 5 {
		m.mu.Unlock()
		return nil, fmt.Errorf("максимальное количество подписок (5) уже достигнуто")
	}
	m.mu.Unlock()

	// Parse nodes from urlOrContent
	obs, err := parseLink(urlOrContent)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга ссылки: %w", err)
	}
	if len(obs) == 0 {
		return nil, fmt.Errorf("в ссылке не обнаружено доступных узлов")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Assign sub ID: sub-1 .. sub-5
	subNum := 1
	for {
		candidateID := fmt.Sprintf("sub-%d", subNum)
		used := false
		for _, s := range m.Subscriptions {
			if s.ID == candidateID {
				used = true
				break
			}
		}
		if !used {
			break
		}
		subNum++
	}
	subID := fmt.Sprintf("sub-%d", subNum)

	nodes := make([]*NodeInfo, 0, len(obs))
	for i, o := range obs {
		tag, _ := o["tag"].(string)
		if tag == "" {
			tag = fmt.Sprintf("%s-node-%d", subID, i+1)
			o["tag"] = tag
		}
		tp, _ := o["type"].(string)
		srv, _ := o["server"].(string)
		var port int
		switch v := o["server_port"].(type) {
		case float64:
			port = int(v)
		case int:
			port = v
		}

		if tp == "urltest" {
			if childList, ok := o["outbounds"].([]string); ok {
				srv = fmt.Sprintf("Автовыбор из %d узлов (urltest 30s)", len(childList))
			} else {
				srv = "Автовыбор узлов (urltest 30s)"
			}
		}

		nodes = append(nodes, &NodeInfo{
			Tag:           tag,
			Type:          tp,
			Server:        srv,
			ServerPort:    port,
			EnabledInPool: true, // Selected by default
			IsActive:      i == 0 && len(m.Subscriptions) == 0,
			Status:        "unknown",
			RawOutbound:   o,
		})
	}

	proxyPort := 10808 + subNum // e.g. 10809 for sub-1, 10810 for sub-2
	sub := &Subscription{
		ID:            subID,
		Name:          strings.TrimSpace(name),
		URL:           strings.TrimSpace(urlOrContent),
		Enabled:       true,
		LastUpdated:   time.Now(),
		ProxyPort:     proxyPort,
		KeeneticIface: fmt.Sprintf("Proxy%d", subNum),
		Nodes:         nodes,
	}
	if sub.Name == "" {
		sub.Name = fmt.Sprintf("Подписка %d", subNum)
	}

	m.Subscriptions = append(m.Subscriptions, sub)

	// Persist
	data, _ := json.MarshalIndent(m.Subscriptions, "", "  ")
	_ = os.WriteFile(m.filePath, data, 0644)

	// Rebuild sing-box config
	go func() {
		_ = m.RebuildSingboxConfig()
	}()

	return sub, nil
}

// RefreshSubscription re-fetches and updates nodes for an existing subscription
func (m *SubscriptionsManager) RefreshSubscription(id string) error {
	m.mu.Lock()
	var targetSub *Subscription
	for _, s := range m.Subscriptions {
		if s.ID == id {
			targetSub = s
			break
		}
	}
	if targetSub == nil {
		m.mu.Unlock()
		return fmt.Errorf("подписка с ID %s не найдена", id)
	}
	subURL := targetSub.URL
	prevPoolMap := make(map[string]bool)
	prevActiveTag := ""
	for _, n := range targetSub.Nodes {
		prevPoolMap[n.Tag] = n.EnabledInPool
		if n.IsActive {
			prevActiveTag = n.Tag
		}
	}
	m.mu.Unlock()

	obs, err := parseLink(subURL)
	if err != nil {
		return fmt.Errorf("ошибка обновления подписки: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	newNodes := make([]*NodeInfo, 0, len(obs))
	for i, o := range obs {
		tag, _ := o["tag"].(string)
		if tag == "" {
			tag = fmt.Sprintf("%s-node-%d", id, i+1)
			o["tag"] = tag
		}
		tp, _ := o["type"].(string)
		srv, _ := o["server"].(string)
		var port int
		switch v := o["server_port"].(type) {
		case float64:
			port = int(v)
		case int:
			port = v
		}

		enabledInPool := true
		if val, exists := prevPoolMap[tag]; exists {
			enabledInPool = val
		}

		isActive := false
		if prevActiveTag == tag {
			isActive = true
		} else if prevActiveTag == "" && i == 0 {
			isActive = true
		}

		if tp == "urltest" {
			if childList, ok := o["outbounds"].([]string); ok {
				srv = fmt.Sprintf("Автовыбор из %d узлов (urltest 30s)", len(childList))
			} else {
				srv = "Автовыбор узлов (urltest 30s)"
			}
		}

		newNodes = append(newNodes, &NodeInfo{
			Tag:           tag,
			Type:          tp,
			Server:        srv,
			ServerPort:    port,
			EnabledInPool: enabledInPool,
			IsActive:      isActive,
			Status:        "unknown",
			RawOutbound:   o,
		})
	}

	targetSub.Nodes = newNodes
	targetSub.LastUpdated = time.Now()

	data, _ := json.MarshalIndent(m.Subscriptions, "", "  ")
	_ = os.WriteFile(m.filePath, data, 0644)

	go func() {
		_ = m.RebuildSingboxConfig()
	}()

	return nil
}

// DeleteSubscription removes a subscription
func (m *SubscriptionsManager) DeleteSubscription(id string) error {
	m.mu.Lock()
	idx := -1
	for i, s := range m.Subscriptions {
		if s.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		m.mu.Unlock()
		return fmt.Errorf("подписка с ID %s не найдена", id)
	}
	m.Subscriptions = append(m.Subscriptions[:idx], m.Subscriptions[idx+1:]...)

	data, _ := json.MarshalIndent(m.Subscriptions, "", "  ")
	_ = os.WriteFile(m.filePath, data, 0644)
	m.mu.Unlock()

	return m.RebuildSingboxConfig()
}

// ToggleNodePool toggles whether a node participates in the failover/monitoring pool
func (m *SubscriptionsManager) ToggleNodePool(subID, nodeTag string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.Subscriptions {
		if s.ID == subID {
			for _, n := range s.Nodes {
				if n.Tag == nodeTag {
					n.EnabledInPool = enabled
					data, _ := json.MarshalIndent(m.Subscriptions, "", "  ")
					_ = os.WriteFile(m.filePath, data, 0644)

					go func() {
						_ = m.RebuildSingboxConfig()
					}()
					return nil
				}
			}
		}
	}
	return fmt.Errorf("узел %s в подписке %s не найден", nodeTag, subID)
}

// SetAllNodesInPool sets EnabledInPool for all nodes in a subscription
func (m *SubscriptionsManager) SetAllNodesInPool(subID string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.Subscriptions {
		if s.ID == subID {
			for _, n := range s.Nodes {
				n.EnabledInPool = enabled
			}
			data, _ := json.MarshalIndent(m.Subscriptions, "", "  ")
			_ = os.WriteFile(m.filePath, data, 0644)

			go func() {
				_ = m.RebuildSingboxConfig()
			}()
			return nil
		}
	}
	return fmt.Errorf("подписка %s не найдена", subID)
}

// SetActiveNode marks a node as active in its subscription and in the master pool
func (m *SubscriptionsManager) SetActiveNode(subID, nodeTag string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.Subscriptions {
		for _, n := range s.Nodes {
			n.IsActive = (s.ID == subID && n.Tag == nodeTag)
		}
	}
	data, _ := json.MarshalIndent(m.Subscriptions, "", "  ")
	_ = os.WriteFile(m.filePath, data, 0644)

	// Send switch command to Clash API
	go func() {
		clashURL := "http://127.0.0.1:9090/proxies/select"
		body := fmt.Sprintf(`{"name": "%s"}`, nodeTag)
		req, _ := http.NewRequest(http.MethodPut, clashURL, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 3 * time.Second}
		_, _ = client.Do(req)
	}()

	return nil
}

// RebuildSingboxConfig generates a comprehensive sing-box config including:
// - Mixed inbound on 7890
// - Master SOCKS5 inbound on 10808 (routes to master failover selector)
// - Sub-specific SOCKS5 inbounds on 10809..10813 (routes to respective subscription selectors)
// - Outbounds from all enabled subscriptions
// - Master selector group containing all nodes marked EnabledInPool
// - Subscription-specific selector groups
func (m *SubscriptionsManager) RebuildSingboxConfig() error {
	m.rebuildMu.Lock()
	defer m.rebuildMu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := ensureConfigTemplate(m.configPath); err != nil {
		return err
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", m.configPath, err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("invalid json in %s: %w", m.configPath, err)
	}

	// Build Inbounds
	inbounds := []interface{}{
		map[string]interface{}{
			"type":        "mixed",
			"tag":         "mixed-in",
			"listen":      "0.0.0.0",
			"listen_port": 7890,
		},
		map[string]interface{}{
			"type":        "socks",
			"tag":         "socks-in",
			"listen":      "0.0.0.0",
			"listen_port": 10808,
		},
	}

	for _, s := range m.Subscriptions {
		if s.Enabled && s.ProxyPort > 0 {
			inbounds = append(inbounds, map[string]interface{}{
				"type":        "socks",
				"tag":         fmt.Sprintf("%s-in", s.ID),
				"listen":      "0.0.0.0",
				"listen_port": s.ProxyPort,
			})
		}
	}
	cfg["inbounds"] = inbounds

	// Ensure standard modern DNS config
	cfg["dns"] = map[string]interface{}{
		"servers": []interface{}{
			map[string]interface{}{
				"tag":     "remote-dns",
				"address": "tls://1.1.1.1",
				"detour":  "select",
			},
			map[string]interface{}{
				"tag":     "local-dns",
				"address": "local",
				"detour":  "direct",
			},
		},
		"final":    "local-dns",
		"strategy": "prefer_ipv4",
	}

	// Collect pool nodes and sub nodes
	var masterPoolTags []string
	seenMaster := make(map[string]bool)
	subSelectors := make(map[string][]string)
	allOutboundsMap := make(map[string]map[string]interface{})

	var activeNodeTag string

	for _, s := range m.Subscriptions {
		if !s.Enabled {
			continue
		}
		var subTags []string
		seenSub := make(map[string]bool)
		for _, n := range s.Nodes {
			tag := n.Tag
			if tag == "" || n.RawOutbound == nil {
				continue
			}
			tp, _ := n.RawOutbound["type"].(string)
			if tp == "loopback" || tp == "direct" || tp == "block" || tp == "dns" {
				continue
			}

			if !seenSub[tag] {
				seenSub[tag] = true
				subTags = append(subTags, tag)
			}
			allOutboundsMap[tag] = n.RawOutbound

			if n.EnabledInPool && !seenMaster[tag] {
				seenMaster[tag] = true
				masterPoolTags = append(masterPoolTags, tag)
			}
			if n.IsActive {
				activeNodeTag = tag
			}
		}
		subSelectors[s.ID] = subTags
	}

	if len(masterPoolTags) == 0 {
		masterPoolTags = append(masterPoolTags, "direct")
	}

	defaultTag := "direct"
	if activeNodeTag != "" && seenMaster[activeNodeTag] {
		defaultTag = activeNodeTag
	} else if len(masterPoolTags) > 0 && masterPoolTags[0] != "direct" {
		defaultTag = masterPoolTags[0]
	}

	// Build Outbounds list
	outbounds := []interface{}{
		// Master Selector
		map[string]interface{}{
			"type":      "selector",
			"tag":       "select",
			"outbounds": masterPoolTags,
			"default":   defaultTag,
		},
	}

	// Add sub-specific selectors
	for _, s := range m.Subscriptions {
		if !s.Enabled {
			continue
		}
		tags := subSelectors[s.ID]
		if len(tags) == 0 {
			tags = []string{"direct"}
		}
		outbounds = append(outbounds, map[string]interface{}{
			"type":      "selector",
			"tag":       fmt.Sprintf("%s-select", s.ID),
			"outbounds": tags,
			"default":   tags[0],
		})
	}

	// Add standard outbounds
	outbounds = append(outbounds,
		map[string]interface{}{"type": "direct", "tag": "direct"},
		map[string]interface{}{"type": "block", "tag": "block"},
		map[string]interface{}{"type": "dns", "tag": "dns-out"},
	)

	// Add all raw node outbounds
	for _, raw := range allOutboundsMap {
		outbounds = append(outbounds, raw)
	}
	cfg["outbounds"] = outbounds

	// Build Route rules
	rules := []interface{}{
		map[string]interface{}{
			"protocol": "dns",
			"outbound": "dns-out",
		},
		map[string]interface{}{
			"ip_is_private": true,
			"outbound":      "direct",
		},
	}

	// Route each sub-inbound to its respective sub-selector
	for _, s := range m.Subscriptions {
		if s.Enabled && s.ProxyPort > 0 {
			rules = append(rules, map[string]interface{}{
				"inbound":  []string{fmt.Sprintf("%s-in", s.ID)},
				"outbound": fmt.Sprintf("%s-select", s.ID),
			})
		}
	}

	cfg["route"] = map[string]interface{}{
		"auto_detect_interface": true,
		"rules":                 rules,
		"final":                 "select",
	}

	outBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := m.configPath + ".tmp"
	if err := os.WriteFile(tmpFile, outBytes, 0644); err != nil {
		return err
	}

	// Test configuration with sing-box check if binary is available
	if _, err := exec.LookPath("sing-box"); err == nil {
		cmd := exec.Command("sing-box", "check", "-c", tmpFile)
		if out, err := cmd.CombinedOutput(); err != nil {
			_ = os.Remove(tmpFile)
			return fmt.Errorf("проверка sing-box не прошла: %s", string(out))
		}
	}

	if err := os.Rename(tmpFile, m.configPath); err != nil {
		return err
	}

	// Reload sing-box
	_ = restartSingboxService()
	return nil
}
