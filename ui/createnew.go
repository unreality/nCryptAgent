package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"ncryptagent/ncrypt"
)

var algorithmChoices = []string{
	"RSA-2048",
	"RSA-4096",
	"ECDSA-P256",
	"ECDSA-P384",
	"ECDSA-P521",
}

type CreateNewKey struct {
	*walk.Dialog
	nameEdit          *walk.LineEdit
	containerNameEdit *walk.LineEdit
	algorithmDropdown *walk.ComboBox
	keyLengthEdit     *walk.LineEdit

	saveButton   *walk.PushButton
	cancelButton *walk.PushButton

	config ncrypt.KeyConfig
}

func runCreateKeyDialog(owner walk.Form, km *ncrypt.KeyManager) *ncrypt.KeyConfig {
	dlg, err := newCreateKeyDialog(owner, km)
	if showError(err, owner) {
		return nil
	}

	if dlg.Run() == walk.DlgCmdOK {
		return &dlg.config
	}

	return nil
}

func newCreateKeyDialog(owner walk.Form, km *ncrypt.KeyManager) (*CreateNewKey, error) {
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
	//ddl := DropdownList{
	//	values: map[string]string{
	//		"RSA-2048":   "RSA 2048",
	//		"RSA-4096":   "RSA 4096",
	//		"ECDSA_P256": "ECDSA P256",
	//		"ECDSA_P384": "ECDSA P384",
	//		"ECDSA_P521": "ECDSAP521",
	//	},
	//}

	dlg.algorithmDropdown.SetModel(algorithmChoices)
	dlg.algorithmDropdown.SetCurrentIndex(0)
	layout.SetRange(dlg.algorithmDropdown, walk.Rectangle{1, 2, 1, 1})

	buttonsContainer, err := walk.NewComposite(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(buttonsContainer, walk.Rectangle{0, 3, 2, 1})
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

	var algorithm string
	var length int
	switch dlg.algorithmDropdown.Text() {
	case "RSA-2048":
		algorithm = "RSA"
		length = 2048
	case "RSA-4096":
		algorithm = "RSA"
		length = 4096
	case "ECDSA-P256":
		algorithm = "ECDSA_P256"
		length = 0
	case "ECDSA-P384":
		algorithm = "ECDSA_P384"
		length = 0
	case "ECDSA-P521":
		algorithm = "ECDSA_P521"
		length = 0
	default:
		algorithm = "RSA"
		length = 2048
	}

	dlg.config = ncrypt.KeyConfig{
		Name:          dlg.nameEdit.Text(),
		Type:          "NCRYPT",
		Algorithm:     algorithm,
		Length:        length,
		ContainerName: dlg.containerNameEdit.Text(),
		ProviderName:  ncrypt.ProviderMSSC,
		SSHPublicKey:  "",
	}

	dlg.Accept()
}
