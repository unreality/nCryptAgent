package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"github.com/lxn/win"
	"ncryptagent/ncrypt"
)

type widgetsLine interface {
	widgets() (walk.Widget, walk.Widget)
}

type widgetsLinesView interface {
	widgetsLines() []widgetsLine
}

type labelTextLineItem struct {
	label string
	ptr   **labelTextLine
}

type labelTextLine struct {
	label *walk.TextLabel
	text  *walk.TextEdit
}

func (lt *labelTextLine) widgets() (walk.Widget, walk.Widget) {
	return lt.label, lt.text
}

func (lt *labelTextLine) show(text string) {
	s, e := lt.text.TextSelection()
	lt.text.SetText(text)
	lt.label.SetVisible(true)
	lt.text.SetVisible(true)
	lt.text.SetTextSelection(s, e)
}

func (lt *labelTextLine) hide() {
	lt.text.SetText("")
	lt.label.SetVisible(false)
	lt.text.SetVisible(false)
}

func (lt *labelTextLine) Dispose() {
	lt.label.Dispose()
	lt.text.Dispose()
}

type keyInfoView struct {
	name                 *labelStatusLine
	keyType              *labelTextLine
	keyAlgorithm         *labelTextLine
	keyContainer         *labelTextLine
	keyFingerprint       *labelTextLine
	sshCertificateSerial *labelTextLine
	sshPublicKeyLocation *labelTextLine
	loadError            *labelTextLine

	copyPublicKeyLocation *keyActionsButtonLine
	lines                 []widgetsLine
	currentKey            *ncrypt.Key
}

func (kiv *keyInfoView) widgetsLines() []widgetsLine {
	return kiv.lines
}

func (kiv *keyInfoView) apply(ki *ncrypt.Key) {
	kiv.currentKey = ki
	kiv.keyType.show(ki.Type)
	kiv.keyAlgorithm.show(ki.SSHPublicKeyType())
	if ki.SSHPublicKey != nil {
		kiv.keyFingerprint.show(ki.SSHPublicKeyFingerprint())
	} else {
		kiv.keyFingerprint.hide()
	}

	if ki.SSHCertificate != nil {
		kiv.sshCertificateSerial.show(ki.SSHCertificateSerial())
	} else {
		kiv.sshCertificateSerial.hide()
	}
	kiv.keyContainer.show(ki.ContainerName())
	kiv.sshPublicKeyLocation.show(ki.SSHPublicKeyLocation)

	if ki.LoadError != nil {
		kiv.loadError.show(fmt.Sprintf("%s", ki.LoadError))
	} else {
		kiv.loadError.hide()
	}
}

func (kiv *keyInfoView) onCopyPublicKeyLocation() {
	walk.Clipboard().SetText(kiv.currentKey.SSHPublicKeyLocation)
}

func (kiv *keyInfoView) onCopyPublicKey() {
	walk.Clipboard().SetText(kiv.currentKey.SSHPublicKeyString())
}

func newKeyInfoView(parent walk.Container) (*keyInfoView, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	iv := new(keyInfoView)

	if iv.name, err = newLabelStatusLine(parent); err != nil {
		return nil, err
	}
	disposables.Add(iv.name)

	items := []labelTextLineItem{
		{fmt.Sprintf("Key Type:"), &iv.keyType},
		{fmt.Sprintf("Algorithm:"), &iv.keyAlgorithm},
		{fmt.Sprintf("Container Name:"), &iv.keyContainer},
		{fmt.Sprintf("Fingerprint:"), &iv.keyFingerprint},
		{fmt.Sprintf("Certificate Serial:"), &iv.sshCertificateSerial},
		{fmt.Sprintf("Public Key Location:"), &iv.sshPublicKeyLocation},
		{fmt.Sprintf("Errors:"), &iv.loadError},
	}
	if iv.lines, err = createLabelTextLines(items, parent, &disposables); err != nil {
		return nil, err
	}

	if iv.copyPublicKeyLocation, err = newPublicKeyActionsLine(parent); err != nil {
		return nil, err
	}
	iv.copyPublicKeyLocation.copyLocationToClipboard.Clicked().Attach(iv.onCopyPublicKeyLocation)
	iv.copyPublicKeyLocation.copyKeyToClipboard.Clicked().Attach(iv.onCopyPublicKey)
	disposables.Add(iv.copyPublicKeyLocation)

	iv.lines = append([]widgetsLine{iv.name}, append(iv.lines, iv.copyPublicKeyLocation)...)

	layoutInGrid(iv, parent.Layout().(*walk.GridLayout))

	disposables.Spare()

	return iv, nil
}

