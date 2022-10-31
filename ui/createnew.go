package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"ncryptagent/keyman"
	"ncryptagent/ncrypt"
)

var algorithmChoices = []string{
	"RSA-2048",
	"RSA-4096",
	"ECDSA-P256",
	"ECDSA-P384",
	"ECDSA-P521",
}

type NewKeyConfig struct {
	Name          string
	Type          string
	ContainerName string
	ProviderName  string
	Password      string
	Algorithm     string
}

type CreateNewKey struct {
	*walk.Dialog
	nameEdit          *walk.LineEdit
	containerNameEdit *walk.LineEdit
	algorithmDropdown *walk.ComboBox
	keyLengthEdit     *walk.LineEdit

	saveButton   *walk.PushButton
	cancelButton *walk.PushButton

	config       NewKeyConfig
	passwordEdit *walk.LineEdit
}

func runCreateKeyDialog(owner walk.Form, km *keyman.KeyManager) *NewKeyConfig {
	dlg, err := newCreateKeyDialog(owner, km)
	if showError(err, owner) {
		return nil
	}

	if dlg.Run() == walk.DlgCmdOK {
		return &dlg.config
	}

	return nil
}

func newCreateKeyDialog(owner walk.Form, km *keyman.KeyManager) (*CreateNewKey, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	dlg := new(CreateNewKey)

	layout := walk.NewGridLayout()
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	if dlg.Dialog, err = walk.NewDialog(owner); err != nil {
		return nil, err
	}
	disposables.Add(dlg)
	dlg.SetIcon(owner.Icon())
	dlg.SetTitle("Create new key")
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

	//Setup the containerName
	containerLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(containerLabel, walk.Rectangle{0, 1, 1, 1})
	containerLabel.SetTextAlignment(walk.AlignHFarVCenter)
	containerLabel.SetText(fmt.Sprintf("&Container Name:"))

	if dlg.containerNameEdit, err = walk.NewLineEdit(dlg); err != nil {
		return nil, err
	}
	layout.SetRange(dlg.containerNameEdit, walk.Rectangle{1, 1, 1, 1})
	err = dlg.containerNameEdit.SetText(dlg.config.ContainerName)
	dlg.containerNameEdit.SetAlignment(walk.AlignHFarVCenter)

	//Setup the password field
	passwordLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(passwordLabel, walk.Rectangle{0, 2, 1, 1})
	passwordLabel.SetTextAlignment(walk.AlignHFarVCenter)
	passwordLabel.SetText(fmt.Sprintf("&Password/PIN:"))

	if dlg.passwordEdit, err = walk.NewLineEdit(dlg); err != nil {
		return nil, err
	}
	layout.SetRange(dlg.passwordEdit, walk.Rectangle{1, 2, 1, 1})
	err = dlg.passwordEdit.SetText(dlg.config.Password)
	dlg.passwordEdit.SetPasswordMode(true)
	dlg.passwordEdit.SetAlignment(walk.AlignHFarVCenter)

	//Setup the algorithm list dropdown
	algorithmLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(algorithmLabel, walk.Rectangle{0, 3, 1, 1})
	algorithmLabel.SetTextAlignment(walk.AlignHFarVCenter)
	algorithmLabel.SetText(fmt.Sprintf("&Key Algorithm:"))
	if dlg.algorithmDropdown, err = walk.NewDropDownBox(dlg); err != nil {
		return nil, err
	}

	dlg.algorithmDropdown.SetModel(algorithmChoices)
	dlg.algorithmDropdown.SetCurrentIndex(0)
	layout.SetRange(dlg.algorithmDropdown, walk.Rectangle{1, 3, 1, 1})

	buttonsContainer, err := walk.NewComposite(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(buttonsContainer, walk.Rectangle{0, 4, 2, 1})
	buttonsContainer.SetLayout(walk.NewHBoxLayout())
	buttonsContainer.Layout().SetMargins(walk.Margins{})

	walk.NewHSpacer(buttonsContainer)
	if dlg.saveButton, err = walk.NewPushButton(buttonsContainer); err != nil {
		return nil, err
	}
	dlg.saveButton.SetText(fmt.Sprintf("&Save"))
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

func (dlg *CreateNewKey) onSaveButtonClicked() {

	dlg.config = NewKeyConfig{
		Name:          dlg.nameEdit.Text(),
		Type:          "NCRYPT",
		Algorithm:     dlg.algorithmDropdown.Text(),
		ContainerName: dlg.containerNameEdit.Text(),
		Password:      dlg.passwordEdit.Text(),
		ProviderName:  ncrypt.ProviderMSPlatform,
	}

	dlg.Accept()
}
