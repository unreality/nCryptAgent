package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"ncryptagent/keyman"
)

var algorithmChoicesWebAuthN = []string{
	keyman.OPENSSH_SK_ECDSA,
	keyman.OPENSSH_SK_ED25519,
}

type NewWebAuthNConfig struct {
	Name           string
	Type           string
	Algorithm      string
	Resident       bool
	VerifyRequired bool
}

type CreateNewWebAuthNKey struct {
	*walk.Dialog
	nameEdit          *walk.LineEdit
	algorithmDropdown *walk.ComboBox
	verifyRequired    *walk.CheckBox
	residentRequired  *walk.CheckBox

	saveButton   *walk.PushButton
	cancelButton *walk.PushButton

	config NewWebAuthNConfig
}

func runCreateNewWebAuthNKeyDialog(owner walk.Form, km *keyman.KeyManager) *NewWebAuthNConfig {
	dlg, err := newCreateNewWebAuthNKeyDialog(owner, km)
	if showError(err, owner) {
		return nil
	}

	if dlg.Run() == walk.DlgCmdOK {
		return &dlg.config
	}

	return nil
}

func newCreateNewWebAuthNKeyDialog(owner walk.Form, km *keyman.KeyManager) (*CreateNewWebAuthNKey, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	dlg := new(CreateNewWebAuthNKey)

	layout := walk.NewGridLayout()
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	if dlg.Dialog, err = walk.NewDialog(owner); err != nil {
		return nil, err
	}
	disposables.Add(dlg)
	dlg.SetIcon(owner.Icon())
	dlg.SetTitle("Create new WebAuthN key")
	dlg.SetLayout(layout)
	dlg.SetMinMaxSize(walk.Size{500, 200}, walk.Size{0, 0})
	if icon, err := loadSystemIcon("imageres", 109, 32); err == nil {
		dlg.SetIcon(icon)
	}

	//Setup the name
	nameLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(nameLabel, walk.Rectangle{0, 0, 1, 1})
	nameLabel.SetTextAlignment(walk.AlignHFarVCenter)
	nameLabel.SetText(fmt.Sprintf("&Name:"))

	if dlg.nameEdit, err = walk.NewLineEdit(dlg); err != nil {
		return nil, err
	}
	layout.SetRange(dlg.nameEdit, walk.Rectangle{1, 0, 1, 1})
	dlg.nameEdit.SetText(dlg.config.Name)
	dlg.nameEdit.SetAlignment(walk.AlignHFarVCenter)

	//Setup the algorithm list dropdown
	algorithmLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(algorithmLabel, walk.Rectangle{0, 2, 1, 1})
	algorithmLabel.SetTextAlignment(walk.AlignHFarVCenter)
	algorithmLabel.SetText(fmt.Sprintf("&Key Algorithm:"))
	if dlg.algorithmDropdown, err = walk.NewDropDownBox(dlg); err != nil {
		return nil, err
	}

	dlg.algorithmDropdown.SetModel(algorithmChoicesWebAuthN)
	dlg.algorithmDropdown.SetCurrentIndex(0)
	layout.SetRange(dlg.algorithmDropdown, walk.Rectangle{1, 2, 1, 1})
	dlg.algorithmDropdown.SetAlignment(walk.AlignHFarVCenter)

	//verification required
	verifyRequiredLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(verifyRequiredLabel, walk.Rectangle{0, 3, 1, 1})
	verifyRequiredLabel.SetTextAlignment(walk.AlignHNearVCenter)
	verifyRequiredLabel.SetText(fmt.Sprintf("&User verification required:"))

	if dlg.verifyRequired, err = walk.NewCheckBox(dlg); err != nil {
		return nil, err
	}
	layout.SetRange(dlg.verifyRequired, walk.Rectangle{1, 3, 1, 1})
	dlg.verifyRequired.SetChecked(false)
	dlg.verifyRequired.SetAlignment(walk.AlignHNearVCenter)

	// resident required
	residentRequiredLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(residentRequiredLabel, walk.Rectangle{0, 4, 1, 1})
	residentRequiredLabel.SetTextAlignment(walk.AlignHNearVCenter)
	residentRequiredLabel.SetText(fmt.Sprintf("&Resident key required:"))

	if dlg.residentRequired, err = walk.NewCheckBox(dlg); err != nil {
		return nil, err
	}
	layout.SetRange(dlg.residentRequired, walk.Rectangle{1, 4, 1, 1})
	dlg.residentRequired.SetChecked(false)
	dlg.residentRequired.SetAlignment(walk.AlignHNearVCenter)

	buttonsContainer, err := walk.NewComposite(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(buttonsContainer, walk.Rectangle{0, 5, 2, 1})
	buttonsContainer.SetLayout(walk.NewHBoxLayout())
	buttonsContainer.Layout().SetMargins(walk.Margins{})

	walk.NewHSpacer(buttonsContainer)
	if dlg.saveButton, err = walk.NewPushButton(buttonsContainer); err != nil {
		return nil, err
	}
	dlg.saveButton.SetText(fmt.Sprintf("&Create"))
	dlg.saveButton.Clicked().Attach(dlg.onSaveButtonClicked)

	cancelButton, err := walk.NewPushButton(buttonsContainer)
	if err != nil {
		return nil, err
	}
	cancelButton.SetText(fmt.Sprintf("Cancel"))
	cancelButton.Clicked().Attach(dlg.Cancel)

	dlg.SetCancelButton(cancelButton)
	dlg.SetDefaultButton(dlg.saveButton)

	disposables.Spare()

	return dlg, nil

}

func (dlg *CreateNewWebAuthNKey) onSaveButtonClicked() {

	dlg.config = NewWebAuthNConfig{
		Name:           dlg.nameEdit.Text(),
		Type:           "WEBAUTHN",
		Algorithm:      dlg.algorithmDropdown.Text(),
		VerifyRequired: dlg.verifyRequired.Checked(),
		Resident:       dlg.residentRequired.Checked(),
	}

	dlg.Accept()
}