type KeyView struct {
	*walk.ScrollView
	name       *walk.GroupBox
	keyInfo    *keyInfoView
	currentKey *ncrypt.Key
}

func NewKeyView(parent walk.Container) (*KeyView, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	kv := new(KeyView)
	if kv.ScrollView, err = walk.NewScrollView(parent); err != nil {
		return nil, err
	}
	disposables.Add(kv)
	vlayout := walk.NewVBoxLayout()
	vlayout.SetMargins(walk.Margins{5, 0, 5, 0})
	kv.SetLayout(vlayout)

	if kv.name, err = newPaddedGroupGrid(kv); err != nil {
		return nil, err
	}
	if kv.keyInfo, err = newKeyInfoView(kv.name); err != nil {
		return nil, err
	}
	kv.SetKey(nil)

	if err := walk.InitWrapperWindow(kv); err != nil {
		return nil, err
	}
	kv.SetDoubleBuffering(true)

	disposables.Spare()

	return kv, nil
}

func (kv *KeyView) SetKey(k *ncrypt.Key) {
	kv.name.SetVisible(k != nil)
	if k == nil {
		return
	}
	kv.currentKey = k

	title := fmt.Sprintf("Key: %s", k.Name)
	if kv.name.Title() != title {
		kv.SetSuspended(true)
		defer kv.SetSuspended(false)
		kv.name.SetTitle(title)
	}

	kv.keyInfo.apply(k)
	kv.keyInfo.name.update(*k)
}

type labelStatusLine struct {
	label           *walk.TextLabel
	statusComposite *walk.Composite
	statusImage     *walk.ImageView
	statusLabel     *walk.LineEdit
}

func (lsl *labelStatusLine) widgets() (walk.Widget, walk.Widget) {
	return lsl.label, lsl.statusComposite
}

func (lsl *labelStatusLine) update(k ncrypt.Key) {
	var icon *walk.Icon
	var err error

	if k.Missing {
		icon, err = loadSystemIcon("imageres", 99, 14)
	} else {
		icon, err = loadSystemIcon("imageres", 101, 14)
	}

	if err == nil {
		lsl.statusImage.SetImage(icon)
	} else {
		lsl.statusImage.SetImage(nil)
	}

	s, e := lsl.statusLabel.TextSelection()

	if k.Missing {
		lsl.statusLabel.SetText("Missing")
	} else {
		lsl.statusLabel.SetText("Available")
	}
	lsl.statusLabel.SetTextSelection(s, e)
}

func (lsl *labelStatusLine) Dispose() {
	lsl.label.Dispose()
	lsl.statusComposite.Dispose()
}

func newLabelStatusLine(parent walk.Container) (*labelStatusLine, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	lsl := new(labelStatusLine)

	if lsl.label, err = walk.NewTextLabel(parent); err != nil {
		return nil, err
	}
	disposables.Add(lsl.label)
	lsl.label.SetText(fmt.Sprintf("Status:"))
	lsl.label.SetTextAlignment(walk.AlignHFarVNear)

	if lsl.statusComposite, err = walk.NewComposite(parent); err != nil {
		return nil, err
	}
	disposables.Add(lsl.statusComposite)
	layout := walk.NewHBoxLayout()
	layout.SetMargins(walk.Margins{})
	layout.SetAlignment(walk.AlignHNearVNear)
	layout.SetSpacing(0)
	lsl.statusComposite.SetLayout(layout)

	if lsl.statusImage, err = walk.NewImageView(lsl.statusComposite); err != nil {
		return nil, err
	}
	disposables.Add(lsl.statusImage)
	lsl.statusImage.SetMargin(2)
	lsl.statusImage.SetMode(walk.ImageViewModeIdeal)

	if lsl.statusLabel, err = walk.NewLineEdit(lsl.statusComposite); err != nil {
		return nil, err
	}
	disposables.Add(lsl.statusLabel)
	win.SetWindowLong(lsl.statusLabel.Handle(), win.GWL_EXSTYLE, win.GetWindowLong(lsl.statusLabel.Handle(), win.GWL_EXSTYLE)&^win.WS_EX_CLIENTEDGE)
	lsl.statusLabel.SetReadOnly(true)
	lsl.statusLabel.SetBackground(walk.NullBrush())
	lsl.statusLabel.FocusedChanged().Attach(func() {
		lsl.statusLabel.SetTextSelection(0, 0)
	})

	lsl.statusLabel.Accessibility().SetRole(walk.AccRoleStatictext)

	disposables.Spare()

	return lsl, nil
}

