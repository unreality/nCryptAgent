/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"fmt"
	"log"
	"ncryptagent/deviceevents"
	"ncryptagent/keyman"
	"sync"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

type ManageKeysWindow struct {
	walk.FormBase

	tabs     *walk.TabWidget
	keysPage *KeysPage
	confPage *ConfPage

	keyManager *keyman.KeyManager
}

const (
	manageWindowWindowClass = "nCryptAgent UI"
	raiseMsg                = win.WM_USER + 0x3510
	aboutNCryptAgentCmd     = 0x37
)

//var taskbarButtonCreatedMsg uint32

var initedManageTunnels sync.Once

func NewManageKeysWindow(keyManager *keyman.KeyManager) (*ManageKeysWindow, error) {
	initedManageTunnels.Do(func() {
		walk.AppendToWalkInit(func() {
			walk.MustRegisterWindowClass(manageWindowWindowClass)
			//taskbarButtonCreatedMsg = win.RegisterWindowMessage(windows.StringToUTF16Ptr("TaskbarButtonCreated"))
		})
	})

	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	font, err := walk.NewFont("Segoe UI", 9, 0)
	if err != nil {
		return nil, err
	}

	mkw := new(ManageKeysWindow)

	mkw.keyManager = keyManager
	mkw.SetName("nCryptAgent")

	err = walk.InitWindow(mkw, nil, manageWindowWindowClass, win.WS_OVERLAPPEDWINDOW, win.WS_EX_CONTROLPARENT)
	if err != nil {
		return nil, err
	}
	disposables.Add(mkw)
	win.ChangeWindowMessageFilterEx(mkw.Handle(), raiseMsg, win.MSGFLT_ALLOW, nil)
	mkw.SetPersistent(true)

	if icon, err := loadLogoIcon(32); err == nil {
		mkw.SetIcon(icon)
	}
	mkw.SetTitle("nCryptAgent")
	mkw.SetFont(font)
	mkw.SetSize(walk.Size{675, 525})
	mkw.SetMinMaxSize(walk.Size{500, 400}, walk.Size{0, 0})
	vlayout := walk.NewVBoxLayout()
	vlayout.SetMargins(walk.Margins{5, 5, 5, 5})
	mkw.SetLayout(vlayout)
	mkw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		// "Close to tray" instead of exiting application
		*canceled = true
		if !noTrayAvailable {
			mkw.Hide()
		} else {
			win.ShowWindow(mkw.Handle(), win.SW_MINIMIZE)
		}
	})
	mkw.VisibleChanged().Attach(func() {
		if mkw.Visible() {
			win.SetForegroundWindow(mkw.Handle())
			win.BringWindowToTop(mkw.Handle())
		}
	})

	if mkw.tabs, err = walk.NewTabWidget(mkw); err != nil {
		return nil, err
	}

	if mkw.keysPage, err = NewKeysPage(keyManager); err != nil {
		return nil, err
	}
	mkw.tabs.Pages().Add(mkw.keysPage.TabPage)
	mkw.keysPage.CreateToolbar()

	if mkw.confPage, err = NewConfPage(keyManager); err != nil {
		return nil, err
	}
	mkw.tabs.Pages().Add(mkw.confPage.TabPage)

	systemMenu := win.GetSystemMenu(mkw.Handle(), false)
	if systemMenu != 0 {
		win.InsertMenuItem(systemMenu, 0, true, &win.MENUITEMINFO{
			CbSize:     uint32(unsafe.Sizeof(win.MENUITEMINFO{})),
			FMask:      win.MIIM_ID | win.MIIM_STRING | win.MIIM_FTYPE,
			FType:      win.MIIM_STRING,
			DwTypeData: windows.StringToUTF16Ptr(fmt.Sprintf("&About nCryptAgent…")),
			WID:        uint32(aboutNCryptAgentCmd),
		})
		win.InsertMenuItem(systemMenu, 1, true, &win.MENUITEMINFO{
			CbSize: uint32(unsafe.Sizeof(win.MENUITEMINFO{})),
			FMask:  win.MIIM_TYPE,
			FType:  win.MFT_SEPARATOR,
		})
	}

	disposables.Spare()

	return mkw, nil
}

func (mkw *ManageKeysWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_QUERYENDSESSION:
		if lParam == win.ENDSESSION_CLOSEAPP {
			return win.TRUE
		}
	case win.WM_ENDSESSION:
		if lParam == win.ENDSESSION_CLOSEAPP && wParam == 1 {
			walk.App().Exit(198)
		}
	case win.WM_SYSCOMMAND:
		if wParam == aboutNCryptAgentCmd {
			onAbout(mkw)
			return 0
		}
	case win.WM_DEVICECHANGE:

		if wParam == deviceevents.DBT_DEVICEARRIVAL || wParam == deviceevents.DBT_DEVICEREMOVECOMPLETE {
			deviceType, deviceGUID, deviceName, err := deviceevents.ReadDeviceInfo(lParam)
			if err != nil {
				log.Printf("Error decoding device information: %s", err)
				return 0
			}

			if deviceGUID == deviceevents.SMARTCARD_DEVICE_CLASS {
				log.Printf("Smartcard insert/remove detected: Type: %d, GUID: %v, name: %s\n", deviceType, deviceGUID, deviceName)
				mkw.keyManager.RescanNCryptKeys()
				mkw.ReloadKeys()
			}
		}
	}

	return mkw.FormBase.WndProc(hwnd, msg, wParam, lParam)
}

func (mkw *ManageKeysWindow) ReloadKeys() {
	mkw.keysPage.listView.Load(false)
}
