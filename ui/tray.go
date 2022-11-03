/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"fmt"
	"github.com/lxn/walk"
)

type Tray struct {
	*walk.NotifyIcon

	mtw     *ManageKeysWindow
	clicked func()
}

func NewTray(mtw *ManageKeysWindow) (*Tray, error) {
	var err error

	tray := &Tray{
		mtw: mtw,
	}

	tray.NotifyIcon, err = walk.NewNotifyIcon(mtw)
	if err != nil {
		return nil, err
	}

	return tray, tray.setup()
}

func (tray *Tray) setup() error {
	tray.clicked = tray.onManageKeys

	tray.SetToolTip(fmt.Sprintf("nCryptAgent - Running"))
	tray.SetVisible(true)
	if icon, err := loadLogoIcon(16); err == nil {
		tray.SetIcon(icon)
	}

	tray.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			tray.clicked()
		}
	})
	tray.MessageClicked().Attach(func() {
		tray.clicked()
	})

	for _, item := range [...]struct {
		label     string
		handler   walk.EventHandler
		enabled   bool
		hidden    bool
		separator bool
		defawlt   bool
	}{
		{label: fmt.Sprintf("&Manage keys…"), handler: tray.onManageKeys, enabled: true, defawlt: true},
		{separator: true},
		{label: fmt.Sprintf("&About nCryptAgent…"), handler: tray.onAbout, enabled: true},
		{label: fmt.Sprintf("E&xit"), handler: onQuit, enabled: true},
	} {
		var action *walk.Action
		if item.separator {
			action = walk.NewSeparatorAction()
		} else {
			action = walk.NewAction()
			action.SetText(item.label)
			action.SetEnabled(item.enabled)
			action.SetVisible(!item.hidden)
			action.SetDefault(item.defawlt)
			if item.handler != nil {
				action.Triggered().Attach(item.handler)
			}
		}

		tray.ContextMenu().Actions().Add(action)
	}

	return nil
}

func (tray *Tray) Dispose() error {
	return tray.NotifyIcon.Dispose()
}

func (tray *Tray) onManageKeys() {
	raise(tray.mtw.Handle())
}

func (tray *Tray) onAbout() {
	if tray.mtw.Visible() {
		onAbout(tray.mtw)
	} else {
		onAbout(nil)
	}
}
