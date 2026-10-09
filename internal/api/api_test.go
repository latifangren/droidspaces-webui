package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/latifangren/droidspaces-webui/internal/model"
	"github.com/latifangren/droidspaces-webui/internal/terminal"
)

func setupTestServer(t *testing.T) (*Server, string, string) {
	tmpDir := t.TempDir()

	t.Setenv("DROIDSPACES_CONTAINERS_DIR", filepath.Join(tmpDir, "Containers"))
	t.Setenv("DROIDSPACES_ROOTFS_DIR", filepath.Join(tmpDir, "rootfs"))
	t.Setenv("DROIDSPACES_LOGS_DIR", filepath.Join(tmpDir, "Logs"))
	t.Setenv("DROIDSPACES_AUTH_FILE", filepath.Join(tmpDir, ".webui_auth"))

	_ = os.MkdirAll(filepath.Join(tmpDir, "Containers"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "rootfs"), 0755)
	_ = os.MkdirAll(filepath.Join(tmpDir, "Logs"), 0755)

	s := NewServer(8080)
	token, _ := s.authMgr.CreateSession()
	return s, tmpDir, token
}

func doRequest(handler http.Handler, token, method, path string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(data)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func TestAuthEndpoints(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()

	// 1. Auth status
	w := doRequest(h, "", "GET", "/api/auth/status", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for auth status, got %d", w.Code)
	}

	// 2. Auth login with wrong password
	w = doRequest(h, "", "POST", "/api/auth/login", map[string]string{"password": "wrongpassword"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}

	// 3. Auth login with default password
	w = doRequest(h, "", "POST", "/api/auth/login", map[string]string{"password": "Droidspaces"})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for default password, got %d", w.Code)
	}

	// 4. Change password with valid token (invalidates all old tokens)
	w = doRequest(h, token, "POST", "/api/auth/change-password", map[string]string{
		"current_password": "Droidspaces",
		"new_password":     "NewSecretPassword123",
	})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 changing password, got %d", w.Code)
	}

	// 5. Login with new password
	w = doRequest(h, "", "POST", "/api/auth/login", map[string]string{"password": "NewSecretPassword123"})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for login with new password, got %d", w.Code)
	}

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	newToken := resp.Data.Token

	// 6. Auth logout with fresh token
	w = doRequest(h, newToken, "POST", "/api/auth/logout", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 logging out, got %d", w.Code)
	}
}

func TestSystemAndTelemetryEndpoints(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()

	// 1. Hardware stats
	w := doRequest(h, token, "GET", "/api/hardware", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for hardware, got %d", w.Code)
	}

	// 2. Host interfaces
	w = doRequest(h, token, "GET", "/api/host/interfaces", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for host interfaces, got %d", w.Code)
	}

	// 3. Settings
	w = doRequest(h, token, "GET", "/api/settings", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for settings, got %d", w.Code)
	}

	// 4. Check
	w = doRequest(h, token, "GET", "/api/check", nil)
	_ = w.Code

	// 5. Host exec validation
	w = doRequest(h, token, "POST", "/api/host/exec", map[string]string{"command": ""})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty host exec command, got %d", w.Code)
	}

	// 6. Boot priority GET & POST
	w = doRequest(h, token, "GET", "/api/boot-priority", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for boot priority, got %d", w.Code)
	}

	w = doRequest(h, token, "POST", "/api/boot-priority", model.UpdateBootPriorityRequest{
		Items: []model.BootPriorityItem{
			{Name: "box-test", RunAtBoot: true, RunAtBootPriority: 10},
		},
	})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 setting boot priority, got %d", w.Code)
	}
}

