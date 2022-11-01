package ui

import (
	"fmt"
	"github.com/lxn/walk"
)

type DeleteKey struct {
	*walk.Dialog
	confirmLabel       *walk.TextLabel
	warningLabel       *walk.TextLabel
	deleteFromKeystore *walk.CheckBox

	deleteButton         *walk.PushButton
	cancelButton         *walk.PushButton
	doDeleteFromKeystore bool
}

func runDeleteKeyDialog(owner walk.Form, keyName string) (bool, bool) {
	dlg, err := newDeleteKeyDialog(owner, keyName)
	if showError(err, owner) {
		return false, false
	}

	if dlg.Run() == walk.DlgCmdOK {
		return true, dlg.doDeleteFromKeystore
	}

	return false, false
}

func newDeleteKeyDialog(owner walk.Form, keyName string) (*DeleteKey, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	dlg := new(DeleteKey)

	layout := walk.NewGridLayout()
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	if dlg.Dialog, err = walk.NewDialog(owner); err != nil {
		return nil, err
	}
	disposables.Add(dlg)
	dlg.SetIcon(owner.Icon())
	dlg.SetTitle("Delete key")
	dlg.SetLayout(layout)
	dlg.SetMinMaxSize(walk.Size{500, 200}, walk.Size{0, 0})
	if icon, err := loadSystemIcon("imageres", 109, 32); err == nil {
		dlg.SetIcon(icon)
	}

	dlg.confirmLabel, err = walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(dlg.confirmLabel, walk.Rectangle{0, 0, 2, 1})
	dlg.confirmLabel.SetTextAlignment(walk.AlignHNearVCenter)
	dlg.confirmLabel.SetText(fmt.Sprintf("Are you sure you want to delete key \"%s\"?", keyName))
	dlg.confirmLabel.SetVisible(true)

	deleteLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(deleteLabel, walk.Rectangle{0, 1, 1, 1})
	deleteLabel.SetTextAlignment(walk.AlignHNearVCenter)
	deleteLabel.SetText(fmt.Sprintf("&Delete from keystore:"))

	if dlg.deleteFromKeystore, err = walk.NewCheckBox(dlg); err != nil {
		return nil, err
	}
	layout.SetRange(dlg.deleteFromKeystore, walk.Rectangle{1, 1, 1, 1})
	dlg.deleteFromKeystore.SetChecked(false)
	dlg.deleteFromKeystore.CheckedChanged().Attach(dlg.onDeleteChecked)
	dlg.deleteFromKeystore.SetAlignment(walk.AlignHNearVCenter)

	dlg.warningLabel, err = walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(dlg.warningLabel, walk.Rectangle{0, 2, 2, 1})
	dlg.warningLabel.SetTextAlignment(walk.AlignHNearVCenter)
	dlg.warningLabel.SetText(fmt.Sprintf("WARNING: Deleting the key from the keystore PERMANENTLY removes the private key!"))
	dlg.warningLabel.SetTextColor(walk.RGB(255, 0, 0))
	dlg.warningLabel.SetVisible(false)

	buttonsContainer, err := walk.NewComposite(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(buttonsContainer, walk.Rectangle{0, 3, 2, 1})
	buttonsContainer.SetLayout(walk.NewHBoxLayout())
	buttonsContainer.Layout().SetMargins(walk.Margins{})

	walk.NewHSpacer(buttonsContainer)
	if dlg.deleteButton, err = walk.NewPushButton(buttonsContainer); err != nil {
		return nil, err
	}
	dlg.deleteButton.SetText(fmt.Sprintf("&Delete"))
	dlg.deleteButton.Clicked().Attach(dlg.Accept)

	cancelButton, err := walk.NewPushButton(buttonsContainer)
	if err != nil {
		return nil, err
	}
	cancelButton.SetText(fmt.Sprintf("Cancel"))
	cancelButton.Clicked().Attach(dlg.Cancel)

	dlg.SetCancelButton(cancelButton)
	dlg.SetDefaultButton(dlg.deleteButton)

	disposables.Spare()

	return dlg, nil
}

func (dlg *DeleteKey) onDeleteChecked() {
	dlg.warningLabel.SetVisible(dlg.deleteFromKeystore.Checked())
	dlg.doDeleteFromKeystore = dlg.deleteFromKeystore.Checked()
}
