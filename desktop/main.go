// Command desktop is the New API desktop app. It runs the new-api server
// shipped among its resources and shows the web console in a window, with a
// tray icon that keeps the server running while the window is closed.
package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/egoist/mygo"
)

const (
	serverPort = 3000
	// devFrontendURL is the Rsbuild dev server of web/, which development
	// builds show instead of starting the server.
	devFrontendURL = "http://127.0.0.1:5173"
	// titleBarHeight is the height of the console's header (--app-header-height
	// in web/), which takes the place of the window's title bar.
	titleBarHeight = 48
	// startupTimeout bounds the wait for the server's first answer; the
	// first launch migrates a new database.
	startupTimeout = 60 * time.Second
	// stopTimeout is how long a quitting app waits for the server to shut
	// down before killing it.
	stopTimeout = 5 * time.Second
	// outputTailSize is how much of the server's latest output the error
	// dialogs can show.
	outputTailSize = 16 << 10
)

var (
	//go:embed tray/tray-icon.png
	trayIcon []byte
	//go:embed tray/tray-icon-template.png
	trayIconTemplate []byte
)

const startingPage = `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<style>
  :root { color-scheme: light dark; font-family: system-ui, sans-serif; }
  body { margin: 0; height: 100vh; display: grid; place-items: center; color: #888; --app-region: drag; }
</style>
</head>
<body>正在启动 New API…</body>
</html>`

var (
	mainWindow *mygo.Window
	server     *serverProcess
	// hasTray tells that closing the window can hide it to the tray, which
	// shows it again. Without a tray, such as on Linux without
	// AppIndicator, closing the window quits.
	hasTray  bool
	quitting atomic.Bool
)

// Desktop is what the console's pages call, through window.newApiDesktop.
type Desktop struct{}

// SetTheme draws the window, the controls over the console's header
// included, in the console's theme: "dark", "light" or "system".
func (Desktop) SetTheme(theme string) {
	switch theme {
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	mygo.Bind(Desktop{})
	mygo.App.OnSecondInstance(func([]string, string) { showMainWindow() })
	mygo.App.OnActivate(func(bool) { showMainWindow() })
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting.Store(true) })
	mygo.App.WhenReady(launch)

	err := mygo.App.Run()
	if server != nil {
		server.stop()
	}
	if err != nil {
		log.Fatal(err)
	}
}

// launch creates the tray and the window, starts the server, and shows the
// console once it answers.
func launch() {
	createTray()

	if mygo.IsDev() {
		// Run `go run main.go` and `make dev-web` from the repository root.
		openMainWindow(desktopAPIScript(""))
		go func() {
			if err := waitUntilServing(devFrontendURL, nil); err != nil {
				reportFailure("无法连接前端开发服务器", "请先在仓库根目录运行 go run main.go 和 make dev-web。", err.Error(), "")
				return
			}
			_ = mainWindow.Page().LoadURL(devFrontendURL)
		}()
		return
	}

	dataDir, err := serverDataDir()
	if err != nil {
		openMainWindow("")
		go reportFailure("启动失败", "无法创建数据目录", err.Error(), "")
		return
	}
	openMainWindow(desktopAPIScript(dataDir))
	server, err = startServer(dataDir)
	if err != nil {
		go reportFailure("启动失败", "无法启动服务器", err.Error(), "")
		return
	}
	go superviseServer(server)
}

// desktopAPIScript defines window.newApiDesktop (web/src/lib/desktop.ts) in
// the console's pages. The setup wizard shows dataDir, where the SQLite
// database is kept.
func desktopAPIScript(dataDir string) string {
	dirJSON, _ := json.Marshal(dataDir)
	return fmt.Sprintf(`window.newApiDesktop = Object.freeze({
  dataDir: %s || undefined,
  setTheme: (theme) => window.mygo.call("Desktop.SetTheme", theme).catch(() => {}),
});`, dirJSON)
}

// openMainWindow shows the window with a starting page until the console
// loads. The console's header is the title bar, under the window controls.
// Closing the window hides it to the tray.
func openMainWindow(preload string) {
	mainWindow = mygo.NewWindow(mygo.WindowOptions{
		Title:          "New API",
		Width:          1080,
		Height:         720,
		MinWidth:       640,
		MinHeight:      480,
		StateKey:       "main",
		TitleBarStyle:  mygo.TitleBarHidden,
		TitleBarHeight: titleBarHeight,
		Page: mygo.PageOptions{
			PreloadScript: preload,
			// The console calls Desktop. Development builds trust their
			// dev server on localhost already.
			TrustedOrigins: []string{"http://127.0.0.1:" + strconv.Itoa(serverPort)},
		},
	})
	mainWindow.Page().LoadHTML(startingPage, "")
	mainWindow.OnClose(func(e *mygo.CloseEvent) {
		if !hasTray || quitting.Load() {
			return
		}
		e.PreventDefault()
		mainWindow.Hide()
		if runtime.GOOS == "darwin" {
			mygo.App.Dock.Hide()
		}
	})
}

