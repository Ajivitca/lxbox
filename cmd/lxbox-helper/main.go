package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var pkcs1Keys = []string{
	// key[0] — happ://crypt/ (RSA-1024)
	"MIICXwIBAAKBgQCxsS7PUq1biQlVD92rf6eXKr9oG1/SrYx3qWahZP+Jq35m4Wb/Z+mB6eBWrPzJ/zZpZLWLQorcvOKt+sLaCHyH1HLNkti4jlaEQX6x97XgBm8GK08+lLLWquFDhWRNxsrfzJyNdpVopzBRmCJKTc8ObYyPbrv9T35a8Kd5WqjnUwIDAQABAoGBAJoqe85skPPF5U7jwRM2YhUJhZ+xgGWtJR3834pPslWjcLuZ/F7DrRiF7ZnF5FztDCxMsCXuycPSLWl9EulQS5mrL/fnwpK2jVE8O1Em9RsBOOrWwzuZnAuooRIb/8zC0fvH2oGkk60zSKycMe69uvYUDjhvULX2Spjmf9CS9/HhAkEA3I797En/DrpAZz6NM4GqZ1mkH0kEX/kAHLP1lBgYL1kVK455EG/ecJkMJmtK7A+fWw0N0IcxrpYAbbOAo19vjwJBAM4+0MAZ8TIZUk6Rs2gYUo04A6mYUy5MWtRa9pyFIgD71oHDR+1jrnPLqQyCj0tfbZBc1iVgsisJBpocC8sKaf0CQQDRNd3Mxb/nY2p1xJLBmaxezlvsxSEePB4MG/PFXzmJqBF5uHJD0imIWtR4mOt/ka4R+wbwl1zcAzMy28MYtQ0nAkEAuUILWML0uL+uAw01TeerH1aVU52T+h5z6BPdOTMNHD0arWywCzhi13i03JvaAyYw0F/Tq7dz0txEpeFTZopwMQJBANnHbzB87/xTjDQA4/L8sSU8m0vM1nRWmJIaAC94pcM+KDGLnbBhWrvZGy8Zg8vQwNvdvCLvylk0jVTTFqW3ibM=",
	// key[1] — happ://crypt2/ (RSA-4096)
	"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",
	// key[2] — happ://crypt3/ (RSA-4096)
	"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",
	// key[3] — happ://crypt4/ (RSA-4096)
	"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",
}

