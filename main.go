package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	ExitOk       = 0
	ExitWarning  = 1
	ExitCritical = 2
	ExitUnknown  = 3
)

func getStatusNamed(code int) string {
	switch code {
	case ExitOk:
		return "OK"
	case ExitWarning:
		return "WARNING"
	case ExitCritical:
		return "CRITICAL"
	case ExitUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN"
	}
}

const GithubAPI = "https://api.github.com/repos/cloudflare/cloudflared/releases/latest"
const Timeout = 10 * time.Second

func exitWith(msg string, code int) {
	newMsg := fmt.Sprintf("%s - %s", getStatusNamed(code), msg)
	fmt.Println(newMsg)
	os.Exit(code)
}

func getInstalledVersion() string {
	cmd := exec.Command("cloudflared", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		exitWith("cloudflared not installed", ExitUnknown)
	}

	lines := strings.Split(out.String(), "\n")
	if len(lines) == 0 {
		exitWith("cloudflared version output empty", ExitUnknown)
	}

	parts := strings.Fields(lines[0])
	if len(parts) < 3 {
		exitWith("cloudflared version output malformed", ExitUnknown)
	}

	return strings.TrimPrefix(parts[2], "v")
}

func getLatestVersion(token string) string {
	client := &http.Client{Timeout: Timeout}
	req, err := http.NewRequest("GET", GithubAPI, nil)
	if err != nil {
		exitWith("Failed to create HTTP request", ExitUnknown)
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		exitWith("Failed to fetch Github API:"+err.Error(), ExitUnknown)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		exitWith(fmt.Sprintf("Failed to fetch Github API: %s", resp.Status), ExitUnknown)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		exitWith("Failed to decode Github API response: "+err.Error(), ExitUnknown)
	}

	tag, ok := data["tag_name"].(string)
	if !ok || tag == "" {
		exitWith("Failed to parse Github API response: tag_name not found", ExitUnknown)
	}

	return strings.TrimPrefix(tag, "v")
}

func main() {
	token := flag.String("token", "", "Github API token")
	flag.Parse()

	installed := getInstalledVersion()
	latest := getLatestVersion(*token)
	if installed != latest {
		exitWith(fmt.Sprintf("Installed version: %s, Latest version: %s", installed, latest), ExitWarning)
	}
	exitWith(fmt.Sprintf("Installed version: %s, Latest version: %s", installed, latest), ExitOk)
}