func TestLogsEndpointsAndAutoTruncate(t *testing.T) {
	s, tmpDir, token := setupTestServer(t)
	h := s.Handler()

	logsDir := filepath.Join(tmpDir, "Logs")
	testLogFile := filepath.Join(logsDir, "test-daemon.log")
	_ = os.WriteFile(testLogFile, []byte("line1\nline2\nline3\n"), 0644)

	// 1. List logs
	w := doRequest(h, token, "GET", "/api/logs", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 listing logs, got %d", w.Code)
	}

	// 2. Read specific log
	w = doRequest(h, token, "GET", "/api/logs?file=test-daemon.log", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 reading log file, got %d", w.Code)
	}

	// 3. Download log
	w = doRequest(h, token, "GET", "/api/logs?file=test-daemon.log&download=1", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 downloading log, got %d", w.Code)
	}

	// 4. Clear log file
	w = doRequest(h, token, "POST", "/api/logs?file=test-daemon.log&action=clear", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 clearing log, got %d", w.Code)
	}

	// 5. Test lines parameter and missing file
	_ = doRequest(h, token, "GET", "/api/logs?file=test-daemon.log&lines=2", nil)
	_ = doRequest(h, token, "GET", "/api/logs?file=nonexistent.log", nil)
	_ = doRequest(h, token, "POST", "/api/logs?file=test-daemon.log&action=unknown", nil)

	// 6. Test malicious log path traversal
	w = doRequest(h, token, "GET", "/api/logs?file=../../etc/passwd", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for path traversal, got %d", w.Code)
	}
	bigLog := filepath.Join(logsDir, "big.log")
	var lineBuf bytes.Buffer
	for range 3500 {
		lineBuf.WriteString("2026-10-09 12:00:00 [info] This is a realistic long log line entry repeated to test log truncation mechanism effectively in test suites.\n")
	}
	pad := bytes.Repeat([]byte("A"), 2*1024*1024)
	lineBuf.Write(pad)
	for range 3000 {
		lineBuf.WriteString("2026-10-09 12:01:00 [info] subsequent log line after huge chunk to ensure retained lines exceed limit\n")
	}
	_ = os.WriteFile(bigLog, lineBuf.Bytes(), 0644)
	RotateAndTruncateLogs()

	info, err := os.Stat(bigLog)
	if err != nil || info.Size() > MaxLogSizeBytes {
		t.Errorf("expected log to be truncated under %d bytes, got %v (size: %d)", MaxLogSizeBytes, err, info.Size())
	}
}

func TestTemplatesEndpoints(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()

	// 1. List templates and status
	w := doRequest(h, token, "GET", "/api/templates", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 listing templates, got %d", w.Code)
	}

	// 2. Start download with empty ID
	w = doRequest(h, token, "POST", "/api/templates/download", map[string]string{"id": ""})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty template id, got %d", w.Code)
	}

	// 3. Delete template with empty ID
	w = doRequest(h, token, "POST", "/api/templates/delete", map[string]string{"id": ""})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty delete id, got %d", w.Code)
	}
}

func TestTerminalSessionsEndpoints(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()

	// 1. List sessions
	w := doRequest(h, token, "GET", "/api/terminal/sessions", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 listing terminal sessions, got %d", w.Code)
	}

	// 2. Create session with invalid parameters
	w = doRequest(h, token, "POST", "/api/terminal/sessions", map[string]string{
		"target":    "container",
		"container": "",
	})
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
		t.Errorf("expected error for invalid session creation, got %d", w.Code)
	}

	// 3. Delete nonexistent session
	w = doRequest(h, token, "DELETE", "/api/terminal/sessions/ses_nonexistent", nil)
	_ = w.Code
}

func TestStaticEmbedAndFallback(t *testing.T) {
	s, _, _ := setupTestServer(t)
	h := s.Handler()

	// Root path serves SPA
	w := doRequest(h, "", "GET", "/", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for root, got %d", w.Code)
	}

	// SPA route fallback serves index.html
	w = doRequest(h, "", "GET", "/containers/detail", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for SPA route, got %d", w.Code)
	}
}

