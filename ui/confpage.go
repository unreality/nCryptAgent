package ui

import (
	"fmt"
	"github.com/lxn/walk"
	"ncryptagent/ncrypt"
	"ncryptagent/ncrypt/listeners"
	"strconv"
)

type ConfPageView struct {
	*walk.ScrollView
	globalConfView    *GlobalConfView
	pageantConfView   *PageantConfView
	cygwinConfView    *CygwinConfView
	vsockConfView     *VSockConfView
	namedPipeConfView *NamedPipeConfView
	saveButton        *walk.PushButton
}

func NewConfPageView(parent walk.Container) (*ConfPageView, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	cpv := new(ConfPageView)
	if cpv.ScrollView, err = walk.NewScrollView(parent); err != nil {
		return nil, err
	}
	disposables.Add(cpv)

	vlayout := walk.NewVBoxLayout()
	vlayout.SetMargins(walk.Margins{5, 0, 5, 0})
	cpv.SetLayout(vlayout)

	if cpv.globalConfView, err = NewGlobalConfView(cpv); err != nil {
		return nil, err
	}

	if cpv.pageantConfView, err = NewPageantConfView(cpv); err != nil {
		return nil, err
	}

	if cpv.namedPipeConfView, err = NewNamedPipeConfView(cpv); err != nil {
		return nil, err
	}

	if cpv.vsockConfView, err = NewVSockConfView(cpv); err != nil {
		return nil, err
	}

	if cpv.cygwinConfView, err = NewCygwinConfView(cpv); err != nil {
		return nil, err
	}

	//walk.NewHSpacer(cpv)
	if cpv.saveButton, err = walk.NewPushButton(cpv); err != nil {
		return nil, err
	}
	cpv.saveButton.SetText(fmt.Sprintf("&Save"))
	//dlg.saveButton.Clicked().Attach(dlg.onSaveButtonClicked)

	disposables.Spare()

	return cpv, nil
}

type ConfPage struct {
	*walk.TabPage
	//keyView       *KeyView
	//fillerButton  *walk.PushButton
	//fillerHandler func()
	//
	//fillerContainer     *walk.Composite
	//currentKeyContainer *walk.Composite
	keyManager   *ncrypt.KeyManager
	confPageView *ConfPageView

	globalConfView    *GlobalConfView
	pageantConfView   *PageantConfView
	cygwinConfView    *CygwinConfView
	vsockConfView     *VSockConfView
	namedPipeConfView *NamedPipeConfView
}

func NewConfPage(keyManager *ncrypt.KeyManager) (*ConfPage, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	cp := new(ConfPage)

	cp.keyManager = keyManager

	if cp.TabPage, err = walk.NewTabPage(); err != nil {
		return nil, err
	}

	disposables.Add(cp)

	cp.SetTitle(fmt.Sprintf("Config"))
	vlayout := walk.NewVBoxLayout()
	vlayout.SetMargins(walk.Margins{5, 0, 5, 0})
	cp.SetLayout(vlayout)

	if cp.confPageView, err = NewConfPageView(cp); err != nil {
		return nil, err
	}

	cp.confPageView.saveButton.Clicked().Attach(cp.onSaveConfigClicked)
	cp.confPageView.VisibleChanged().Attach(cp.onTabSelected)

	disposables.Spare()

	return cp, nil
}

func (cp *ConfPage) onSaveConfigClicked() {
	intVal, err := strconv.Atoi(cp.confPageView.globalConfView.PinTimeoutEdit.Text())
	if err != nil {
		showError(fmt.Errorf("Invalid Pin Cache duration %s", cp.confPageView.globalConfView.PinTimeoutEdit.Text()), cp.Form())
	} else {
		cp.keyManager.SetPinTimeout(intVal)
	}

	cp.keyManager.EnableListener(listeners.TYPE_PAGEANT, cp.confPageView.pageantConfView.ListenerEnabled.Checked())
	cp.keyManager.EnableListener(listeners.TYPE_NAMED_PIPE, cp.confPageView.namedPipeConfView.ListenerEnabled.Checked())
	cp.keyManager.EnableListener(listeners.TYPE_VSOCK, cp.confPageView.vsockConfView.ListenerEnabled.Checked())
	cp.keyManager.EnableListener(listeners.TYPE_CYGWIN, cp.confPageView.cygwinConfView.ListenerEnabled.Checked())

	cp.keyManager.SaveConfig()
}

func (cp *ConfPage) onTabSelected() {
	if cp.confPageView.Visible() {
		cp.confPageView.cygwinConfView.SocketLocation.SetText(cp.keyManager.CygwinSocketLocation())
		cp.confPageView.pageantConfView.ListenerEnabled.SetChecked(cp.keyManager.GetListenerEnabled(listeners.TYPE_PAGEANT))
		cp.confPageView.namedPipeConfView.ListenerEnabled.SetChecked(cp.keyManager.GetListenerEnabled(listeners.TYPE_NAMED_PIPE))
		cp.confPageView.vsockConfView.ListenerEnabled.SetChecked(cp.keyManager.GetListenerEnabled(listeners.TYPE_VSOCK))
		cp.confPageView.cygwinConfView.ListenerEnabled.SetChecked(cp.keyManager.GetListenerEnabled(listeners.TYPE_CYGWIN))
		cp.confPageView.globalConfView.PinTimeoutEdit.SetText(strconv.Itoa(cp.keyManager.GetPinTimeout()))
	}
}
