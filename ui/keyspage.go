package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"ncryptagent/ncrypt"
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

	createAction := walk.NewAction()
	createAction.SetText(fmt.Sprintf("Create &NCrypt Key…"))
	importActionIcon, _ := loadSystemIcon("imageres", 3, 16)
	createAction.SetImage(importActionIcon)
	createAction.SetShortcut(walk.Shortcut{walk.ModControl, walk.KeyN})
	createAction.SetDefault(true)
	createAction.Triggered().Attach(kp.onCreateKey)
	addMenu.Actions().Add(createAction)

	createExistingAction := walk.NewAction()
	createExistingAction.SetText(fmt.Sprintf("Add &Existing key…"))
	createExistingActionIcon, _ := loadSystemIcon("imageres", 3, 16)
	createExistingAction.SetImage(createExistingActionIcon)
	createExistingAction.SetShortcut(walk.Shortcut{walk.ModControl, walk.KeyE})
	createExistingAction.SetDefault(true)
	createExistingAction.Triggered().Attach(kp.onCreateExistingKey)
	addMenu.Actions().Add(createExistingAction)

	createFIDOAction := walk.NewAction()
	createFIDOAction.SetText(fmt.Sprintf("Create &FIDO Key…"))
	addActionIcon, _ := loadSystemIcon("imageres", 2, 16)
	createFIDOAction.SetImage(addActionIcon)
	createFIDOAction.SetShortcut(walk.Shortcut{walk.ModControl, walk.KeyF})
	createFIDOAction.Triggered().Attach(kp.onDummy)
	addMenu.Actions().Add(createFIDOAction)

	addMenuAction := walk.NewMenuAction(addMenu)
	addMenuActionIcon, _ := loadSystemIcon("shell32", 149, 16)
	addMenuAction.SetImage(addMenuActionIcon)
	addMenuAction.SetText(fmt.Sprintf("Create Key"))
	addMenuAction.SetToolTip(createAction.Text())
	addMenuAction.Triggered().Attach(kp.onCreateKey)
	kp.listToolbar.Actions().Add(addMenuAction)

	kp.listToolbar.Actions().Add(walk.NewSeparatorAction())

	deleteAction := walk.NewAction()
	deleteActionIcon, _ := loadSystemIcon("shell32", 131, 16)
	deleteAction.SetImage(deleteActionIcon)
	deleteAction.SetShortcut(walk.Shortcut{0, walk.KeyDelete})
	deleteAction.SetToolTip(fmt.Sprintf("Delete selected key(s)"))
	deleteAction.Triggered().Attach(kp.onDelete)
	kp.listToolbar.Actions().Add(deleteAction)
	kp.listToolbar.Actions().Add(walk.NewSeparatorAction())

	//exportAction := walk.NewAction()
	//exportActionIcon, _ := loadSystemIcon("imageres", 165, 16) // Or "shell32", 45?
	//exportAction.SetImage(exportActionIcon)
	//exportAction.SetToolTip(fmt.Sprintf("Export all keys to zip"))
	//exportAction.Triggered().Attach(kp.onDummy)
	//kp.listToolbar.Actions().Add(exportAction)

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

	importAction2 := walk.NewAction()
	importAction2.SetText(fmt.Sprintf("&Create new nCrypt Key…"))
	importAction2.SetShortcut(walk.Shortcut{walk.ModControl, walk.KeyN})
	importAction2.Triggered().Attach(kp.onCreateKey)
	contextMenu.Actions().Add(importAction2)
	kp.ShortcutActions().Add(importAction2)

	createExistingAction2 := walk.NewAction()
	createExistingAction2.SetText(fmt.Sprintf("&Add existing key…"))
	createExistingAction2.SetShortcut(walk.Shortcut{walk.ModControl, walk.KeyE})
	createExistingAction2.Triggered().Attach(kp.onCreateExistingKey)
	contextMenu.Actions().Add(createExistingAction2)
	kp.ShortcutActions().Add(createExistingAction2)

	createFIDOAction2 := walk.NewAction()
	createFIDOAction2.SetText(fmt.Sprintf("Add &FIDO Key…"))
	createFIDOAction2.SetShortcut(walk.Shortcut{walk.ModControl, walk.KeyN})
	createFIDOAction2.Triggered().Attach(kp.onDummy)
	contextMenu.Actions().Add(createFIDOAction2)
	kp.ShortcutActions().Add(createFIDOAction2)

	kp.listView.SetContextMenu(contextMenu)

	setSelectionOrientedOptions := func() {
		selected := len(kp.listView.SelectedIndexes())
		deleteAction.SetEnabled(selected > 0)
	}
	kp.listView.SelectedIndexesChanged().Attach(setSelectionOrientedOptions)
	setSelectionOrientedOptions()

	//setExport := func() {
	//    all := len(kp.listView.model.keys)
	//    //exportAction.SetEnabled(all > 0)
	//    exportAction2.SetEnabled(all > 0)
	//}
	//setExportRange := func(from, to int) { setExport() }
	//kp.listView.model.RowsInserted().Attach(setExportRange)
	//kp.listView.model.RowsRemoved().Attach(setExportRange)
	//kp.listView.model.RowsReset().Attach(setExport)
	//setExport()

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

func (kp *KeysPage) onCreateExistingKey() {
	if config := runLoadExistingDialog(kp.Form(), kp.keyManager); config != nil {
		go func() {
			_, err := kp.keyManager.LoadKey(config)

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
			kp.keyView.SetKey(&ncrypt.Key{
				Name:                 "No Keys",
				Type:                 "None",
				SSHPublicKey:         nil,
				SSHPublicKeyLocation: "",
				Missing:              true,
				LoadError:            nil,
			})
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
