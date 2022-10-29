package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

var showingAboutDialog *walk.Dialog

func onAbout(owner walk.Form) {
	showError(runAboutDialog(owner), owner)
}

func runAboutDialog(owner walk.Form) error {
	if showingAboutDialog != nil {
		showingAboutDialog.Show()
		raise(showingAboutDialog.Handle())
		return nil
	}

	vbl := walk.NewVBoxLayout()
	vbl.SetMargins(walk.Margins{80, 20, 80, 20})
	vbl.SetSpacing(10)

	var disposables walk.Disposables
	defer disposables.Treat()

	var err error
	showingAboutDialog, err = walk.NewDialogWithFixedSize(owner)
	if err != nil {
		return err
	}
	defer func() {
		showingAboutDialog = nil
	}()
	disposables.Add(showingAboutDialog)
	showingAboutDialog.SetTitle(fmt.Sprintf("About nCryptAgent"))
	showingAboutDialog.SetLayout(vbl)
	if icon, err := loadLogoIcon(32); err == nil {
		showingAboutDialog.SetIcon(icon)
	}

	font, _ := walk.NewFont("Segoe UI", 9, 0)
	showingAboutDialog.SetFont(font)

	iv, err := walk.NewImageView(showingAboutDialog)
	if err != nil {
		return err
	}

	if logo, err := loadLogoIcon(128); err == nil {
		iv.SetImage(logo)
	}
	iv.Accessibility().SetName(fmt.Sprintf("nCryptAgent image"))

	wgLbl, err := walk.NewTextLabel(showingAboutDialog)
	if err != nil {
		return err
	}
	wgFont, _ := walk.NewFont("Segoe UI", 16, walk.FontBold)
	wgLbl.SetFont(wgFont)
	wgLbl.SetTextAlignment(walk.AlignHCenterVNear)
	wgLbl.SetText("nCryptAgent")

	detailsLbl, err := walk.NewTextLabel(showingAboutDialog)
	if err != nil {
		return err
	}
	detailsLbl.SetTextAlignment(walk.AlignHCenterVNear)
	detailsLbl.SetText(
		fmt.Sprintf("An SSH Agent for hardware backed keys on Windows\n\nWith ♥ from the nCryptAgent Contributors\n\nIcon designed by 'Icon Home' on Flaticon\n"),
	)

	copyrightLbl, err := walk.NewTextLabel(showingAboutDialog)
	if err != nil {
		return err
	}
	copyrightFont, _ := walk.NewFont("Segoe UI", 7, 0)
	copyrightLbl.SetFont(copyrightFont)
	copyrightLbl.SetTextAlignment(walk.AlignHCenterVNear)
	copyrightLbl.SetText("Copyright © 2022 The nCryptAgent Contributors. All Rights Reserved.")

	buttonCP, err := walk.NewComposite(showingAboutDialog)
	if err != nil {
		return err
	}
	hbl := walk.NewHBoxLayout()
	hbl.SetMargins(walk.Margins{VNear: 10})
	buttonCP.SetLayout(hbl)
	walk.NewHSpacer(buttonCP)
	closePB, err := walk.NewPushButton(buttonCP)
	if err != nil {
		return err
	}
	closePB.SetAlignment(walk.AlignHCenterVNear)
	closePB.SetText("Close")
	closePB.Clicked().Attach(showingAboutDialog.Accept)

	websitePB, err := walk.NewPushButton(buttonCP)

	if err != nil {
		return err
	}
	websitePB.SetAlignment(walk.AlignHCenterVNear)
	websitePB.SetText(fmt.Sprintf("Go to website"))
	websitePB.Clicked().Attach(func() {
		win.ShellExecute(showingAboutDialog.Handle(), nil, windows.StringToUTF16Ptr("https://github.com/unreality/nCryptAgent"), nil, nil, win.SW_SHOWNORMAL)
		showingAboutDialog.Accept()
	})

	walk.NewHSpacer(buttonCP)

	showingAboutDialog.SetCancelButton(closePB)

	disposables.Spare()

	showingAboutDialog.Run()

	return nil
}
