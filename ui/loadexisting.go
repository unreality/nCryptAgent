package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"ncryptagent/ncrypt"
	"ncryptagent/scard"
	"sync"
)

type LoadExistingKey struct {
	*walk.Dialog
	nameEdit         *walk.LineEdit
	cardReaderSelect *walk.ComboBox
	containerSelect  *walk.ComboBox

	saveButton   *walk.PushButton
	cancelButton *walk.PushButton

	config        ncrypt.KeyConfig
	scardReaders  []string
	containerList []ncrypt.NCryptKeyDescriptor
	onChangeMU    sync.Mutex
}

func runLoadExistingDialog(owner walk.Form, km *ncrypt.KeyManager) *ncrypt.KeyConfig {
	dlg, err := newLoadExistingDialog(owner, km)
	if showError(err, owner) {
		return nil
	}

	if dlg.Run() == walk.DlgCmdOK {
		return &dlg.config
	}

	return nil
}

func newLoadExistingDialog(owner walk.Form, km *ncrypt.KeyManager) (*LoadExistingKey, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	dlg := new(LoadExistingKey)

	scardReaders, err := scard.SCardListReaders()
	if err != nil {
		return nil, err
	}

	dlg.scardReaders = scardReaders

	layout := walk.NewGridLayout()
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	if dlg.Dialog, err = walk.NewDialog(owner); err != nil {
		return nil, err
	}
	disposables.Add(dlg)
	dlg.SetIcon(owner.Icon())
	dlg.SetTitle("Load existing key…")
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

	//Setup the cardReader list dropdown
	cardReaderLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(cardReaderLabel, walk.Rectangle{0, 1, 1, 1})
	cardReaderLabel.SetTextAlignment(walk.AlignHFarVCenter)
	cardReaderLabel.SetText(fmt.Sprintf("&Card Reader:"))

	if dlg.cardReaderSelect, err = walk.NewDropDownBox(dlg); err != nil {
		return nil, err
	}
	dlg.cardReaderSelect.SetModel(scardReaders)
	dlg.cardReaderSelect.SetCurrentIndex(0)
	dlg.cardReaderSelect.CurrentIndexChanged().Attach(dlg.onCardChange)
	dlg.cardReaderSelect.SetAlignment(walk.AlignHFarVCenter)
	layout.SetRange(dlg.cardReaderSelect, walk.Rectangle{1, 1, 1, 1})

	//Setup the containerSelect list dropdown
	containerSelectLabel, err := walk.NewTextLabel(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(containerSelectLabel, walk.Rectangle{0, 2, 1, 1})
	containerSelectLabel.SetTextAlignment(walk.AlignHFarVCenter)
	containerSelectLabel.SetText(fmt.Sprintf("&Container:"))

	if dlg.containerSelect, err = walk.NewDropDownBox(dlg); err != nil {
		return nil, err
	}
	dlg.containerSelect.SetModel([]string{"Select card reader..."})
	dlg.containerSelect.SetEnabled(false)
	dlg.containerSelect.SetCurrentIndex(0)
	dlg.containerSelect.SetAlignment(walk.AlignHFarVCenter)
	layout.SetRange(dlg.containerSelect, walk.Rectangle{1, 2, 1, 1})

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

func (dlg *LoadExistingKey) onSaveButtonClicked() {
	var algorithm string
	for _, d := range dlg.containerList {
		if d.Container == dlg.containerSelect.Text() {
			algorithm = d.Algorithm
		}
	}

	dlg.config = ncrypt.KeyConfig{
		Name:          dlg.nameEdit.Text(),
		Type:          "NCRYPT",
		ContainerName: dlg.containerSelect.Text(),
		Algorithm:     algorithm,
		ProviderName:  ncrypt.ProviderMSSC,
		SSHPublicKey:  "",
	}

	dlg.Accept()
}

func (dlg *LoadExistingKey) onCardChange() {
	if !dlg.onChangeMU.TryLock() {
		return
	}
	defer dlg.onChangeMU.Unlock()

	var err error
	dlg.containerList, err = ncrypt.ListContainersOnCard(ncrypt.ProviderMSSC, dlg.cardReaderSelect.Text())

	if err != nil {
		showError(err, dlg)
	}

	var containerList []string
	for _, d := range dlg.containerList {
		containerList = append(containerList, d.Container)
	}

	dlg.containerSelect.SetModel(containerList)
	dlg.containerSelect.SetEnabled(len(containerList) > 0)
	dlg.containerSelect.SetCurrentIndex(0)
}