func TestContainerConfigAndActionEndpoints(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()

	// 1. Container action dispatch with invalid name
	w := doRequest(h, token, "GET", "/api/containers/invalid!name/status", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid container name, got %d", w.Code)
	}

	// 2. Container action dispatch with unknown action
	w = doRequest(h, token, "POST", "/api/containers/test-box/unknownaction", nil)
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("expected error for unknown container action, got %d", w.Code)
	}
	// 5. Container top-level GET and POST
	_ = doRequest(h, token, "GET", "/api/containers", nil)
	wEmpty := doRequest(h, token, "POST", "/api/containers", model.StartRequest{Name: ""})
	if wEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty start request name, got %d", wEmpty.Code)
	}

	// 6. Container sub-actions: info, start, stop, restart, delete, users, services, exec, clone, backups, backup, processes
	_ = doRequest(h, token, "GET", "/api/containers/test-box", nil)
	_ = doRequest(h, token, "POST", "/api/containers/test-box/start", nil)
	_ = doRequest(h, token, "POST", "/api/containers/test-box/stop", nil)
	_ = doRequest(h, token, "POST", "/api/containers/test-box/restart", nil)
	_ = doRequest(h, token, "POST", "/api/containers/test-box/delete", nil)
	_ = doRequest(h, token, "POST", "/api/containers/test-box/exec", model.ExecRequest{Command: "uptime"})
	_ = doRequest(h, token, "POST", "/api/containers/test-box/clone", map[string]string{"target": "cloned"})
	_ = doRequest(h, token, "POST", "/api/containers/test-box/backup", nil)
	_ = doRequest(h, token, "POST", "/api/containers/test-box/backup/delete", map[string]string{"filename": "test.tar.gz"})
	_ = doRequest(h, token, "GET", "/api/containers/test-box/users", nil)
	_ = doRequest(h, token, "GET", "/api/containers/test-box/services", nil)
	_ = doRequest(h, token, "POST", "/api/containers/test-box/services/manage", map[string]string{"action": "start", "service": "ssh"})
	_ = doRequest(h, token, "GET", "/api/containers/test-box/services/journal?service=ssh", nil)
	_ = doRequest(h, token, "GET", "/api/containers/test-box/processes", nil)
	_ = doRequest(h, token, "POST", "/api/containers/test-box/processes/kill", model.KillProcessRequest{PID: 100})
	_ = doRequest(h, token, "GET", "/api/containers/test-box/backups", nil)

	// 7. Status endpoint
	_ = doRequest(h, token, "GET", "/api/status", nil)

	// 8. Backup download
	wDlEmpty := doRequest(h, token, "GET", "/api/backups/download", nil)
	if wDlEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty backup download file, got %d", wDlEmpty.Code)
	}
	wDlMiss := doRequest(h, token, "GET", "/api/backups/download?file=missing.tar.gz", nil)
	if wDlMiss.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing backup download, got %d", wDlMiss.Code)
	}

	// 4. Restore action validations
	w = doRequest(h, token, "POST", "/api/containers/restore", model.RestoreRequest{TargetName: ""})
	if w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
		t.Errorf("expected error for empty restore target, got %d", w.Code)
	}

	// 5. Global backups list
	w = doRequest(h, token, "GET", "/api/backups", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/backups, got %d", w.Code)
	}
}


