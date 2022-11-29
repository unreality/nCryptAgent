package deviceevents

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/sys/windows"
	"reflect"
	"unsafe"
)

const (
	//dbt.h - https://github.com/tpn/winsdk-10/blob/master/Include/10.0.16299.0/um/Dbt.h
	DBT_DEVICEARRIVAL           = 0x8000
	DBT_DEVICEQUERYREMOVE       = 0x8001 // wants to remove, may fail
	DBT_DEVICEQUERYREMOVEFAILED = 0x8002 // removal aborted
	DBT_DEVICEREMOVEPENDING     = 0x8003 // about to remove, still avail.
	DBT_DEVICEREMOVECOMPLETE    = 0x8004 // device is gone
	DBT_DEVICETYPESPECIFIC      = 0x8005 // type specific event

	DEVICE_NOTIFY_WINDOW_HANDLE         = 0x0
	DEVICE_NOTIFY_ALL_INTERFACE_CLASSES = 0x00000004
	DBT_DEVTYP_DEVICEINTERFACE          = 0x00000005 // device interface class
)

var (
	user32                         = windows.MustLoadDLL("user32.dll")
	registerDeviceNotificationProc = user32.MustFindProc("RegisterDeviceNotificationW")

	SMARTCARD_DEVICE_CLASS = windows.GUID{
		0xDEEBE6AD,
		0x9E01,
		0x47E2,
		[8]byte{0xA3, 0xB2, 0xA6, 0x6A, 0xA2, 0xC0, 0x36, 0xC9},
	}
)

func RegisterDeviceNotification(hwnd windows.HWND) error {

	var notificationFilter struct {
		dwSize       uint32
		dwDeviceType uint32
		dwReserved   uint32
		classGuid    windows.GUID
		szName       uint16
	}
	notificationFilter.dwSize = uint32(unsafe.Sizeof(notificationFilter))
	notificationFilter.dwDeviceType = DBT_DEVTYP_DEVICEINTERFACE
	notificationFilter.dwReserved = 0
	//notificationFilter.classGuid = SMARTCARD_DEVICE_CLASS // seems to be ignored
	notificationFilter.szName = 0

	r1, _, err := registerDeviceNotificationProc.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&notificationFilter)),
		DEVICE_NOTIFY_WINDOW_HANDLE|DEVICE_NOTIFY_ALL_INTERFACE_CLASSES,
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func ReadDeviceInfo(devInfoPtr uintptr) (uint32, windows.GUID, string, error) {
	var devInfo struct {
		dbccSize       uint32
		dbccDeviceType uint32
		dbccReserved   uint32
		GUID           windows.GUID
	}

	var err error

	var devInfoBytes []byte
	slice := (*reflect.SliceHeader)(unsafe.Pointer(&devInfoBytes))
	slice.Data = devInfoPtr
	slice.Len = int(uint32(unsafe.Sizeof(devInfo)))
	slice.Cap = int(uint32(unsafe.Sizeof(devInfo)))

	reader := bytes.NewReader(devInfoBytes)
	// TODO: ARM might need to use different endianness
	err = binary.Read(reader, binary.LittleEndian, &devInfo.dbccSize)
	err = binary.Read(reader, binary.LittleEndian, &devInfo.dbccDeviceType)
	err = binary.Read(reader, binary.LittleEndian, &devInfo.dbccReserved)
	err = binary.Read(reader, binary.LittleEndian, &devInfo.GUID)

	if err != nil {
		return 0, windows.GUID{}, "", err
	}

	var devNameBytes []byte
	devNameSlice := (*reflect.SliceHeader)(unsafe.Pointer(&devNameBytes))
	devNameSlice.Data = devInfoPtr + unsafe.Sizeof(devInfo)
	devNameSlice.Len = int(devInfo.dbccSize - uint32(unsafe.Sizeof(devInfo)))
	devNameSlice.Cap = int(devInfo.dbccSize - uint32(unsafe.Sizeof(devInfo)))

	return devInfo.dbccDeviceType, devInfo.GUID, string(devNameBytes), nil
}
