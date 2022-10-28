/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"fmt"
	"ncryptagent/ncrypt"
	"sync"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

type ManageTunnelsWindow struct {
	walk.FormBase

	tabs     *walk.TabWidget
	keysPage *KeysPage
}

const (
	manageWindowWindowClass = "nCryptAgent UI"
	raiseMsg                = win.WM_USER + 0x3510
	aboutWireGuardCmd       = 0x37
)

var taskbarButtonCreatedMsg uint32

var initedManageTunnels sync.Once

func NewManageTunnelsWindow(keyManager *ncrypt.KeyManager) (*ManageTunnelsWindow, error) {
	initedManageTunnels.Do(func() {
		walk.AppendToWalkInit(func() {
			walk.MustRegisterWindowClass(manageWindowWindowClass)
			taskbarButtonCreatedMsg = win.RegisterWindowMessage(windows.StringToUTF16Ptr("TaskbarButtonCreated"))
		})
	})

	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	font, err := walk.NewFont("Segoe UI", 9, 0)
	if err != nil {
		return nil, err
	}

	mtw := new(ManageTunnelsWindow)
	mtw.SetName("nCryptAgent")

	err = walk.InitWindow(mtw, nil, manageWindowWindowClass, win.WS_OVERLAPPEDWINDOW, win.WS_EX_CONTROLPARENT)
	if err != nil {
		return nil, err
	}
	disposables.Add(mtw)
	win.ChangeWindowMessageFilterEx(mtw.Handle(), raiseMsg, win.MSGFLT_ALLOW, nil)
	mtw.SetPersistent(true)

	if icon, err := loadLogoIcon(32); err == nil {
		mtw.SetIcon(icon)
	}
	mtw.SetTitle("nCryptAgent")
	mtw.SetFont(font)
	mtw.SetSize(walk.Size{675, 525})
	mtw.SetMinMaxSize(walk.Size{500, 400}, walk.Size{0, 0})
	vlayout := walk.NewVBoxLayout()
	vlayout.SetMargins(walk.Margins{5, 5, 5, 5})
	mtw.SetLayout(vlayout)
	mtw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		// "Close to tray" instead of exiting application
		*canceled = true
		if !noTrayAvailable {
			mtw.Hide()
		} else {
			win.ShowWindow(mtw.Handle(), win.SW_MINIMIZE)
		}
	})
	mtw.VisibleChanged().Attach(func() {
		if mtw.Visible() {
			win.SetForegroundWindow(mtw.Handle())
			win.BringWindowToTop(mtw.Handle())
		}
	})

	if mtw.tabs, err = walk.NewTabWidget(mtw); err != nil {
		return nil, err
	}

	if mtw.keysPage, err = NewKeysPage(keyManager); err != nil {
		return nil, err
	}
	mtw.tabs.Pages().Add(mtw.keysPage.TabPage)
	mtw.keysPage.CreateToolbar()

	systemMenu := win.GetSystemMenu(mtw.Handle(), false)
	if systemMenu != 0 {
		win.InsertMenuItem(systemMenu, 0, true, &win.MENUITEMINFO{
			CbSize:     uint32(unsafe.Sizeof(win.MENUITEMINFO{})),
			FMask:      win.MIIM_ID | win.MIIM_STRING | win.MIIM_FTYPE,
			FType:      win.MIIM_STRING,
			DwTypeData: windows.StringToUTF16Ptr(fmt.Sprintf("&About nCryptAgent…")),
			WID:        uint32(aboutWireGuardCmd),
		})
		win.InsertMenuItem(systemMenu, 1, true, &win.MENUITEMINFO{
			CbSize: uint32(unsafe.Sizeof(win.MENUITEMINFO{})),
			FMask:  win.MIIM_TYPE,
			FType:  win.MFT_SEPARATOR,
		})
	}

	disposables.Spare()

	return mtw, nil
}

//func (mtw *ManageTunnelsWindow) Dispose() {
//    mtw.FormBase.Dispose()
//}

//func (mtw *ManageTunnelsWindow) WndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
//    switch msg {
//    case win.WM_QUERYENDSESSION:
//        if lParam == win.ENDSESSION_CLOSEAPP {
//            return win.TRUE
//        }
//    case win.WM_ENDSESSION:
//        if lParam == win.ENDSESSION_CLOSEAPP && wParam == 1 {
//            walk.App().Exit(198)
//        }
//    case win.WM_SYSCOMMAND:
//        if wParam == aboutWireGuardCmd {
//            onAbout(mtw)
//            return 0
//        }
//    case raiseMsg:
//        if mtw.tunnelsPage == nil || mtw.tabs == nil {
//            mtw.Synchronize(func() {
//                mtw.SendMessage(msg, wParam, lParam)
//            })
//            return 0
//        }
//        if !mtw.Visible() {
//            mtw.tunnelsPage.listView.SelectFirstActiveTunnel()
//            if mtw.tabs.Pages().Len() != 3 {
//                mtw.tabs.SetCurrentIndex(0)
//            }
//        }
//        if mtw.tabs.Pages().Len() == 3 {
//            mtw.tabs.SetCurrentIndex(2)
//        }
//        raise(mtw.Handle())
//        return 0
//    case taskbarButtonCreatedMsg:
//        ret := mtw.FormBase.WndProc(hwnd, msg, wParam, lParam)
//        go func() {
//            globalState, err := manager.IPCClientGlobalState()
//            if err == nil {
//                mtw.Synchronize(func() {
//                    mtw.updateProgressIndicator(globalState)
//                })
//            }
//        }()
//        return ret
//    }
//
//    return mtw.FormBase.WndProc(hwnd, msg, wParam, lParam)
//}
