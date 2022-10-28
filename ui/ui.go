/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"fmt"
	"ncryptagent/ncrypt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

var noTrayAvailable = false
var startTime = time.Now()

func RunUI() {
	runtime.LockOSThread()
	windows.SetProcessPriorityBoost(windows.CurrentProcess(), false)
	defer func() {
		if err := recover(); err != nil {
			showErrorCustom(nil, "Panic", fmt.Sprint(err, "\n\n", string(debug.Stack())))
			panic(err)
		}
	}()

	var (
		err  error
		mtw  *ManageTunnelsWindow
		tray *Tray
	)

	homeDir, err := os.UserConfigDir()
	if err != nil {
		showErrorCustom(nil, "Unable discover home directory", fmt.Sprintf("%s", err))
		return
	}
	configPath := filepath.Join(homeDir, "nCryptAgent/config.json")

	km, err := ncrypt.NewKeyManager(configPath)
	if err != nil {
		showErrorCustom(nil, "Unable to load KeyManager", fmt.Sprintf("%s", err))
		return
	}

	defer km.Close()

	for mtw == nil {
		mtw, err = NewManageTunnelsWindow(km)
		if err != nil {
			time.Sleep(time.Millisecond * 400)
		}
	}

	for tray == nil {
		tray, err = NewTray(mtw)
		if err != nil {
			time.Sleep(time.Millisecond * 400)
		}
	}

	km.SetHwnd(mtw.Handle())

	if tray == nil {
		win.ShowWindow(mtw.Handle(), win.SW_MINIMIZE)
	}

	mtw.Run()
	if tray != nil {
		tray.Dispose()
	}
	mtw.Dispose()

}

func onQuit() {
	walk.App().Exit(0)
}

func showError(err error, owner walk.Form) bool {
	if err == nil {
		return false
	}

	showErrorCustom(owner, fmt.Sprintf("Error"), err.Error())

	return true
}

func showErrorCustom(owner walk.Form, title, message string) {
	walk.MsgBox(owner, title, message, walk.MsgBoxIconError)
}

func showWarningCustom(owner walk.Form, title, message string) {
	walk.MsgBox(owner, title, message, walk.MsgBoxIconWarning)
}

func raise(hwnd win.HWND) {
	if win.IsIconic(hwnd) {
		win.ShowWindow(hwnd, win.SW_RESTORE)
	}

	win.SetActiveWindow(hwnd)
	win.SetWindowPos(hwnd, win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_SHOWWINDOW)
	win.SetForegroundWindow(hwnd)
	win.SetWindowPos(hwnd, win.HWND_NOTOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_SHOWWINDOW)
}