func parsePKCS1(keyB64 string) (*rsa.PrivateKey, error) {
	pemStr := fmt.Sprintf("-----BEGIN RSA PRIVATE KEY-----\n%s\n-----END RSA PRIVATE KEY-----", keyB64)
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func b64DecodeURLSafe(s string) ([]byte, error) {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	for len(s)%4 != 0 {
		s += "="
	}
	return base64.StdEncoding.DecodeString(s)
}

func decryptHappRSA(ordinal int, payload string) (string, error) {
	if ordinal < 0 || ordinal >= len(pkcs1Keys) {
		return "", fmt.Errorf("unsupported crypt version index: %d", ordinal)
	}
	key, err := parsePKCS1(pkcs1Keys[ordinal])
	if err != nil {
		return "", fmt.Errorf("parse key failed: %w", err)
	}
	cipherBytes, err := b64DecodeURLSafe(payload)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	keySize := key.Size()
	var plaintext strings.Builder
	for i := 0; i < len(cipherBytes); i += keySize {
		end := i + keySize
		if end > len(cipherBytes) {
			end = len(cipherBytes)
		}
		part, err := rsa.DecryptPKCS1v15(nil, key, cipherBytes[i:end])
		if err != nil {
			return "", fmt.Errorf("rsa decrypt chunk failed: %w", err)
		}
		plaintext.Write(part)
	}
	return plaintext.String(), nil
}

func decryptHapp(link string) (string, error) {
	path := strings.TrimPrefix(link, "happ://")
	if strings.HasPrefix(path, "crypt/") {
		return decryptHappRSA(0, strings.TrimPrefix(path, "crypt/"))
	} else if strings.HasPrefix(path, "crypt2/") {
		return decryptHappRSA(1, strings.TrimPrefix(path, "crypt2/"))
	} else if strings.HasPrefix(path, "crypt3/") {
		return decryptHappRSA(2, strings.TrimPrefix(path, "crypt3/"))
	} else if strings.HasPrefix(path, "crypt4/") {
		return decryptHappRSA(3, strings.TrimPrefix(path, "crypt4/"))
	}
	return "", fmt.Errorf("unsupported or unrecognized happ scheme: %s", link)
}

// Outbound represents a sing-box outbound config structure
type Outbound map[string]interface{}

func parseVLESS(raw string) (Outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid vless url: %w", err)
	}
	uuid := u.User.Username()
	server := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "443"
	}
	port, _ := strconv.Atoi(portStr)
	q := u.Query()
	name := u.Fragment
	if name == "" {
		name = server
	}

	ob := Outbound{
		"type":        "vless",
		"tag":         name,
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
	}

	if flow := q.Get("flow"); flow != "" {
		ob["flow"] = flow
	}

	sec := q.Get("security")
	if sec == "reality" || sec == "tls" {
		tls := map[string]interface{}{
			"enabled":     true,
			"server_name": q.Get("sni"),
		}
		if fp := q.Get("fp"); fp != "" {
			tls["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": fp,
			}
		}
		if sec == "reality" {
			tls["reality"] = map[string]interface{}{
				"enabled":    true,
				"public_key": q.Get("pbk"),
				"short_id":   q.Get("sid"),
			}
		}
		ob["tls"] = tls
	}

	transportType := q.Get("type")
	if transportType != "" && transportType != "tcp" {
		tr := map[string]interface{}{
			"type": transportType,
		}
		if path := q.Get("path"); path != "" {
			tr["path"] = path
		}
		if host := q.Get("host"); host != "" {
			tr["headers"] = map[string]interface{}{
				"Host": host,
			}
		}
		ob["transport"] = tr
	}

	return ob, nil
}

func parseHysteria2(raw string) (Outbound, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid hysteria2 url: %w", err)
	}
	password := u.User.Username()
	server := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "443"
	}
	port, _ := strconv.Atoi(portStr)
	name := u.Fragment
	if name == "" {
		name = server
	}
	q := u.Query()

	ob := Outbound{
		"type":        "hysteria2",
		"tag":         name,
		"server":      server,
		"server_port": port,
		"password":    password,
	}
	tls := map[string]interface{}{
		"enabled": true,
	}
	if sni := q.Get("sni"); sni != "" {
		tls["server_name"] = sni
	}
	if q.Get("insecure") == "1" {
		tls["insecure"] = true
	}
	ob["tls"] = tls
	return ob, nil
}

func getInt(v interface{}, def int) int {
	if v == nil {
		return def
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return def
}

func tryBase64Decode(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return string(b), true
	}
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return string(b), true
	}
	if b, err := base64.URLEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return string(b), true
	}
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return string(b), true
	}
	return "", false
}

