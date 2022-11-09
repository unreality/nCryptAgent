/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"ncryptagent/keyman"
)

// ListModel is a struct to store the currently known keys to the GUI, suitable as a model for a walk.TableView.
type ListModel struct {
	walk.TableModelBase
	walk.SorterBase

	keys []*keyman.Key
}

func (t *ListModel) RowCount() int {
	return len(t.keys)
}

func (t *ListModel) Value(row, col int) interface{} {
	if col != 0 || row < 0 || row >= len(t.keys) {
		return ""
	}
	return t.keys[row].Name
}

type ListView struct {
	*walk.TableView
	model      *ListModel
	keyManager *keyman.KeyManager
}

func NewListView(parent walk.Container, keyManager *keyman.KeyManager) (*ListView, error) {
	var disposables walk.Disposables
	defer disposables.Treat()

	tv, err := walk.NewTableView(parent)
	if err != nil {
		return nil, err
	}
	disposables.Add(tv)

	tv.SetDoubleBuffering(true)

	model := new(ListModel)
	tv.SetModel(model)
	tv.SetLastColumnStretched(true)
	tv.SetHeaderHidden(true)
	tv.SetIgnoreNowhere(true)
	tv.SetScrollbarOrientation(walk.Vertical)
	tv.Columns().Add(walk.NewTableViewColumn())

	keysView := &ListView{
		TableView:  tv,
		model:      model,
		keyManager: keyManager,
	}
	tv.SetCellStyler(keysView)

	disposables.Spare()

	return keysView, nil
}

func (tv *ListView) CurrentKey() *keyman.Key {
	idx := tv.CurrentIndex()
	if idx == -1 {
		return nil
	}

	return tv.model.keys[idx]
}

var dummyBitmap *walk.Bitmap

func (tv *ListView) StyleCell(style *walk.CellStyle) {
	row := style.Row()
	if row < 0 || row >= len(tv.model.keys) {
		return
	}
	key := tv.model.keys[row]
	var icon *walk.Icon
	var err error

	//https://diymediahome.org/windows-icons-reference-list-with-details-locations-images/
	if key.LoadError != nil {
		icon, err = loadSystemIcon("imageres", 100, 32) // Cross Shield
	} else {
		icon, err = loadSystemIcon("imageres", 77, 32) // Key
	}
	//icon, err := loadSystemIcon("imageres", 54, 32) // Padlock
	//icon, err := loadSystemIcon("imageres", 77, 32) // Key
	//icon, err := loadSystemIcon("imageres", 73, 32) // UAC Shield
	//icon, err := loadSystemIcon("wmploc", 14, 32) // UAC Shield

	if err != nil {
		return
	}
	margin := tv.IntFrom96DPI(1)
	bitmapWidth := tv.IntFrom96DPI(16)

	if win.IsAppThemed() {
		bitmap, err := walk.NewBitmapWithTransparentPixelsForDPI(walk.Size{bitmapWidth, bitmapWidth}, tv.DPI())
		if err != nil {
			return
		}
		canvas, err := walk.NewCanvasFromImage(bitmap)
		if err != nil {
			return
		}
		bounds := walk.Rectangle{X: margin, Y: margin, Height: bitmapWidth - 2*margin, Width: bitmapWidth - 2*margin}
		err = canvas.DrawImageStretchedPixels(icon, bounds)
		canvas.Dispose()
		if err != nil {
			return
		}
		//cachedListViewIconsForWidthAndState[cacheKey] = bitmap
		style.Image = bitmap
	} else {
		if dummyBitmap == nil {
			dummyBitmap, _ = walk.NewBitmapForDPI(tv.SizeFrom96DPI(walk.Size{}), 96)
		}
		style.Image = dummyBitmap
		canvas := style.Canvas()
		if canvas == nil {
			return
		}
		bounds := style.BoundsPixels()
		bounds.Width = bitmapWidth - 2*margin
		bounds.X = (bounds.Height - bounds.Width) / 2
		bounds.Height = bounds.Width
		bounds.Y += bounds.X
		canvas.DrawImageStretchedPixels(icon, bounds)
	}
}

func (tv *ListView) Load(asyncUI bool) {

	keys := make([]*keyman.Key, 0, len(tv.keyManager.Keys))

	for _, k := range tv.keyManager.Keys {
		keys = append(keys, k)
	}

	doUI := func() {
		tv.model.keys = keys
		tv.model.PublishRowsReset()
		if len(keys) > 0 {
			tv.selectKey(keys[0].Name)
		}
	}
	if asyncUI {
		tv.Synchronize(doUI)
	} else {
		doUI()
	}
}

func (tv *ListView) selectKey(tunnelName string) {
	for i, tunnel := range tv.model.keys {
		if tunnel.Name == tunnelName {
			tv.SetCurrentIndex(i)
			break
		}
	}
}
