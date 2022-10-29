package ui

import (
	"fmt"
	"github.com/lxn/walk"
)

type GlobalConfView struct {
	*walk.GroupBox

	PinTimeoutEdit *walk.LineEdit
}

func NewGlobalConfView(parent walk.Container) (*GlobalConfView, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	gcv := new(GlobalConfView)

	if gcv.GroupBox, err = newPaddedGroupGrid(parent); err != nil {
		return nil, err
	}
	disposables.Add(gcv)

	gcv.SetTitle("Global Config:")

	layout := gcv.Layout().(*walk.GridLayout)
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	//Setup the name
	pinTimeoutLabel, err := walk.NewTextLabel(gcv)
	if err != nil {
		return nil, err
	}
	layout.SetRange(pinTimeoutLabel, walk.Rectangle{0, 0, 1, 1})
	pinTimeoutLabel.SetTextAlignment(walk.AlignHFarVCenter)
	pinTimeoutLabel.SetText(fmt.Sprintf("&PIN Cache Duration:"))
	pinTimeoutLabel.SetToolTipText("Amount of time in seconds the PIN is cached for. Set a duration in seconds, 0 for no cache, or -1 for indefinite caching.")

	if gcv.PinTimeoutEdit, err = walk.NewLineEdit(gcv); err != nil {
		return nil, err
	}
	layout.SetRange(gcv.PinTimeoutEdit, walk.Rectangle{1, 0, 1, 1})
	gcv.PinTimeoutEdit.SetText("0")

	if err := walk.InitWrapperWindow(gcv); err != nil {
		return nil, err
	}
	gcv.SetDoubleBuffering(true)

	disposables.Spare()

	return gcv, nil
}

type PageantConfView struct {
	*walk.GroupBox

	ListenerEnabled *walk.CheckBox
}

func NewPageantConfView(parent walk.Container) (*PageantConfView, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	cv := new(PageantConfView)

	if cv.GroupBox, err = newPaddedGroupGrid(parent); err != nil {
		return nil, err
	}
	disposables.Add(cv)

	cv.SetTitle("Pageant/PuTTY:")

	layout := cv.Layout().(*walk.GridLayout)
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	//Setup the name
	enabledLabel, err := walk.NewTextLabel(cv)
	if err != nil {
		return nil, err
	}
	layout.SetRange(enabledLabel, walk.Rectangle{0, 0, 1, 1})
	enabledLabel.SetTextAlignment(walk.AlignHNearVCenter)
	enabledLabel.SetText(fmt.Sprintf("&Enabled:"))

	if cv.ListenerEnabled, err = walk.NewCheckBox(cv); err != nil {
		return nil, err
	}
	layout.SetRange(cv.ListenerEnabled, walk.Rectangle{1, 0, 1, 1})
	cv.ListenerEnabled.SetChecked(true)

	if err := walk.InitWrapperWindow(cv); err != nil {
		return nil, err
	}
	cv.SetDoubleBuffering(true)

	disposables.Spare()

	return cv, nil
}

type NamedPipeConfView struct {
	*walk.GroupBox

	ListenerEnabled *walk.CheckBox
}

func NewNamedPipeConfView(parent walk.Container) (*NamedPipeConfView, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	cv := new(NamedPipeConfView)

	if cv.GroupBox, err = newPaddedGroupGrid(parent); err != nil {
		return nil, err
	}
	disposables.Add(cv)

	cv.SetTitle("Named Pipe (OpenSSH for Windows):")

	layout := cv.Layout().(*walk.GridLayout)
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	//Setup the name
	enabledLabel, err := walk.NewTextLabel(cv)
	if err != nil {
		return nil, err
	}
	layout.SetRange(enabledLabel, walk.Rectangle{0, 0, 1, 1})
	enabledLabel.SetTextAlignment(walk.AlignHNearVCenter)
	enabledLabel.SetText(fmt.Sprintf("&Enabled:"))

	if cv.ListenerEnabled, err = walk.NewCheckBox(cv); err != nil {
		return nil, err
	}
	layout.SetRange(cv.ListenerEnabled, walk.Rectangle{1, 0, 1, 1})
	cv.ListenerEnabled.SetChecked(true)

	if err := walk.InitWrapperWindow(cv); err != nil {
		return nil, err
	}
	cv.SetDoubleBuffering(true)

	disposables.Spare()

	return cv, nil
}

