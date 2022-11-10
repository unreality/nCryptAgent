/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"fmt"
	"log"
	"ncryptagent/keyman"
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
		mtw  *ManageKeysWindow
		tray *Tray
	)

	homeDir, err := os.UserConfigDir()
	if err != nil {
		showErrorCustom(nil, "Unable discover home directory", fmt.Sprintf("%s", err))
		return
	}
	configPath := filepath.Join(homeDir, "nCryptAgent/config.json")
	logPath := filepath.Join(homeDir, "nCryptAgent/nCryptAgent.log")

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err == nil {
		log.Default().SetOutput(f)
	} else {
		log.Printf("Could not open log file, logging to STDOUT")
	}

	km, err := keyman.NewKeyManager(configPath)
	if err != nil {
		showErrorCustom(nil, "Unable to load KeyManager", fmt.Sprintf("%s", err))
		return
	}

	defer km.Close()

	for mtw == nil {
		mtw, err = NewManageKeysWindow(km)
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

	err = km.Start()
	if err != nil {
		showErrorCustom(nil, "Unable to start KeyManager", fmt.Sprintf("%s", err))
		return
	}

	mtw.ReloadKeys()
	km.SetHwnd(mtw.Handle())

	if tray == nil {
		win.ShowWindow(mtw.Handle(), win.SW_MINIMIZE)
	}

	// Setup a chan to receive notification messages
	notifyChan := make(chan keyman.NotifyMsg)
	quitChan := make(chan int)

	go func() {
		for {
			select {
			case v := <-notifyChan:
				icon, _ := loadSystemIcon(v.Icon.DLL, v.Icon.Index, v.Icon.Size)
				tray.ShowCustom(v.Title, v.Message, icon)
			case <-quitChan:
				return
			}
		}
	}()

	km.SetNotifyChan(notifyChan)

	mtw.Run()
	if tray != nil {
		tray.Dispose()
	}
	mtw.Dispose()
	quitChan <- 0
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
