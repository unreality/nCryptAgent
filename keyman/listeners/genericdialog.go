package listeners

import (
	"github.com/lxn/walk"
)

type GenericDlg struct {
	*walk.Dialog
	MessageLabel *walk.LinkLabel
	ButtonOne    *walk.PushButton
	ButtonTwo    *walk.PushButton
}

//func RunGenericDialog(owner walk.Form, keyName string) (bool, bool) {
//    dlg, err := NewGenericDialog(owner, keyName)
//    if showError(err, owner) {
//        return false, false
//    }
//
//    if dlg.Run() == walk.DlgCmdOK {
//        return true, dlg.doDeleteFromKeystore
//    }
//
//    return false, false
//}

func NewGenericDialog(owner walk.Form, title string, message string, buttonOneText string, buttonTwoText string) (*GenericDlg, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	dlg := new(GenericDlg)

	layout := walk.NewGridLayout()
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	if dlg.Dialog, err = walk.NewDialog(owner); err != nil {
		return nil, err
	}
	disposables.Add(dlg)
	//	dlg.SetIcon(owner.Icon())
	dlg.SetTitle(title)
	dlg.SetLayout(layout)
	dlg.SetMinMaxSize(walk.Size{500, 200}, walk.Size{0, 0})
	//if icon, err := ui.loadSystemIcon("imageres", 109, 32); err == nil {
	//    dlg.SetIcon(icon)
	//}

	dlg.MessageLabel, err = walk.NewLinkLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(dlg.MessageLabel, walk.Rectangle{0, 0, 2, 2})
	//dlg.MessageLabel.SetTextAlignment(walk.AlignHNearVCenter)
	dlg.MessageLabel.SetText(message)
	dlg.MessageLabel.SetAlignment(walk.AlignHNearVNear)

	dlg.MessageLabel.SetVisible(true)

	buttonsContainer, err := walk.NewComposite(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(buttonsContainer, walk.Rectangle{0, 3, 2, 1})
	buttonsContainer.SetLayout(walk.NewHBoxLayout())
	buttonsContainer.Layout().SetMargins(walk.Margins{})

	walk.NewHSpacer(buttonsContainer)
	if dlg.ButtonOne, err = walk.NewPushButton(buttonsContainer); err != nil {
		return nil, err
	}
	dlg.ButtonOne.SetText(buttonOneText)

	dlg.ButtonTwo, err = walk.NewPushButton(buttonsContainer)
	if err != nil {
		return nil, err
	}
	dlg.ButtonTwo.SetText(buttonTwoText)

	dlg.SetCancelButton(dlg.ButtonTwo)
	dlg.SetDefaultButton(dlg.ButtonOne)

	disposables.Spare()

	return dlg, nil
}
