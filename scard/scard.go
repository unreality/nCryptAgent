package scard

import (
	"fmt"
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

var (
	winscard             = windows.MustLoadDLL("winscard.dll")
	procSCardListCards   = winscard.MustFindProc("SCardListCardsA")
	procSCardFreeMemory  = winscard.MustFindProc("SCardFreeMemory")
	procSCardListReaders = winscard.MustFindProc("SCardListReadersA")
)

// wide returns a pointer to a a uint16 representing the equivalent
// to a Windows LPCWSTR.
func wide(s string) *uint16 {
	w, _ := syscall.UTF16PtrFromString(s)
	return w
}

func SCardListCards() ([]byte, error) {
	var size uint32

	// Get the size of the data to be returned
	r, _, err := procSCardListCards.Call(
		0, //[in] hContext
		0, // [in, optional] pbAtr
		0, // [in] rgquidInterfaces
		0, // [in] cguidInterfaceCount
		0, // [out] mszCards
		uintptr(unsafe.Pointer(&size)),
	)
	if r != 0 {
		return nil, fmt.Errorf("SCardListCards returned %v during size check: %w", uint32(r), err)
	}

	// Place the data in buf now that we know the size required
	buf := make([]byte, size)
	r, _, err = procSCardListCards.Call(
		0, //[in] hContext
		0, // [in, optional] pbAtr
		0, // [in] rgquidInterfaces
		0, // [in] cguidInterfaceCount
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r != 0 {
		return nil, fmt.Errorf("SCardListCards returned %v during convert: %w", uint32(r), err)
	}
	return buf, nil
}

func SCardListReaders() ([]string, error) {
	var size uint32

	// Get the size of the data to be returned
	r, _, err := procSCardListReaders.Call(
		0,                              //[in] hContext
		0,                              // [in, optional] mszGroups
		0,                              // [out] mszReaders
		uintptr(unsafe.Pointer(&size)), // [in, out] pcchReaders
	)
	if r != 0 {
		return nil, fmt.Errorf("SCardListCards returned %v during size check: %w", uint32(r), err)
	}

	// Place the data in buf now that we know the size required
	buf := make([]byte, size)
	r, _, err = procSCardListReaders.Call(
		0,                                //[in] hContext
		0,                                // [in, optional] mszGroups
		uintptr(unsafe.Pointer(&buf[0])), // [out] mszReaders
		uintptr(unsafe.Pointer(&size)),   // [in, out] pcchReaders
	)
	if r != 0 {
		return nil, fmt.Errorf("SCardListCards returned %v during convert: %w", uint32(r), err)
	}

	var readerList []string
	var startIdx = 0
	for i, b := range buf {
		if b == 0x00 {
			readerList = append(readerList, string(buf[startIdx:i]))
			startIdx = i + 1
		}
	}

	return readerList, nil
}

func SCardFreeMemory(ptr uintptr) error {
	_, _, err := procSCardFreeMemory.Call(
		0,
		ptr,
	)

	if err != syscall.Errno(0) {
		fmt.Printf("err is %v", err)
		return err
	}

	return err
}
