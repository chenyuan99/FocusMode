//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"
	"time"
	"unsafe"
)

const (
	trayCallbackMessage = 0x0400 + 1

	wmCommand   = 0x0111
	wmDestroy   = 0x0002
	wmLButtonUp = 0x0202
	wmRButtonUp = 0x0205

	nimAdd    = 0x00000000
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	mfString    = 0x00000000
	mfSeparator = 0x00000800

	tpmRightButton = 0x00000002
	tpmReturnCmd   = 0x00000100
	tpmNonotify    = 0x00000080

	idiApplication = 32512
	imageIcon      = 1
	lrLoadFromFile = 0x00000010
	lrDefaultSize  = 0x00000040

	firstModeCommand = 1000
	restoreCommand   = 2000
	reportCommand    = 2001
	quitCommand      = 2002
)

type point struct {
	X int32
	Y int32
}

type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

var (
	user32  = syscall.NewLazyDLL("user32.dll")
	shell32 = syscall.NewLazyDLL("shell32.dll")
	kernel  = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procLoadIconW        = user32.NewProc("LoadIconW")
	procLoadImageW       = user32.NewProc("LoadImageW")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procAppendMenuW      = user32.NewProc("AppendMenuW")
	procDestroyMenu      = user32.NewProc("DestroyMenu")
	procSetForegroundWin = user32.NewProc("SetForegroundWindow")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procMessageBoxW      = user32.NewProc("MessageBoxW")

	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandleW = kernel.NewProc("GetModuleHandleW")

	trayHWnd      uintptr
	trayCommands  map[uintptr]func()
	trayModeNames []string
)

func runTray(config *Config, configPath string) error {
	runtime.LockOSThread()

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating executable: %w", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	configArg := configPath
	if configArg != "" {
		if absConfig, err := filepath.Abs(configArg); err == nil {
			configArg = absConfig
		}
	}

	trayCommands = buildTrayCommands(config, exePath, workingDir, configArg)

	instance, _, _ := procGetModuleHandleW.Call(0)
	className, err := syscall.UTF16PtrFromString("FocusModeTrayWindow")
	if err != nil {
		return err
	}

	wndProc := syscall.NewCallback(trayWndProc)
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   wndProc,
		HInstance:     instance,
		LpszClassName: className,
	}

	if atom, _, callErr := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		return fmt.Errorf("registering tray window class: %v", callErr)
	}

	hWnd, _, callErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(className)),
		0,
		0, 0, 0, 0,
		0, 0, instance, 0,
	)
	if hWnd == 0 {
		return fmt.Errorf("creating tray window: %v", callErr)
	}
	trayHWnd = hWnd

	if err := addTrayIcon(hWnd); err != nil {
		procDestroyWindow.Call(hWnd)
		return err
	}
	defer deleteTrayIcon(hWnd)

	var message msg
	for {
		ret, _, callErr := procGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(ret) == -1 {
			return fmt.Errorf("reading tray message: %v", callErr)
		}
		if ret == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}

	return nil
}

func buildTrayCommands(config *Config, exePath string, workingDir string, configPath string) map[uintptr]func() {
	commands := make(map[uintptr]func())

	modes := config.getAvailableModes()
	sort.Strings(modes)
	trayModeNames = modes

	for i, modeName := range modes {
		modeName := modeName
		commandID := uintptr(firstModeCommand + i)
		commands[commandID] = func() {
			startFocusModeCommand(exePath, workingDir, configPath, "-switch", "-mode", modeName)
		}
	}

	commands[restoreCommand] = func() {
		startFocusModeCommand(exePath, workingDir, configPath, "-restore-all")
	}
	commands[reportCommand] = func() {
		showTrayStatsReport(config, getStatsPath(configPath))
	}
	commands[quitCommand] = func() {
		procDestroyWindow.Call(trayHWnd)
	}

	return commands
}

func startFocusModeCommand(exePath string, workingDir string, configPath string, args ...string) {
	commandArgs := []string{}
	if configPath != "" {
		commandArgs = append(commandArgs, "-config", configPath)
	}
	commandArgs = append(commandArgs, args...)

	cmd := exec.Command(exePath, commandArgs...)
	cmd.Dir = workingDir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting FocusMode command: %v\n", err)
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "FocusMode command failed: %v\n", err)
		}
	}()
}