func convertXrayOutboundToSingbox(xrayOb map[string]interface{}, defaultTag string) (Outbound, error) {
	proto, _ := xrayOb["protocol"].(string)
	proto = strings.ToLower(proto)
	tag, _ := xrayOb["tag"].(string)
	if defaultTag != "" {
		tag = defaultTag
	}
	if tag == "" {
		tag = proto
	}

	allowed := map[string]bool{
		"vless": true, "vmess": true, "trojan": true, "shadowsocks": true,
		"hysteria": true, "hysteria2": true, "hy2": true, "socks": true,
		"http": true, "wireguard": true, "tuic": true,
	}
	if !allowed[proto] {
		return nil, fmt.Errorf("non-proxy protocol: %s", proto)
	}

	settings, _ := xrayOb["settings"].(map[string]interface{})
	streamSettings, _ := xrayOb["streamSettings"].(map[string]interface{})

	ob := Outbound{
		"tag": tag,
	}

	switch proto {
	case "vless":
		ob["type"] = "vless"
		vnextList, _ := settings["vnext"].([]interface{})
		if len(vnextList) == 0 {
			return nil, fmt.Errorf("vless has no vnext")
		}
		vnext, _ := vnextList[0].(map[string]interface{})
		addr, _ := vnext["address"].(string)
		port := getInt(vnext["port"], 443)
		ob["server"] = addr
		ob["server_port"] = port

		usersList, _ := vnext["users"].([]interface{})
		if len(usersList) > 0 {
			u, _ := usersList[0].(map[string]interface{})
			id, _ := u["id"].(string)
			ob["uuid"] = id
			if flow, ok := u["flow"].(string); ok && flow != "" {
				ob["flow"] = flow
			}
		}

	case "vmess":
		ob["type"] = "vmess"
		vnextList, _ := settings["vnext"].([]interface{})
		if len(vnextList) == 0 {
			return nil, fmt.Errorf("vmess has no vnext")
		}
		vnext, _ := vnextList[0].(map[string]interface{})
		addr, _ := vnext["address"].(string)
		port := getInt(vnext["port"], 443)
		ob["server"] = addr
		ob["server_port"] = port

		usersList, _ := vnext["users"].([]interface{})
		if len(usersList) > 0 {
			u, _ := usersList[0].(map[string]interface{})
			id, _ := u["id"].(string)
			ob["uuid"] = id
			alterId := getInt(u["alterId"], 0)
			ob["alter_id"] = alterId
			sec, _ := u["security"].(string)
			if sec == "" {
				sec = "auto"
			}
			ob["security"] = sec
		}

	case "trojan":
		ob["type"] = "trojan"
		serversList, _ := settings["servers"].([]interface{})
		if len(serversList) == 0 {
			return nil, fmt.Errorf("trojan has no servers")
		}
		srv, _ := serversList[0].(map[string]interface{})
		addr, _ := srv["address"].(string)
		port := getInt(srv["port"], 443)
		ob["server"] = addr
		ob["server_port"] = port
		pwd, _ := srv["password"].(string)
		ob["password"] = pwd

	case "shadowsocks":
		ob["type"] = "shadowsocks"
		serversList, _ := settings["servers"].([]interface{})
		if len(serversList) == 0 {
			return nil, fmt.Errorf("shadowsocks has no servers")
		}
		srv, _ := serversList[0].(map[string]interface{})
		addr, _ := srv["address"].(string)
		port := getInt(srv["port"], 8388)
		ob["server"] = addr
		ob["server_port"] = port
		method, _ := srv["method"].(string)
		pwd, _ := srv["password"].(string)
		ob["method"] = method
		ob["password"] = pwd

	case "hysteria", "hysteria2", "hy2":
		ob["type"] = proto
		if srv, ok := settings["address"].(string); ok {
			ob["server"] = srv
		}
		if prt, ok := settings["port"]; ok {
			ob["server_port"] = getInt(prt, 443)
		}
		if auth, ok := settings["auth_str"].(string); ok {
			ob["password"] = auth
		}
		ob["up_mbps"] = 100
		ob["down_mbps"] = 100

	default:
		ob["type"] = proto
		if srv, ok := settings["address"].(string); ok {
			ob["server"] = srv
		}
		if prt, ok := settings["port"]; ok {
			ob["server_port"] = getInt(prt, 443)
		}
	}

	if streamSettings != nil {
		net, _ := streamSettings["network"].(string)
		sec, _ := streamSettings["security"].(string)

		switch strings.ToLower(net) {
		case "ws":
			wsMap := map[string]interface{}{"type": "ws"}
			if wsSettings, ok := streamSettings["wsSettings"].(map[string]interface{}); ok {
				if p, ok := wsSettings["path"].(string); ok {
					wsMap["path"] = p
				}
				if h, ok := wsSettings["headers"].(map[string]interface{}); ok {
					wsMap["headers"] = h
				}
			}
			ob["transport"] = wsMap
		case "grpc":
			grpcMap := map[string]interface{}{"type": "grpc"}
			if grpcSettings, ok := streamSettings["grpcSettings"].(map[string]interface{}); ok {
				if sn, ok := grpcSettings["serviceName"].(string); ok {
					grpcMap["service_name"] = sn
				}
			}
			ob["transport"] = grpcMap
		case "httpupgrade":
			huMap := map[string]interface{}{"type": "httpupgrade"}
			if huSettings, ok := streamSettings["httpupgradeSettings"].(map[string]interface{}); ok {
				if p, ok := huSettings["path"].(string); ok {
					huMap["path"] = p
				}
				if h, ok := huSettings["host"].(string); ok {
					huMap["host"] = h
				}
			}
			ob["transport"] = huMap
		case "xhttp":
			httpMap := map[string]interface{}{"type": "http"}
			if xhttpSettings, ok := streamSettings["xhttpSettings"].(map[string]interface{}); ok {
				if p, ok := xhttpSettings["path"].(string); ok && p != "" {
					httpMap["path"] = p
				}
				if h, ok := xhttpSettings["host"].(string); ok && h != "" {
					httpMap["host"] = []string{h}
				}
			}
			ob["transport"] = httpMap
		}

		if strings.EqualFold(sec, "tls") {
			tlsMap := map[string]interface{}{"enabled": true}
			if tlsSettings, ok := streamSettings["tlsSettings"].(map[string]interface{}); ok {
				if sn, ok := tlsSettings["serverName"].(string); ok && sn != "" {
					tlsMap["server_name"] = sn
				}
				if fp, ok := tlsSettings["fingerprint"].(string); ok && fp != "" {
					tlsMap["utls"] = map[string]interface{}{
						"enabled":     true,
						"fingerprint": fp,
					}
				}
				if alpn, ok := tlsSettings["alpn"].([]interface{}); ok && len(alpn) > 0 {
					tlsMap["alpn"] = alpn
				}
			}
			ob["tls"] = tlsMap
		} else if strings.EqualFold(sec, "reality") {
			tlsMap := map[string]interface{}{"enabled": true}
			realityMap := map[string]interface{}{"enabled": true}
			fp := "chrome"
			if realitySettings, ok := streamSettings["realitySettings"].(map[string]interface{}); ok {
				if sn, ok := realitySettings["serverName"].(string); ok && sn != "" {
					tlsMap["server_name"] = sn
				}
				if f, ok := realitySettings["fingerprint"].(string); ok && f != "" {
					fp = f
				}
				if pbk, ok := realitySettings["publicKey"].(string); ok {
					realityMap["public_key"] = pbk
				}
				if sid, ok := realitySettings["shortId"].(string); ok {
					realityMap["short_id"] = sid
				}
			}
			tlsMap["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": fp,
			}
			tlsMap["reality"] = realityMap
			ob["tls"] = tlsMap
		}
	}

	return ob, nil
}