func TestEdgeCasesAndMethodValidation(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()

	// 1. Wrong method for auth status
	w := doRequest(h, token, "POST", "/api/auth/status", nil)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST to auth status, got %d", w.Code)
	}

	// 2. Hardware GET
	w = doRequest(h, token, "GET", "/api/hardware", nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for hardware, got %d", w.Code)
	}
	// 3. Invalid JSON for login
	reqBad := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer([]byte("{invalid-json")))
	wBad := httptest.NewRecorder()
	h.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid login JSON, got %d", wBad.Code)
	}

	// 4. WebSocket terminal non-upgrade HTTP request
	wWS := doRequest(h, token, "GET", "/api/ws/terminal", nil)
	if wWS.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-WS handshake to ws/terminal, got %d", wWS.Code)
	}

	// 5. Container start with invalid JSON
	reqStartBad := httptest.NewRequest("POST", "/api/containers", bytes.NewBuffer([]byte("{invalid-json")))
	reqStartBad.Header.Set("Authorization", "Bearer "+token)
	wStartBad := httptest.NewRecorder()
	h.ServeHTTP(wStartBad, reqStartBad)
	if wStartBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad container JSON, got %d", wStartBad.Code)
	}
}
func TestKernelCheckParsingAndPtyClient(t *testing.T) {
	// Test generateClientID
	id1 := generateClientID()
	id2 := generateClientID()
	if !strings.HasPrefix(id1, "cli_") || !strings.HasPrefix(id2, "cli_") || id1 == id2 {
		t.Errorf("unexpected client IDs: %s vs %s", id1, id2)
	}

	// Test getHumanFeatureDesc
	desc := getHumanFeatureDesc("PID namespace")
	if desc == "" {
		t.Errorf("expected non-empty description for PID namespace")
	}
	descUnknown := getHumanFeatureDesc("UNKNOWN_FEATURE")
	if descUnknown != "" {
		t.Errorf("expected empty description for unknown feature")
	}

	sampleCheck := `
[MUST HAVE]
[v] PID namespace: enabled
      Private process isolation
[ ] Cgroups: missing

[RECOMMENDED]
[v] Overlayfs: enabled

[OPTIONAL]
[v] Macvlan: enabled
`
	res := parseKernelCheckOutput(sampleCheck)
	if len(res.Groups) != 3 {
		t.Errorf("expected 3 groups, got %d", len(res.Groups))
	}
	t.Logf("res: %+v", res)
	for _, g := range res.Groups {
		t.Logf("group: %s, required: %v, items: %+v", g.Title, g.Required, g.Items)
	}
	if res.AllRequiredPassed {
		t.Errorf("expected AllRequiredPassed to be false due to missing cgroups")
	}
}

func TestSettingsAndAllHumanFeatures(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()

	// 1. Settings POST valid
	w := doRequest(h, token, "POST", "/api/settings", model.SettingsConfig{Port: 8095})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 updating settings port, got %d", w.Code)
	}

	// 2. Settings POST invalid JSON
	reqBad := httptest.NewRequest("POST", "/api/settings", bytes.NewBuffer([]byte("{invalid-json")))
	reqBad.Header.Set("Authorization", "Bearer "+token)
	wBad := httptest.NewRecorder()
	h.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad settings json, got %d", wBad.Code)
	}

	// 3. Settings wrong method
	wWrong := doRequest(h, token, "DELETE", "/api/settings", nil)
	if wWrong.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for DELETE settings, got %d", wWrong.Code)
	}

	// 4. Test all human feature descriptions
	features := []string{
		"Root privileges",
		"Linux version",
		"PID namespace",
		"Mount namespace",
		"UTS namespace",
		"IPC namespace",
		"pivot_root syscall",
		"/proc filesystem",
		"/sys filesystem",
		"Seccomp support",
		"Cgroup v2 support",
		"ext4 filesystem",
		"OverlayFS support",
		"Network namespace",
		"Bridge device support",
		"Veth pair support",
		"Memory limit support",
		"CPU limit support",
		"Sandboxing (user namespaces)",
		"IPv6 NAT support",
	}
	for _, feat := range features {
		desc := getHumanFeatureDesc(feat)
		if desc == "" {
			t.Errorf("expected non-empty description for feature %q", feat)
		}
	}
}

