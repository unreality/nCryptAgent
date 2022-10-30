package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"ncryptagent/ncrypt"
	"ncryptagent/webauthn"
)

type KeysPage struct {
	*walk.TabPage

	listView      *ListView
	listContainer walk.Container
	listToolbar   *walk.ToolBar
	keyView       *KeyView
	fillerButton  *walk.PushButton
	fillerHandler func()

	fillerContainer     *walk.Composite
	currentKeyContainer *walk.Composite
	keyManager          *ncrypt.KeyManager
}

func NewKeysPage(keyManager *ncrypt.KeyManager) (*KeysPage, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	kp := new(KeysPage)

	kp.keyManager = keyManager

	if kp.TabPage, err = walk.NewTabPage(); err != nil {
		return nil, err
	}
	disposables.Add(kp)

	kp.SetTitle(fmt.Sprintf("Manage Keys"))
	kp.SetLayout(walk.NewHBoxLayout())

	kp.listContainer, _ = walk.NewComposite(kp)
	vlayout := walk.NewVBoxLayout()
	vlayout.SetMargins(walk.Margins{})
	vlayout.SetSpacing(0)
	kp.listContainer.SetLayout(vlayout)

	if kp.listView, err = NewListView(kp.listContainer, keyManager); err != nil {
		return nil, err
	}

	if kp.currentKeyContainer, err = walk.NewComposite(kp); err != nil {
		return nil, err
	}
	vlayout = walk.NewVBoxLayout()
	vlayout.SetMargins(walk.Margins{})
	kp.currentKeyContainer.SetLayout(vlayout)

	if kp.fillerContainer, err = walk.NewComposite(kp); err != nil {
		return nil, err
	}
	kp.fillerContainer.SetVisible(false)
	hlayout := walk.NewHBoxLayout()
	hlayout.SetMargins(walk.Margins{})
	kp.fillerContainer.SetLayout(hlayout)
	kp.fillerButton, _ = walk.NewPushButton(kp.fillerContainer)
	kp.fillerButton.SetMinMaxSize(walk.Size{200, 0}, walk.Size{200, 0})
	kp.fillerButton.Clicked().Attach(func() {
		if kp.fillerHandler != nil {
			kp.fillerHandler()
		}
	})

	if kp.keyView, err = NewKeyView(kp.currentKeyContainer); err != nil {
		return nil, err
	}

	controlsContainer, err := walk.NewComposite(kp.currentKeyContainer)
	if err != nil {
		return nil, err
	}
	controlsContainer.SetLayout(walk.NewHBoxLayout())
	controlsContainer.Layout().SetMargins(walk.Margins{})

	walk.NewHSpacer(controlsContainer)

	editTunnel, err := walk.NewPushButton(controlsContainer)
	if err != nil {
		return nil, err
	}
	editTunnel.SetEnabled(false)
	kp.listView.CurrentIndexChanged().Attach(func() {
		editTunnel.SetEnabled(kp.listView.CurrentIndex() > -1)
	})
	editTunnel.SetText(fmt.Sprintf("&Add Certificate"))
	editTunnel.Clicked().Attach(kp.onAddCertificate)

	disposables.Spare()

	//kp.listView.ItemCountChanged().Attach(kp.onTunnelsChanged)
	//kp.listView.SelectedIndexesChanged().Attach(kp.onSelectedTunnelsChanged)
	//kp.listView.ItemActivated().Attach(kp.onTunnelsViewItemActivated)
	kp.listView.CurrentIndexChanged().Attach(kp.updateKeyView)

	kp.listView.Load(false)

	return kp, nil
}