func convertClashProxyToSingbox(p map[string]interface{}) (Outbound, error) {
	name, _ := p["name"].(string)
	tp, _ := p["type"].(string)
	server, _ := p["server"].(string)
	port := getInt(p["port"], 443)
	if server == "" || tp == "" {
		return nil, fmt.Errorf("invalid clash proxy")
	}

	ob := Outbound{
		"tag":         name,
		"server":      server,
		"server_port": port,
	}

	switch strings.ToLower(tp) {
	case "ss":
		ob["type"] = "shadowsocks"
		ob["method"], _ = p["cipher"].(string)
		ob["password"], _ = p["password"].(string)
	case "vless":
		ob["type"] = "vless"
		ob["uuid"], _ = p["uuid"].(string)
		if flow, ok := p["flow"].(string); ok && flow != "" {
			ob["flow"] = flow
		}
	case "vmess":
		ob["type"] = "vmess"
		ob["uuid"], _ = p["uuid"].(string)
		ob["alter_id"] = getInt(p["alterId"], 0)
		ob["security"], _ = p["cipher"].(string)
		if ob["security"] == "" {
			ob["security"] = "auto"
		}
	case "trojan":
		ob["type"] = "trojan"
		ob["password"], _ = p["password"].(string)
	case "hysteria2", "hy2":
		ob["type"] = "hysteria2"
		ob["password"], _ = p["password"].(string)
	default:
		ob["type"] = tp
	}

	tlsVal, _ := p["tls"].(bool)
	sni, _ := p["servername"].(string)
	if sni == "" {
		sni, _ = p["sni"].(string)
	}

	if tlsVal || sni != "" || p["reality-opts"] != nil {
		tlsMap := map[string]interface{}{
			"enabled": true,
		}
		if sni != "" {
			tlsMap["server_name"] = sni
		}
		if fp, ok := p["client-fingerprint"].(string); ok && fp != "" {
			tlsMap["utls"] = map[string]interface{}{
				"enabled":     true,
				"fingerprint": fp,
			}
		}
		if ro, ok := p["reality-opts"].(map[string]interface{}); ok {
			realityMap := map[string]interface{}{"enabled": true}
			if pbk, ok := ro["public-key"].(string); ok {
				realityMap["public_key"] = pbk
			}
			if sid, ok := ro["short-id"].(string); ok {
				realityMap["short_id"] = sid
			}
			tlsMap["reality"] = realityMap
		}
		ob["tls"] = tlsMap
	}

	net, _ := p["network"].(string)
	switch strings.ToLower(net) {
	case "ws":
		wsMap := map[string]interface{}{"type": "ws"}
		if wo, ok := p["ws-opts"].(map[string]interface{}); ok {
			if path, ok := wo["path"].(string); ok {
				wsMap["path"] = path
			}
			if headers, ok := wo["headers"].(map[string]interface{}); ok {
				wsMap["headers"] = headers
			}
		}
		ob["transport"] = wsMap
	case "grpc":
		grpcMap := map[string]interface{}{"type": "grpc"}
		if goOpts, ok := p["grpc-opts"].(map[string]interface{}); ok {
			if sn, ok := goOpts["grpc-service-name"].(string); ok {
				grpcMap["service_name"] = sn
			}
		}
		ob["transport"] = grpcMap
	}

	return ob, nil
}