func TestServicesAndBackupsRoutes(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()

	// 1. Services sub-routes
	// GET /api/containers/test-box/services
	_ = doRequest(h, token, "GET", "/api/containers/test-box/services", nil)
	// POST /api/containers/test-box/services
	_ = doRequest(h, token, "POST", "/api/containers/test-box/services", map[string]string{
		"service": "ssh",
		"action":  "start",
	})
	// GET /api/containers/test-box/services/ssh/logs
	_ = doRequest(h, token, "GET", "/api/containers/test-box/services/ssh/logs", nil)

	// 2. All Backups management
	// GET /api/backups
	wList := doRequest(h, token, "GET", "/api/backups", nil)
	if wList.Code != http.StatusOK {
		t.Errorf("expected 200 for GET /api/backups, got %d", wList.Code)
	}
	// DELETE /api/backups with empty file
	wDelEmpty := doRequest(h, token, "DELETE", "/api/backups", nil)
	if wDelEmpty.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty file param on DELETE /api/backups, got %d", wDelEmpty.Code)
	}
	// DELETE /api/backups with non-existent file
	wDelMiss := doRequest(h, token, "DELETE", "/api/backups?file=nonexistent.tar.gz", nil)
	_ = wDelMiss.Code
	// POST /api/backups method not allowed
	wWrong := doRequest(h, token, "POST", "/api/backups", nil)
	if wWrong.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST /api/backups, got %d", wWrong.Code)
	}

	// 3. Terminal session delete with valid and invalid IDs
	wTermDel := doRequest(h, token, "DELETE", "/api/terminal/sessions/ses_notfound", nil)
	_ = wTermDel.Code
	wTermBad := doRequest(h, token, "DELETE", "/api/terminal/sessions/", nil)
	if wTermBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty session id delete, got %d", wTermBad.Code)
	}
}

func TestWebSocketAndPTYBroadcaster(t *testing.T) {
	// 1. Test wsClientBroadcaster methods
	bc := &wsClientBroadcaster{
		sendChan: make(chan []byte, 10),
		done:     make(chan struct{}),
	}
	_ = bc.WriteMessage(websocket.BinaryMessage, []byte("data"))
	bc.Close()
	_ = bc.WriteMessage(websocket.BinaryMessage, []byte("after-close"))

	// 2. Test handleTerminalWS via httptest.Server
	s, _, token := setupTestServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	u := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/ws/terminal?token=" + token + "&session=ses_nonexistent"
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err == nil {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
	}

	// 3. Test handleTerminalWS with active session and bidirectional I/O
	rPipe, wPipe, _ := os.Pipe()
	defer rPipe.Close()
	defer terminal.MockPTY(
		func(cmd *exec.Cmd) (*os.File, error) { return wPipe, nil },
		func(f *os.File, sz *pty.Winsize) error { return nil },
	)()

	liveSess, err := s.termManager.CreateSession("host", "", "root", "Interactive")
	if err == nil && liveSess != nil {
		uLive := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/ws/terminal?token=" + token + "&session=" + liveSess.ID
		connLive, _, errDial := websocket.DefaultDialer.Dial(uLive, nil)
		if errDial == nil {
			_ = connLive.WriteMessage(websocket.BinaryMessage, []byte("echo live\n"))
			_ = connLive.WriteMessage(websocket.TextMessage, []byte(`{"type":"resize","cols":100,"rows":30}`))
			_ = connLive.WriteMessage(websocket.TextMessage, []byte(`{"type":"input","data":"ls -la\n"}`))
			_ = connLive.WriteMessage(websocket.TextMessage, []byte("exit\n"))
			_ = connLive.Close()
		}
	}
}