func addTrayIcon(hWnd uintptr) error {
	icon := loadTrayIcon()

	var data notifyIconData
	data.CbSize = uint32(unsafe.Sizeof(data))
	data.HWnd = hWnd
	data.UID = 1
	data.UFlags = nifMessage | nifIcon | nifTip
	data.UCallbackMessage = trayCallbackMessage
	data.HIcon = icon
	copy(data.SzTip[:], syscall.StringToUTF16("FocusMode"))

	if ret, _, callErr := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&data))); ret == 0 {
		return fmt.Errorf("adding tray icon: %v", callErr)
	}

	return nil
}

func loadTrayIcon() uintptr {
	candidates := []string{}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "focusmode.ico"),
			filepath.Join(exeDir, "assets", "focusmode.ico"),
		)
	}
	if workingDir, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(workingDir, "focusmode.ico"),
			filepath.Join(workingDir, "assets", "focusmode.ico"),
		)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		iconPath, err := syscall.UTF16PtrFromString(candidate)
		if err != nil {
			continue
		}
		icon, _, _ := procLoadImageW.Call(
			0,
			uintptr(unsafe.Pointer(iconPath)),
			imageIcon,
			0,
			0,
			lrLoadFromFile|lrDefaultSize,
		)
		if icon != 0 {
			return icon
		}
	}

	icon, _, _ := procLoadIconW.Call(0, idiApplication)
	return icon
}

func deleteTrayIcon(hWnd uintptr) {
	data := notifyIconData{
		CbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:   hWnd,
		UID:    1,
	}
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
}

func trayWndProc(hWnd uintptr, message uint32, wParam uintptr, lParam uintptr) uintptr {
	switch message {
	case trayCallbackMessage:
		if lParam == wmRButtonUp || lParam == wmLButtonUp {
			showTrayMenu(hWnd)
			return 0
		}
	case wmCommand:
		commandID := wParam & 0xffff
		if handleTrayCommand(commandID) {
			return 0
		}
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hWnd, uintptr(message), wParam, lParam)
	return ret
}

func showTrayMenu(hWnd uintptr) {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	modeIDs := make([]int, 0)
	for commandID := range trayCommands {
		if commandID >= firstModeCommand && commandID < restoreCommand {
			modeIDs = append(modeIDs, int(commandID))
		}
	}
	sort.Ints(modeIDs)

	for _, commandID := range modeIDs {
		title := fmt.Sprintf("Switch to %s", trayModeName(commandID-firstModeCommand))
		appendMenuItem(menu, uintptr(commandID), title)
	}

	procAppendMenuW.Call(menu, mfSeparator, 0, 0)
	appendMenuItem(menu, restoreCommand, "Restore all")
	appendMenuItem(menu, reportCommand, "Report hours")
	procAppendMenuW.Call(menu, mfSeparator, 0, 0)
	appendMenuItem(menu, quitCommand, "Quit")

	var cursor point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	procSetForegroundWin.Call(hWnd)

	commandID, _, _ := procTrackPopupMenu.Call(
		menu,
		tpmRightButton|tpmReturnCmd|tpmNonotify,
		uintptr(cursor.X),
		uintptr(cursor.Y),
		0,
		hWnd,
		0,
	)

	if commandID != 0 {
		handleTrayCommand(commandID)
	}
}

func handleTrayCommand(commandID uintptr) bool {
	action, ok := trayCommands[commandID]
	if !ok {
		return false
	}

	if commandID == quitCommand {
		action()
		return true
	}

	go action()
	return true
}

func showTrayStatsReport(config *Config, statsPath string) {
	db, err := openStatsDB(statsPath)
	if err != nil {
		showTrayMessage("FocusMode hours", fmt.Sprintf("Could not open stats database:\n%v", err))
		return
	}
	defer db.Close()

	totals, activeMode, err := readModeStats(db, time.Now())
	if err != nil {
		showTrayMessage("FocusMode hours", fmt.Sprintf("Could not read stats:\n%v", err))
		return
	}

	showTrayMessage("FocusMode hours", buildModeStatsReport(config, totals, activeMode, statsPath))
}

func showTrayMessage(title string, body string) {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	bodyPtr, err := syscall.UTF16PtrFromString(body)
	if err != nil {
		return
	}
	procMessageBoxW.Call(trayHWnd, uintptr(unsafe.Pointer(bodyPtr)), uintptr(unsafe.Pointer(titlePtr)), 0)
}

func appendMenuItem(menu uintptr, commandID uintptr, title string) {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	procAppendMenuW.Call(menu, mfString, commandID, uintptr(unsafe.Pointer(titlePtr)))
}

func trayModeName(index int) string {
	if index < 0 || index >= len(trayModeNames) {
		return fmt.Sprintf("mode %d", index+1)
	}
	return trayModeNames[index]
}