// superviseServer shows the console once the server answers, and reports
// the server ending before the app quits.
func superviseServer(s *serverProcess) {
	url := "http://127.0.0.1:" + strconv.Itoa(serverPort)
	if err := waitUntilServing(url, s.exited); err != nil {
		if errors.Is(err, errServerExited) {
			reportServerExit(s)
			return
		}
		reportFailure("启动失败", "无法启动服务器", err.Error()+"\n\n最近的输出:\n"+s.output.lastLines(20), s.output.path)
		return
	}
	_ = mainWindow.Page().LoadURL(url)
	<-s.exited
	if !quitting.Load() {
		reportServerExit(s)
	}
}

func reportServerExit(s *serverProcess) {
	if s.exitCode == 0 {
		mygo.App.Quit()
		return
	}
	output := s.output.lastLines(20)
	detail := fmt.Sprintf("退出代码: %d\n\n最近的输出:\n%s", s.exitCode, output)
	if known := diagnose(output); known != nil {
		reportFailure(known.title, known.message, known.solution+"\n\n━━━━━━━━━━━━━━━━━━━━━━\n\n"+detail, s.output.path)
		return
	}
	reportFailure("服务器崩溃", "服务器进程异常退出", detail, s.output.path)
}

// reportFailure shows an error the app cannot recover from, offers the
// server's log when there is one, and quits.
func reportFailure(title, message, detail, logPath string) {
	buttons := []string{"退出应用"}
	if logPath != "" {
		buttons = append(buttons, "查看完整日志")
	}
	res, _ := mygo.Dialog.Message(mygo.MessageOptions{
		Type:    mygo.MessageError,
		Title:   title,
		Message: message,
		Detail:  detail,
		Buttons: buttons,
	})
	if res.Button == 1 {
		if err := mygo.Shell.OpenPath(logPath); err != nil {
			mygo.Shell.ShowItemInFolder(logPath)
		}
	}
	mygo.App.Quit()
}

type knownError struct {
	title, message, solution string
}

// diagnose recognizes the common reasons the server fails to start.
func diagnose(output string) *knownError {
	text := strings.ToLower(output)
	switch {
	case strings.Contains(text, "address already in use"),
		strings.Contains(text, "only one usage of each socket address"):
		return &knownError{
			title:   fmt.Sprintf("端口 %d 被占用", serverPort),
			message: "无法启动服务器，端口已被其他程序占用",
			solution: fmt.Sprintf("可能的解决方案：\n\n1. 关闭占用端口 %d 的其他程序\n2. 检查是否已经运行了另一个 New API 实例\n3. 查找占用端口的进程：\n   Mac/Linux: lsof -i :%d\n   Windows: netstat -ano | findstr :%d",
				serverPort, serverPort, serverPort),
		}
	case strings.Contains(text, "database is locked"), strings.Contains(text, "unable to open database"):
		return &knownError{
			title:    "无法访问数据文件",
			message:  "应用的数据文件正被其他程序占用",
			solution: "可能的解决方案：\n\n1. 退出其他正在运行的 New API\n2. 如果刚刚关闭过应用，请等待 10 秒后再试\n3. 重启电脑以释放被占用的文件",
		}
	case strings.Contains(text, "permission denied"), strings.Contains(text, "access is denied"):
		return &knownError{
			title:    "权限不足",
			message:  "程序没有足够的权限执行操作",
			solution: "可能的解决方案：\n\n1. 检查数据目录的读写权限\n2. 检查安全软件是否拦截了 New API",
		}
	}
	return nil
}