func (kp *KeysPage) CreateToolbar() error {
	if kp.listToolbar != nil {
		return nil
	}

	// HACK: Because of https://github.com/lxn/walk/issues/481
	// we need to put the ToolBar into its own Composite.
	toolBarContainer, err := walk.NewComposite(kp.listContainer)
	if err != nil {
		return err
	}
	toolBarContainer.SetDoubleBuffering(true)
	hlayout := walk.NewHBoxLayout()
	hlayout.SetMargins(walk.Margins{})
	toolBarContainer.SetLayout(hlayout)

	if kp.listToolbar, err = walk.NewToolBarWithOrientationAndButtonStyle(toolBarContainer, walk.Horizontal, walk.ToolBarButtonImageBeforeText); err != nil {
		return err
	}

	addMenu, err := walk.NewMenu()
	if err != nil {
		return err
	}
	kp.AddDisposable(addMenu)

	//createAction := walk.NewAction()
	//createAction.SetText(fmt.Sprintf("Create &nCrypt Key…"))
	//createActionIcon, _ := loadSystemIcon("imageres", 77, 16)
	//createAction.SetImage(createActionIcon)
	//createAction.Triggered().Attach(kp.onCreateKey)
	//addMenu.Actions().Add(createAction)

	createWebAuthN := walk.NewAction()
	createWebAuthN.SetText(fmt.Sprintf("Create &WebAuthN Key…"))
	createWebAuthNIcon, _ := loadSystemIcon("imageres", 77, 16)
	createWebAuthN.SetImage(createWebAuthNIcon)
	createWebAuthN.Triggered().Attach(kp.onCreateWebAuthNKey)
	addMenu.Actions().Add(createWebAuthN)

	createExistingAction := walk.NewAction()
	createExistingAction.SetText(fmt.Sprintf("Add &Existing nCrypt key…"))
	createExistingActionIcon, _ := loadSystemIcon("imageres", 172, 16)
	createExistingAction.SetImage(createExistingActionIcon)
	createExistingAction.Triggered().Attach(kp.onCreateExistingKey)
	addMenu.Actions().Add(createExistingAction)

	addMenuAction := walk.NewMenuAction(addMenu)
	addMenuActionIcon, _ := loadSystemIcon("shell32", 104, 16)
	addMenuAction.SetImage(addMenuActionIcon)
	addMenuAction.SetText(fmt.Sprintf("Add Key"))
	addMenuAction.Triggered().Attach(kp.onCreateExistingKey)
	kp.listToolbar.Actions().Add(addMenuAction)

	kp.listToolbar.Actions().Add(walk.NewSeparatorAction())

	deleteAction := walk.NewAction()
	deleteActionIcon, _ := loadSystemIcon("shell32", 131, 16)
	deleteAction.SetImage(deleteActionIcon)
	deleteAction.SetToolTip(fmt.Sprintf("Delete selected key(s)"))
	deleteAction.Triggered().Attach(kp.onDelete)
	kp.listToolbar.Actions().Add(deleteAction)
	kp.listToolbar.Actions().Add(walk.NewSeparatorAction())

	fixContainerWidthToToolbarWidth := func() {
		toolbarWidth := kp.listToolbar.SizeHint().Width
		kp.listContainer.SetMinMaxSizePixels(walk.Size{toolbarWidth, 0}, walk.Size{toolbarWidth, 0})
	}
	fixContainerWidthToToolbarWidth()
	kp.listToolbar.SizeChanged().Attach(fixContainerWidthToToolbarWidth)

	contextMenu, err := walk.NewMenu()
	if err != nil {
		return err
	}
	kp.listView.AddDisposable(contextMenu)

	//createNewAction2 := walk.NewAction()
	//createNewAction2.SetText(fmt.Sprintf("&Create new nCrypt Key…"))
	//createNewAction2.Triggered().Attach(kp.onCreateKey)
	//contextMenu.Actions().Add(createNewAction2)
	//kp.ShortcutActions().Add(createNewAction2)

	addWebAuthN2 := walk.NewAction()
	addWebAuthN2.SetText(fmt.Sprintf("&Create new WebAuthN Key…"))
	addWebAuthN2.Triggered().Attach(kp.onCreateWebAuthNKey)
	contextMenu.Actions().Add(addWebAuthN2)
	kp.ShortcutActions().Add(addWebAuthN2)

	createExistingAction2 := walk.NewAction()
	createExistingAction2.SetText(fmt.Sprintf("&Add existing key…"))
	createExistingAction2.Triggered().Attach(kp.onCreateExistingKey)
	contextMenu.Actions().Add(createExistingAction2)
	kp.ShortcutActions().Add(createExistingAction2)

	kp.listView.SetContextMenu(contextMenu)

	setSelectionOrientedOptions := func() {
		selected := len(kp.listView.SelectedIndexes())
		deleteAction.SetEnabled(selected > 0)
	}
	kp.listView.SelectedIndexesChanged().Attach(setSelectionOrientedOptions)
	setSelectionOrientedOptions()

	return nil
}