type keyActionsButtonLine struct {
	composite               *walk.Composite
	copyLocationToClipboard *walk.PushButton
	copyKeyToClipboard      *walk.PushButton
}

func (tal *keyActionsButtonLine) widgets() (walk.Widget, walk.Widget) {
	return nil, tal.composite
}

func (tal *keyActionsButtonLine) Dispose() {
	tal.composite.Dispose()
}

func newPublicKeyActionsLine(parent walk.Container) (*keyActionsButtonLine, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	tal := new(keyActionsButtonLine)

	if tal.composite, err = walk.NewComposite(parent); err != nil {
		return nil, err
	}
	disposables.Add(tal.composite)
	layout := walk.NewHBoxLayout()
	layout.SetMargins(walk.Margins{0, 0, 0, 6})
	tal.composite.SetLayout(layout)

	if tal.copyLocationToClipboard, err = walk.NewPushButton(tal.composite); err != nil {
		return nil, err
	}
	tal.copyLocationToClipboard.SetText("Copy Path")
	disposables.Add(tal.copyLocationToClipboard)

	if tal.copyKeyToClipboard, err = walk.NewPushButton(tal.composite); err != nil {
		return nil, err
	}
	tal.copyKeyToClipboard.SetText("Copy Key")
	disposables.Add(tal.copyKeyToClipboard)

	walk.NewHSpacer(tal.composite)

	disposables.Spare()

	return tal, nil
}

func newLabelTextLine(fieldName string, parent walk.Container) (*labelTextLine, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	lt := new(labelTextLine)

	if lt.label, err = walk.NewTextLabel(parent); err != nil {
		return nil, err
	}
	disposables.Add(lt.label)
	lt.label.SetText(fieldName)
	lt.label.SetTextAlignment(walk.AlignHFarVNear)
	lt.label.SetVisible(false)

	if lt.text, err = walk.NewTextEdit(parent); err != nil {
		return nil, err
	}
	disposables.Add(lt.text)
	win.SetWindowLong(lt.text.Handle(), win.GWL_EXSTYLE, win.GetWindowLong(lt.text.Handle(), win.GWL_EXSTYLE)&^win.WS_EX_CLIENTEDGE)
	lt.text.SetCompactHeight(true)
	lt.text.SetReadOnly(true)
	lt.text.SetBackground(walk.NullBrush())
	lt.text.SetVisible(false)
	lt.text.FocusedChanged().Attach(func() {
		lt.text.SetTextSelection(0, 0)
	})
	lt.text.Accessibility().SetRole(walk.AccRoleStatictext)

	disposables.Spare()

	return lt, nil
}

func createLabelTextLines(items []labelTextLineItem, parent walk.Container, disposables *walk.Disposables) ([]widgetsLine, error) {
	var err error
	var disps walk.Disposables
	defer disps.Treat()

	wls := make([]widgetsLine, len(items))
	for i, item := range items {
		if *item.ptr, err = newLabelTextLine(item.label, parent); err != nil {
			return nil, err
		}
		disps.Add(*item.ptr)
		if disposables != nil {
			disposables.Add(*item.ptr)
		}
		wls[i] = *item.ptr
	}

	disps.Spare()

	return wls, nil
}

func layoutInGrid(view widgetsLinesView, layout *walk.GridLayout) {
	for i, l := range view.widgetsLines() {
		w1, w2 := l.widgets()

		if w1 != nil {
			layout.SetRange(w1, walk.Rectangle{0, i, 1, 1})
		}
		if w2 != nil {
			layout.SetRange(w2, walk.Rectangle{2, i, 1, 1})
		}
	}
}

func newPaddedGroupGrid(parent walk.Container) (group *walk.GroupBox, err error) {
	group, err = walk.NewGroupBox(parent)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			group.Dispose()
		}
	}()
	layout := walk.NewGridLayout()
	layout.SetMargins(walk.Margins{10, 5, 10, 5})
	layout.SetSpacing(0)
	err = group.SetLayout(layout)
	if err != nil {
		return nil, err
	}
	spacer, err := walk.NewSpacerWithCfg(group, &walk.SpacerCfg{walk.GrowableHorz | walk.GreedyHorz, walk.Size{10, 0}, false})
	if err != nil {
		return nil, err
	}
	layout.SetRange(spacer, walk.Rectangle{1, 0, 1, 1})
	return group, nil
}