func TestLifecycleRoutesDeep(t *testing.T) {
	s, tmpDir, token := setupTestServer(t)
	h := s.Handler()
	// 1. DELETE /api/containers/{name}
	_ = doRequest(h, token, "DELETE", "/api/containers/test-del-box", nil)

	// 2. POST /api/containers/{name}/backup
	_ = doRequest(h, token, "POST", "/api/containers/test-del-box/backup", nil)

	// 3. POST /api/containers/{name}/backup/delete
	_ = doRequest(h, token, "POST", "/api/containers/test-del-box/backup/delete", map[string]string{
		"filename": "fake_backup.tar.gz",
	})

	// 4. POST /api/containers/{name}/clone
	_ = doRequest(h, token, "POST", "/api/containers/test-del-box/clone", model.CloneRequest{
		TargetName: "target-clone",
		AutoStart:  false,
	})

	// 5. POST /api/containers/{name}/exec
	_ = doRequest(h, token, "POST", "/api/containers/test-del-box/exec", model.ExecRequest{
		Command: "whoami",
		User:    "root",
	})

	// 6. POST /api/containers/{name}/export
	_ = doRequest(h, token, "POST", "/api/containers/test-del-box/export", map[string]string{
		"output": filepath.Join(tmpDir, "export.tar.gz"),
	})

	// 7. Processes kill valid & invalid
	_ = doRequest(h, token, "POST", "/api/containers/test-del-box/processes/kill", model.KillProcessRequest{
		PID:    100,
		Signal: 9,
	})
	_ = doRequest(h, token, "POST", "/api/containers/test-del-box/processes", nil)

	// 8. Services manage and logs
	_ = doRequest(h, token, "POST", "/api/containers/test-del-box/services", map[string]string{
		"service": "ssh",
		"action":  "start",
	})
	_ = doRequest(h, token, "GET", "/api/containers/test-del-box/services/ssh/logs", nil)

	// 9. Backup download real file
	backupDir := "/data/local/Droidspaces/Backups"
	_ = os.MkdirAll(backupDir, 0755)
	_ = os.WriteFile(filepath.Join(backupDir, "dl_test.tar.gz"), []byte("mock-data"), 0644)
	wDl := doRequest(h, token, "GET", "/api/backups/download?file=dl_test.tar.gz", nil)
	if wDl.Code != http.StatusOK {
		t.Errorf("expected 200 downloading backup, got %d", wDl.Code)
	}

	// 10. POST /api/containers/restore
	_ = doRequest(h, token, "POST", "/api/containers/restore", model.RestoreRequest{
		TargetName: "restored-target",
		Filename:   "dl_test.tar.gz",
	})

	// 11. Auth input validation branches
	_ = doRequest(h, token, "POST", "/api/auth/login", map[string]string{"password": ""})
	_ = doRequest(h, token, "POST", "/api/auth/change-password", map[string]string{"current_password": ""})
	_ = doRequest(h, token, "POST", "/api/auth/change-password", map[string]string{
		"current_password": "Droidspaces",
		"new_password":     "",
	})

	// 12. Template download invalid template
	_ = doRequest(h, token, "POST", "/api/templates/download", map[string]string{"id": "unknown-tpl"})
	_ = doRequest(h, token, "POST", "/api/templates/delete", map[string]string{"id": "unknown-tpl"})

	// 13. Terminal sessions create
	_ = doRequest(h, token, "POST", "/api/terminal/sessions", model.CreateSessionRequest{
		Target: "host",
		Title:  "Shell",
	})
}

func TestSystemAndContainerMethodValidations(t *testing.T) {
	s, tmpDir, token := setupTestServer(t)
	h := s.Handler()

	// 1. Method Not Allowed validations
	w3 := doRequest(h, token, "POST", "/api/host/interfaces", nil)
	if w3.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for POST /api/host/interfaces, got %d", w3.Code)
	}

	w4 := doRequest(h, token, "DELETE", "/api/boot-priority", nil)
	if w4.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for DELETE /api/boot-priority, got %d", w4.Code)
	}

	w6 := doRequest(h, token, "GET", "/api/templates/delete", nil)
	if w6.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET /api/templates/delete, got %d", w6.Code)
	}

	w7 := doRequest(h, token, "PUT", "/api/terminal/sessions", nil)
	if w7.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for PUT /api/terminal/sessions, got %d", w7.Code)
	}

	w8 := doRequest(h, token, "GET", "/api/terminal/sessions/ses_abc", nil)
	if w8.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET /api/terminal/sessions/id, got %d", w8.Code)
	}

	w9 := doRequest(h, token, "PUT", "/api/containers", nil)
	if w9.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for PUT /api/containers, got %d", w9.Code)
	}

	// 2. Host exec valid
	_ = doRequest(h, token, "POST", "/api/host/exec", map[string]string{"command": "echo host-test"})

	// 3. DELETE /api/containers/del-box (length 3, MethodDelete)
	boxDir := filepath.Join(tmpDir, "Containers", "del-box")
	_ = os.MkdirAll(boxDir, 0755)
	wDel := doRequest(h, token, "DELETE", "/api/containers/del-box", nil)
	if wDel.Code != http.StatusOK {
		t.Errorf("expected 200 deleting container, got %d", wDel.Code)
	}
}