func (kp *KeysPage) updateKeyView() {
	kp.keyView.SetKey(kp.listView.CurrentKey())
}

func (kp *KeysPage) onCreateKey() {
	if config := runCreateKeyDialog(kp.Form(), kp.keyManager); config != nil {
		go func() {
			_, err := kp.keyManager.CreateNewNCryptKey(config.Name,
				config.ContainerName,
				config.ProviderName,
				config.Algorithm,
				config.Length,
			)

			if err != nil {
				showError(err, kp.Form())
			}

			kp.listView.Load(false)
		}()
	}
}

func (kp *KeysPage) onCreateWebAuthNKey() {
	if config := runCreateNewWebAuthNKeyDialog(kp.Form(), kp.keyManager); config != nil {
		go func() {

			coseAlgorithm := webauthn.COSE_ALGORITHM_ECDSA_P256_WITH_SHA256
			coseHash := webauthn.HASH_ALGORITHM_SHA_256

			switch config.Algorithm {
			case "sk-ecdsa-sha2-nistp256@openssh.com":
				coseAlgorithm = webauthn.COSE_ALGORITHM_ECDSA_P256_WITH_SHA256
			case "sk-ssh-ed25519@openssh.com":
				coseAlgorithm = webauthn.COSE_ALGORITHM_EDDSA_ED25519
			}

			fmt.Printf("Creating WebAuthN Key\n")
			_, err := kp.keyManager.CreateNewWebAuthNKey(config.Name,
				"ssh:",
				int64(coseAlgorithm),
				coseHash,
				uintptr(kp.Handle()),
			)

			if err != nil {
				showError(err, kp.Form())
			}

			kp.listView.Load(false)
		}()
	}
}

func (kp *KeysPage) onCreateExistingKey() {
	if config := runLoadExistingDialog(kp.Form(), kp.keyManager); config != nil {
		go func() {
			_, err := kp.keyManager.LoadNCryptKey(config)

			if err != nil {
				showError(err, kp.Form())
			}

			kp.listView.Load(false)
			kp.keyManager.SaveConfig()
		}()
	}
}

func (kp *KeysPage) onDelete() {
	confirmDelete, andFromKeystore := runDeleteKeyDialog(kp.Form(), kp.listView.CurrentKey().Name)

	if confirmDelete {
		err := kp.keyManager.DeleteKey(kp.listView.CurrentKey(), andFromKeystore)

		if err != nil {
			showError(err, kp.Form())
		}
		kp.listView.Load(false)
		kp.listView.SetCurrentIndex(0)

		if len(kp.keyManager.Keys) == 0 {
			kp.keyView.SetKey(nil)
			//kp.keyView.SetKey(&ncrypt.Key{
			//	Name:                 "No Keys",
			//	Type:                 "None",
			//	SSHPublicKey:         nil,
			//	SSHPublicKeyLocation: "",
			//	Missing:              true,
			//	LoadError:            nil,
			//})
		}
	}
}

func (kp *KeysPage) onDummy() {

}

func (kp *KeysPage) onAddCertificate() {
	dlg := walk.FileDialog{
		Filter: fmt.Sprintf("OpenSSH Key Files (*.pub)|*.pub|All Files (*.*)|*.*"),
		Title:  fmt.Sprintf("Attach certificate to key"),
	}

	if ok, _ := dlg.ShowOpen(kp.Form()); !ok {
		return
	}

	k := kp.listView.CurrentKey()
	err := k.LoadCertificate(dlg.FilePath)

	kp.keyView.SetKey(k)

	if err != nil {
		showError(err, kp.Form())
	}
}
