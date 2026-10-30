package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DeviceProfile represents client device headers and identification (HWID, OS, Model, User-Agent)
type DeviceProfile struct {
	RawHeaders   string            `json:"raw_headers"`
	Headers      map[string]string `json:"headers"`
	HWID         string            `json:"hwid"`
	UserAgent    string            `json:"user_agent"`
	DeviceOS     string            `json:"device_os"`
	DeviceModel  string            `json:"device_model"`
	DeviceLocale string            `json:"device_locale"`
	VerOS        string            `json:"ver_os"`
	AppVersion   string            `json:"app_version"`
	LastUpdated  time.Time         `json:"last_updated"`
}

const (
	PresetLxBoxAndroid = `User-Agent: LxBox/2.25.18/android
X-Device-OS: Android
X-HWID: e6ce21f5efaa40b99711058bd35ba999
X-Ver-OS: 14
X-Device-Model: Pixel 8
X-Device-Locale: ru
X-App-Version: 2.25.18`

	PresetHappIOS = `User-Agent: Happ/4.6.0/ios/1792157899
X-Device-OS: iOS
X-HWID: 4e03f677f4766c43
X-Device-Locale: ru
X-Ver-OS: 18.1
X-App-Version: 4.6.0
X-Device-Model: iPhone 15 Pro`

	PresetWindows = `X-HWID: e6ce21f5efaa40b99711058bd35ba999
X_Device_OS: Windows
X_Ver_OS: 11_10.0.26200
X_Device_model: hsan_x86_64
X_Device_Locale: RU`
)

var (
	globalProfileMu   sync.RWMutex
	globalProfile     *DeviceProfile
	deviceProfilePath = "/opt/etc/lxbox/device_profile.json"
)

// SetDeviceProfilePath overrides the storage location (e.g. for testing)
func SetDeviceProfilePath(path string) {
	globalProfileMu.Lock()
	defer globalProfileMu.Unlock()
	deviceProfilePath = path
}

// GenerateRandomHWID produces a random 32-character hexadecimal string
func GenerateRandomHWID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ParseDeviceHeaders parses multiline header input (supporting colon-separated lines with '-' or '_')
func ParseDeviceHeaders(raw string) *DeviceProfile {
	p := &DeviceProfile{
		RawHeaders:  strings.TrimSpace(raw),
		Headers:     make(map[string]string),
		LastUpdated: time.Now(),
	}

	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		if k == "" {
			continue
		}
		p.Headers[k] = v

		kLower := strings.ToLower(k)
		if strings.Contains(kLower, "hwid") {
			p.HWID = v
		} else if kLower == "user-agent" {
			p.UserAgent = v
		} else if strings.Contains(kLower, "device") && strings.Contains(kLower, "os") {
			p.DeviceOS = v
		} else if strings.Contains(kLower, "model") {
			p.DeviceModel = v
		} else if strings.Contains(kLower, "locale") {
			p.DeviceLocale = v
		} else if strings.Contains(kLower, "ver") && strings.Contains(kLower, "os") {
			p.VerOS = v
		} else if strings.Contains(kLower, "app") && strings.Contains(kLower, "version") {
			p.AppVersion = v
		}
	}

	if p.HWID == "" {
		p.HWID = "e6ce21f5efaa40b99711058bd35ba999"
	}
	if p.DeviceOS == "" {
		p.DeviceOS = "Android"
	}
	if p.DeviceModel == "" {
		p.DeviceModel = "LxBox-Router"
	}
	if p.UserAgent == "" {
		p.UserAgent = "LxBox/2.25.18/android"
	}

	return p
}

func getDeviceProfilePath() string {
	if env := os.Getenv("LXBOX_PROFILE_PATH"); env != "" {
		return env
	}
	return deviceProfilePath
}

// GetDeviceProfile returns the active in-memory device profile, loading from disk if necessary
func GetDeviceProfile() *DeviceProfile {
	globalProfileMu.Lock()
	defer globalProfileMu.Unlock()

	if globalProfile != nil {
		return globalProfile
	}

	path := getDeviceProfilePath()
	// Try reading from file
	if data, err := os.ReadFile(path); err == nil {
		var p DeviceProfile
		if err := json.Unmarshal(data, &p); err == nil && len(p.Headers) > 0 {
			globalProfile = &p
			return globalProfile
		}
	} else if path != "/tmp/device_profile.json" {
		if data, err := os.ReadFile("/tmp/device_profile.json"); err == nil {
			var p DeviceProfile
			if err := json.Unmarshal(data, &p); err == nil && len(p.Headers) > 0 {
				globalProfile = &p
				return globalProfile
			}
		}
	}

	// Fallback to default LxBox Android profile
	globalProfile = ParseDeviceHeaders(PresetLxBoxAndroid)
	_ = saveDeviceProfileLocked(globalProfile, path)
	return globalProfile
}

// UpdateDeviceProfile updates and persists the device profile
func UpdateDeviceProfile(raw string) (*DeviceProfile, error) {
	globalProfileMu.Lock()
	defer globalProfileMu.Unlock()

	p := ParseDeviceHeaders(raw)
	globalProfile = p
	path := getDeviceProfilePath()
	if err := saveDeviceProfileLocked(p, path); err != nil {
		return p, err
	}
	return p, nil
}

func saveDeviceProfileLocked(p *DeviceProfile, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Fallback for non-root environments (e.g. testing)
		if path != "/tmp/device_profile.json" {
			fallback := "/tmp/device_profile.json"
			return saveDeviceProfileLocked(p, fallback)
		}
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		if path != "/tmp/device_profile.json" {
			fallback := "/tmp/device_profile.json"
			return saveDeviceProfileLocked(p, fallback)
		}
		return err
	}
	return os.Rename(tmp, path)
}

// ApplyToRequest attaches the device identification headers and User-Agent to the outgoing HTTP request
func (p *DeviceProfile) ApplyToRequest(req *http.Request) {
	if p == nil {
		return
	}

	for k, v := range p.Headers {
		// Use direct slice assignment to preserve exact case and underscores (e.g. X_Device_OS)
		req.Header[k] = []string{v}
		// If key contains underscore, also provide dashed alternative if not already present
		if strings.Contains(k, "_") {
			dashed := strings.ReplaceAll(k, "_", "-")
			if _, exists := p.Headers[dashed]; !exists {
				req.Header[dashed] = []string{v}
			}
		}
	}

	if p.UserAgent != "" {
		req.Header["User-Agent"] = []string{p.UserAgent}
	}

	logMsg := fmt.Sprintf("Передача X-HWID: %s (OS: %s, Model: %s, заголовков: %d)",
		p.HWID, p.DeviceOS, p.DeviceModel, len(req.Header))
	fmt.Printf("[+] [HWID] %s\n", logMsg)

	// Forward event to live watchdog / web UI logs
	LogEvent("info", logMsg)
}