func parseJSONSubscription(content string) ([]Outbound, error) {
	tagCounts := make(map[string]int)
	dedupTag := func(baseTag string) string {
		baseTag = strings.TrimSpace(baseTag)
		if baseTag == "" {
			baseTag = "node"
		}
		tagCounts[baseTag]++
		if tagCounts[baseTag] > 1 {
			return fmt.Sprintf("%s (%d)", baseTag, tagCounts[baseTag])
		}
		return baseTag
	}

	isDummyNode := func(ob Outbound) bool {
		srv, _ := ob["server"].(string)
		srv = strings.TrimSpace(srv)
		if srv == "" || srv == "127.0.0.1" || srv == "0.0.0.0" {
			return true
		}
		tag, _ := ob["tag"].(string)
		tLower := strings.ToLower(tag)
		if strings.Contains(tLower, "трафик") || strings.Contains(tLower, "использовано") || strings.Contains(tLower, "остаток") {
			return true
		}
		return false
	}

	var arr []interface{}
	if err := json.Unmarshal([]byte(content), &arr); err == nil {
		var result []Outbound
		for _, item := range arr {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			remarks, _ := m["remarks"].(string)
			remarks = strings.TrimSpace(remarks)
			if remarks == "" {
				remarks = "node"
			}

			if outboundsRaw, ok := m["outbounds"].([]interface{}); ok {
				var itemProxies []Outbound
				for _, obRaw := range outboundsRaw {
					if obMap, ok := obRaw.(map[string]interface{}); ok {
						if converted, err := convertXrayOutboundToSingbox(obMap, remarks); err == nil {
							if !isDummyNode(converted) {
								itemProxies = append(itemProxies, converted)
							}
						}
					}
				}
				for _, ob := range itemProxies {
					ob["tag"] = dedupTag(remarks)
					result = append(result, ob)
				}
				continue
			}

			if tp, ok := m["type"].(string); ok && tp != "" {
				if tp != "direct" && tp != "block" && tp != "dns" && tp != "selector" && tp != "urltest" {
					ob := Outbound(m)
					if !isDummyNode(ob) {
						ob["tag"] = dedupTag(remarks)
						result = append(result, ob)
					}
					continue
				}
			}

			if _, ok := m["server"].(string); ok {
				if converted, err := convertClashProxyToSingbox(m); err == nil {
					if !isDummyNode(converted) {
						converted["tag"] = dedupTag(remarks)
						result = append(result, converted)
					}
					continue
				}
			}
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(content), &obj); err == nil {
		var result []Outbound
		if outboundsRaw, ok := obj["outbounds"].([]interface{}); ok {
			remarks, _ := obj["remarks"].(string)
			remarks = strings.TrimSpace(remarks)
			if remarks == "" {
				remarks = "node"
			}
			for _, obRaw := range outboundsRaw {
				if obMap, ok := obRaw.(map[string]interface{}); ok {
					if _, hasProto := obMap["protocol"]; hasProto {
						if converted, err := convertXrayOutboundToSingbox(obMap, remarks); err == nil {
							if !isDummyNode(converted) {
								converted["tag"] = dedupTag(remarks)
								result = append(result, converted)
							}
						}
					} else if tp, ok := obMap["type"].(string); ok && tp != "" {
						if tp != "direct" && tp != "block" && tp != "dns" && tp != "selector" && tp != "urltest" {
							ob := Outbound(obMap)
							if !isDummyNode(ob) {
								ob["tag"] = dedupTag(remarks)
								result = append(result, ob)
							}
						}
					}
				}
			}
		}

		if proxiesRaw, ok := obj["proxies"].([]interface{}); ok {
			for _, pRaw := range proxiesRaw {
				if pMap, ok := pRaw.(map[string]interface{}); ok {
					if converted, err := convertClashProxyToSingbox(pMap); err == nil {
						if !isDummyNode(converted) {
							tag, _ := converted["tag"].(string)
							converted["tag"] = dedupTag(tag)
							result = append(result, converted)
						}
					}
				}
			}
		}

		if len(result) > 0 {
			return result, nil
		}
	}

	return nil, fmt.Errorf("no supported proxy structures found in JSON")
}

func parseLink(line string) ([]Outbound, error) {
	line = strings.TrimSpace(line)
	if strings.Contains(line, "\n") || strings.Contains(line, "\r") {
		return parseMultiple(line)
	}

	if strings.HasPrefix(line, "[") || strings.HasPrefix(line, "{") {
		if obs, err := parseJSONSubscription(line); err == nil && len(obs) > 0 {
			return obs, nil
		}
	}

	if strings.HasPrefix(line, "happ://") {
		decrypted, err := decryptHapp(line)
		if err != nil {
			return nil, fmt.Errorf("happ decrypt failed: %w", err)
		}
		// If decrypted output is an HTTP(S) URL, fetch it
		if strings.HasPrefix(decrypted, "http://") || strings.HasPrefix(decrypted, "https://") {
			return fetchSubscription(decrypted)
		}
		// Otherwise it might be lines of links or JSON
		return parseMultiple(decrypted)
	}

	if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
		return fetchSubscription(line)
	}

	if strings.HasPrefix(line, "vless://") {
		ob, err := parseVLESS(line)
		if err != nil {
			return nil, err
		}
		return []Outbound{ob}, nil
	}

	if strings.HasPrefix(line, "hysteria2://") || strings.HasPrefix(line, "hy2://") {
		ob, err := parseHysteria2(line)
		if err != nil {
			return nil, err
		}
		return []Outbound{ob}, nil
	}

	// Try base64 decoded
	if decoded, ok := tryBase64Decode(line); ok {
		return parseMultiple(decoded)
	}

	return nil, fmt.Errorf("unknown link protocol: %s", line)
}