type VSockConfView struct {
	*walk.GroupBox

	ListenerEnabled *walk.CheckBox
}

func NewVSockConfView(parent walk.Container) (*VSockConfView, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	cv := new(VSockConfView)

	if cv.GroupBox, err = newPaddedGroupGrid(parent); err != nil {
		return nil, err
	}
	disposables.Add(cv)

	cv.SetTitle("WSL2 (Hyper-V Socket):")

	layout := cv.Layout().(*walk.GridLayout)
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	//Setup the name
	enabledLabel, err := walk.NewTextLabel(cv)
	if err != nil {
		return nil, err
	}
	layout.SetRange(enabledLabel, walk.Rectangle{0, 0, 1, 1})
	enabledLabel.SetTextAlignment(walk.AlignHNearVCenter)
	enabledLabel.SetText(fmt.Sprintf("&Enabled:"))

	if cv.ListenerEnabled, err = walk.NewCheckBox(cv); err != nil {
		return nil, err
	}
	layout.SetRange(cv.ListenerEnabled, walk.Rectangle{1, 0, 1, 1})
	cv.ListenerEnabled.SetChecked(true)

	if err := walk.InitWrapperWindow(cv); err != nil {
		return nil, err
	}
	cv.SetDoubleBuffering(true)

	disposables.Spare()

	return cv, nil
}

type CygwinConfView struct {
	*walk.GroupBox

	ListenerEnabled *walk.CheckBox
	SocketLocation  *walk.TextEdit
}

func NewCygwinConfView(parent walk.Container) (*CygwinConfView, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	cv := new(CygwinConfView)

	if cv.GroupBox, err = newPaddedGroupGrid(parent); err != nil {
		return nil, err
	}
	disposables.Add(cv)

	cv.SetTitle("Cygwin (GIT for Windows/MSYS/mingw):")

	layout := cv.Layout().(*walk.GridLayout)
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	//Setup the name
	enabledLabel, err := walk.NewTextLabel(cv)
	if err != nil {
		return nil, err
	}
	layout.SetRange(enabledLabel, walk.Rectangle{0, 0, 1, 1})
	enabledLabel.SetTextAlignment(walk.AlignHNearVCenter)
	enabledLabel.SetText(fmt.Sprintf("&Enabled:"))

	if cv.ListenerEnabled, err = walk.NewCheckBox(cv); err != nil {
		return nil, err
	}
	layout.SetRange(cv.ListenerEnabled, walk.Rectangle{1, 0, 1, 1})
	cv.ListenerEnabled.SetAlignment(walk.AlignHNearVCenter)
	cv.ListenerEnabled.SetChecked(true)

	// Socket Location
	socketLocationLabel, err := walk.NewTextLabel(cv)
	if err != nil {
		return nil, err
	}
	layout.SetRange(socketLocationLabel, walk.Rectangle{0, 1, 1, 1})
	socketLocationLabel.SetTextAlignment(walk.AlignHNearVCenter)
	socketLocationLabel.SetText(fmt.Sprintf("&Socket Location:"))

	if cv.SocketLocation, err = walk.NewTextEdit(cv); err != nil {
		return nil, err
	}
	layout.SetRange(cv.SocketLocation, walk.Rectangle{1, 1, 1, 1})
	cv.SocketLocation.SetAlignment(walk.AlignHNearVCenter)
	cv.SocketLocation.SetText("")
	cv.SocketLocation.SetReadOnly(true)

	if err := walk.InitWrapperWindow(cv); err != nil {
		return nil, err
	}
	cv.SetDoubleBuffering(true)

	disposables.Spare()

	return cv, nil
}