func TestContainerActionErrors(t *testing.T) {
	s, _, token := setupTestServer(t)
	h := s.Handler()

	// Start failure
	_ = doRequest(h, token, "POST", "/api/containers", model.StartRequest{Name: "fail-box"})
	_ = doRequest(h, token, "POST", "/api/containers/fail-box/start", nil)

	// Stop failure
	_ = doRequest(h, token, "POST", "/api/containers/fail-box/stop", nil)

	// Restart failure
	_ = doRequest(h, token, "POST", "/api/containers/fail-box/restart", nil)

	// Clone failure
	_ = doRequest(h, token, "POST", "/api/containers/fail-box/clone", model.CloneRequest{TargetName: "dst"})

	// Exec failure (returns 200 with ExitCode 1)
	wExec := doRequest(h, token, "POST", "/api/containers/fail-box/exec", model.ExecRequest{Command: "fail"})
	if wExec.Code != http.StatusOK {
		t.Errorf("expected 200 for failed exec with exit code, got %d", wExec.Code)
	}

	// Services manage failure
	_ = doRequest(h, token, "POST", "/api/containers/fail-box/services", map[string]string{
		"service": "ssh",
		"action":  "start",
	})

	// Processes kill failure
	_ = doRequest(h, token, "POST", "/api/containers/fail-box/processes/kill", model.KillProcessRequest{PID: 10, Signal: 9})

	// Restore failure
	// Restore failure and validation branches
	_ = doRequest(h, token, "POST", "/api/containers/restore", model.RestoreRequest{
		TargetName: "fail-box",
		Filename:   "dl_test.tar.gz",
	})
	wResMethod := doRequest(h, token, "GET", "/api/containers/restore", nil)
	if wResMethod.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET /api/containers/restore, got %d", wResMethod.Code)
	}
	wResNoFile := doRequest(h, token, "POST", "/api/containers/restore", model.RestoreRequest{
		TargetName: "box",
		Filename:   "",
	})
	if wResNoFile.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for empty restore filename, got %d", wResNoFile.Code)
	}
	wResNoTarget := doRequest(h, token, "POST", "/api/containers/restore", model.RestoreRequest{
		TargetName: "",
		Filename:   "b.tar.gz",
	})
	if wResNoTarget.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for empty restore target, got %d", wResNoTarget.Code)
	}

	// Invalid container name on Start
	_ = doRequest(h, token, "POST", "/api/containers", model.StartRequest{Name: "../bad"})

	// Wrong password on ChangePassword
	_ = doRequest(h, token, "POST", "/api/auth/change-password", map[string]string{
		"current_password": "WrongPassword",
		"new_password":     "ValidNewSecret123",
	})

	// Malformed JSON bodies on various handlers
	for _, route := range []string{
		"/api/containers/test-box/exec",
		"/api/containers/test-box/clone",
		"/api/containers/test-box/processes/kill",
		"/api/containers/restore",
	} {
		reqBad := httptest.NewRequest("POST", route, bytes.NewBuffer([]byte("{bad-json")))
		reqBad.Header.Set("Authorization", "Bearer "+token)
		wBad := httptest.NewRecorder()
		h.ServeHTTP(wBad, reqBad)
	}
}