func parseMultiple(content string) ([]Outbound, error) {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		if obs, err := parseJSONSubscription(trimmed); err == nil && len(obs) > 0 {
			return obs, nil
		}
	}

	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")
	var result []Outbound
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		obs, err := parseLink(l)
		if err == nil {
			result = append(result, obs...)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid proxy nodes found in content")
	}
	return result, nil
}

func fetchSubscription(subURL string) ([]Outbound, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", subURL, nil)
	if err != nil {
		return nil, err
	}

	// Apply configured device profile & hardware ID headers
	profile := GetDeviceProfile()
	profile.ApplyToRequest(req)

	resp, err := client.Do(req)
	if err != nil {
		LogEvent("fail", "Ошибка запроса подписки: %v", err)
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	content := string(data)
	if decoded, ok := tryBase64Decode(content); ok {
		content = decoded
	}
	return parseMultiple(content)
}

func ensureConfigTemplate(configPath string) error {
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	baseTemplate := `{
  "log": {
    "level": "info",
    "output": "/opt/var/log/lxbox.log",
    "timestamp": true
  },
  "dns": {
    "servers": [
      {
        "tag": "remote-dns",
        "address": "tls://1.1.1.1",
        "detour": "select"
      },
      {
        "tag": "local-dns",
        "address": "local",
        "detour": "direct"
      }
    ],
    "final": "local-dns",
    "strategy": "prefer_ipv4"
  },
  "inbounds": [
    {
      "type": "mixed",
      "tag": "mixed-in",
      "listen": "0.0.0.0",
      "listen_port": 7890
    },
    {
      "type": "socks",
      "tag": "socks-in",
      "listen": "0.0.0.0",
      "listen_port": 10808
    }
  ],
  "outbounds": [
    {
      "type": "selector",
      "tag": "select",
      "outbounds": [
        "direct"
      ],
      "default": "direct"
    },
    {
      "type": "direct",
      "tag": "direct"
    },
    {
      "type": "block",
      "tag": "block"
    },
    {
      "type": "dns",
      "tag": "dns-out"
    }
  ],
  "route": {
    "auto_detect_interface": true,
    "rules": [
      {
        "protocol": "dns",
        "outbound": "dns-out"
      },
      {
        "ip_is_private": true,
        "outbound": "direct"
      }
    ],
    "final": "select"
  },
  "experimental": {
    "clash_api": {
      "external_controller": "0.0.0.0:9090",
      "external_ui": "/opt/etc/lxbox/ui",
      "secret": "",
      "default_mode": "rule"
    }
  }
}`
	return os.WriteFile(configPath, []byte(baseTemplate), 0644)
}

func updateConfig(configPath string, newOutbounds []Outbound) error {
	if err := ensureConfigTemplate(configPath); err != nil {
		return fmt.Errorf("cannot initialize config %s: %w", configPath, err)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("cannot read config %s: %w", configPath, err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("invalid config json: %w", err)
	}

	outboundsRaw, ok := cfg["outbounds"].([]interface{})
	if !ok {
		outboundsRaw = []interface{}{}
	}

	// Index existing outbound tags
	existing := make(map[string]int)
	for i, o := range outboundsRaw {
		if om, ok := o.(map[string]interface{}); ok {
			if tag, ok := om["tag"].(string); ok {
				existing[tag] = i
			}
		}
	}

	// Update or append
	for _, nob := range newOutbounds {
		tag, _ := nob["tag"].(string)
		if idx, exists := existing[tag]; exists {
			outboundsRaw[idx] = map[string]interface{}(nob)
		} else {
			outboundsRaw = append(outboundsRaw, map[string]interface{}(nob))
		}
	}

	// Update selector if present
	for _, o := range outboundsRaw {
		if om, ok := o.(map[string]interface{}); ok {
			if om["type"] == "selector" {
				var outList []string
				if origList, ok := om["outbounds"].([]interface{}); ok {
					for _, item := range origList {
						if s, ok := item.(string); ok {
							outList = append(outList, s)
						}
					}
				}
				for _, nob := range newOutbounds {
					tag, _ := nob["tag"].(string)
					found := false
					for _, existingTag := range outList {
						if existingTag == tag {
							found = true
							break
						}
					}
					if !found {
						outList = append(outList, tag)
					}
				}
				om["outbounds"] = outList
			}
		}
	}

	cfg["outbounds"] = outboundsRaw

	outBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write
	tmpFile := configPath + ".tmp"
	if err := os.WriteFile(tmpFile, outBytes, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, configPath)
}

func main() {
	importCmd := flag.NewFlagSet("import", flag.ExitOnError)
	cfgPath := importCmd.String("c", "/opt/etc/lxbox/config.json", "Path to sing-box config.json")

	webCmd := flag.NewFlagSet("web", flag.ExitOnError)
	webAddr := webCmd.String("addr", "0.0.0.0:9095", "Listen address for web server")
	webCfgPath := webCmd.String("c", "/opt/etc/lxbox/config.json", "Path to sing-box config.json")

	if len(os.Args) < 2 {
		fmt.Println("LxBox Router CLI Helper")
		fmt.Println("Usage:")
		fmt.Println("  lxbox-helper import '<happ/vless/sub-url>' [-c /path/to/config.json]")
		fmt.Println("  lxbox-helper parse '<happ/vless/sub-url>'")
		fmt.Println("  lxbox-helper web [-addr 0.0.0.0:9095] [-c /path/to/config.json]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "web":
		webCmd.Parse(os.Args[2:])
		if err := runWebServer(*webAddr, *webCfgPath); err != nil {
			fmt.Printf("Web server error: %v\n", err)
			os.Exit(1)
		}

	case "import":
		if len(os.Args) < 3 {
			fmt.Println("Error: link argument is required")
			os.Exit(1)
		}
		link := os.Args[2]
		importCmd.Parse(os.Args[3:])

		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
			subFile := filepath.Join(filepath.Dir(*cfgPath), "subscription.txt")
			_ = os.WriteFile(subFile, []byte(link), 0644)
		}

		obs, err := parseLink(link)
		if err != nil {
			fmt.Printf("Error: failed to parse link: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[+] Parsed %d outbound node(s):\n", len(obs))
		for _, o := range obs {
			fmt.Printf("    - %s (%s -> %s:%v)\n", o["tag"], o["type"], o["server"], o["server_port"])
		}

		if err := updateConfig(*cfgPath, obs); err != nil {
			fmt.Printf("Error updating config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[+] Configuration updated successfully: %s\n", *cfgPath)

		// Test configuration with sing-box check if sing-box binary exists
		if _, err := exec.LookPath("sing-box"); err == nil {
			cmd := exec.Command("sing-box", "check", "-c", *cfgPath)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Printf("[!] Warning: sing-box check failed: %s\n", string(out))
			} else {
				fmt.Println("[+] sing-box configuration check passed!")
			}
		}

	case "parse":
		if len(os.Args) < 3 {
			fmt.Println("Error: link argument is required")
			os.Exit(1)
		}
		link := os.Args[2]
		obs, err := parseLink(link)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		out, _ := json.MarshalIndent(obs, "", "  ")
		fmt.Println(string(out))

	case "hwid":
		if len(os.Args) < 3 || os.Args[2] == "show" {
			p := GetDeviceProfile()
			fmt.Printf("Active X-HWID: %s\n", p.HWID)
			fmt.Printf("Device OS:     %s\n", p.DeviceOS)
			fmt.Printf("Device Model:  %s\n", p.DeviceModel)
			fmt.Printf("User-Agent:    %s\n", p.UserAgent)
			fmt.Println("\nRaw Headers:")
			fmt.Println(p.RawHeaders)
			return
		}
		if os.Args[2] == "preset" && len(os.Args) >= 4 {
			var tpl string
			switch strings.ToLower(os.Args[3]) {
			case "android":
				tpl = PresetLxBoxAndroid
			case "ios":
				tpl = PresetHappIOS
			case "windows":
				tpl = PresetWindows
			default:
				fmt.Printf("Unknown preset: %s (choose: android, ios, windows)\n", os.Args[3])
				os.Exit(1)
			}
			p, err := UpdateDeviceProfile(tpl)
			if err != nil {
				fmt.Printf("Error saving preset: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[+] Applied preset '%s'! Active X-HWID: %s\n", os.Args[3], p.HWID)
			return
		}
		if os.Args[2] == "set" && len(os.Args) >= 4 {
			p, err := UpdateDeviceProfile(os.Args[3])
			if err != nil {
				fmt.Printf("Error saving HWID headers: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[+] HWID headers updated successfully! Active X-HWID: %s\n", p.HWID)
			return
		}
		if os.Args[2] == "random" {
			p := GetDeviceProfile()
			newHWID := GenerateRandomHWID()
			var newRaw string
			if strings.Contains(strings.ToLower(p.RawHeaders), "hwid") {
				lines := strings.Split(p.RawHeaders, "\n")
				for i, l := range lines {
					if strings.Contains(strings.ToLower(l), "hwid") {
						lines[i] = "X-HWID: " + newHWID
					}
				}
				newRaw = strings.Join(lines, "\n")
			} else {
				newRaw = "X-HWID: " + newHWID + "\n" + p.RawHeaders
			}
			p, err := UpdateDeviceProfile(newRaw)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[+] Generated new random X-HWID: %s\n", p.HWID)
			return
		}
		fmt.Println("Usage: lxbox-helper hwid [show | preset <android|ios|windows> | set '<headers>' | random]")

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