func createTray() {
	icon, template := trayIcon, false
	if runtime.GOOS == "darwin" {
		icon, template = trayIconTemplate, true
	}
	_, err := mygo.NewTray(mygo.TrayOptions{
		Icon:           icon,
		IconIsTemplate: template,
		ToolTip:        "New API",
		Menu: mygo.NewMenu([]*mygo.MenuItem{
			{Label: "Show New API", Click: func(*mygo.MenuItem, *mygo.Window) { showMainWindow() }},
			mygo.Separator(),
			{Label: "Quit", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
		}),
	})
	if err != nil {
		log.Println("tray:", err)
		return
	}
	hasTray = true
}

func showMainWindow() {
	if mainWindow == nil {
		return
	}
	if runtime.GOOS == "darwin" {
		mygo.App.Dock.Show()
	}
	mainWindow.Show()
	if mainWindow.IsMinimized() {
		mainWindow.Restore()
	}
	mainWindow.Focus()
}

// serverDataDir returns the directory of the server's SQLite database.
// Installs of the former Electron app kept it in "new-api-electron", its
// package name, under the user's configuration directory, which stays in
// use while it holds a database and the app's own directory does not.
func serverDataDir() (string, error) {
	userData, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(userData, "data")
	if _, err := os.Stat(filepath.Join(dir, "new-api.db")); errors.Is(err, os.ErrNotExist) {
		if appData, err := mygo.App.Path(mygo.PathAppData); err == nil {
			legacy := filepath.Join(appData, "new-api-electron", "data")
			if _, err := os.Stat(filepath.Join(legacy, "new-api.db")); err == nil {
				return legacy, nil
			}
		}
	}
	return dir, os.MkdirAll(dir, 0o755)
}

type serverProcess struct {
	cmd    *exec.Cmd
	output *outputLog
	// exited is closed once the process has ended and exitCode is set.
	exited   chan struct{}
	exitCode int
}

func startServer(dataDir string) (*serverProcess, error) {
	resources, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return nil, err
	}
	logDir, err := mygo.App.Path(mygo.PathLogs)
	if err != nil {
		return nil, err
	}
	output, err := newOutputLog(filepath.Join(logDir, "server.log"))
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(filepath.Join(resources, "bin", "new-api"), "--log-dir", logDir)
	// The working directory holds the database, and an optional .env.
	cmd.Dir = dataDir
	cmd.Env = append(os.Environ(),
		"PORT="+strconv.Itoa(serverPort),
		"SQLITE_PATH="+filepath.Join(dataDir, "new-api.db"),
	)
	cmd.Stdout, cmd.Stderr = output, output
	configureServerCommand(cmd)
	log.Printf("starting %s, data in %s", cmd.Path, dataDir)
	if err := cmd.Start(); err != nil {
		output.Close()
		return nil, err
	}
	if err := tieServerToApp(cmd.Process); err != nil {
		log.Println("server: tie to app:", err)
	}

	s := &serverProcess{cmd: cmd, output: output, exited: make(chan struct{})}
	go func() {
		err := cmd.Wait()
		s.exitCode = cmd.ProcessState.ExitCode()
		if err != nil {
			log.Println("server exited:", err)
		}
		output.Close()
		close(s.exited)
	}()
	return s, nil
}

// stop asks the server to shut down, and kills it when it has not within
// stopTimeout. Windows has no termination signal, so there it is killed.
func (s *serverProcess) stop() {
	select {
	case <-s.exited:
		return
	default:
	}
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = s.cmd.Process.Kill()
	}
	select {
	case <-s.exited:
	case <-time.After(stopTimeout):
		_ = s.cmd.Process.Kill()
		<-s.exited
	}
}

var errServerExited = errors.New("the server exited")

// waitUntilServing polls url until it answers, the server exits (exited
// closes) or startupTimeout passes.
func waitUntilServing(url string, exited <-chan struct{}) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(startupTimeout)
	for {
		res, err := client.Get(url)
		if err == nil {
			res.Body.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s did not answer within %s: %w", url, startupTimeout, err)
		}
		select {
		case <-exited:
			return errServerExited
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// outputLog writes what the server prints to a log file, replaced at each
// launch, and keeps its end for the error dialogs.
type outputLog struct {
	path string
	mu   sync.Mutex
	file *os.File
	tail []byte
}

func newOutputLog(path string) (*outputLog, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &outputLog{path: path, file: f}, nil
}

func (l *outputLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tail = append(l.tail, p...)
	if over := len(l.tail) - outputTailSize; over > 0 {
		l.tail = append(l.tail[:0:0], l.tail[over:]...)
	}
	return l.file.Write(p)
}

func (l *outputLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

func (l *outputLog) lastLines(n int) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	lines := strings.Split(strings.TrimRight(string(l.tail), "\r\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	if len(lines) == 1 && lines[0] == "" {
		return "没有捕获到输出"
	}
	return strings.Join(lines, "\n")
}
