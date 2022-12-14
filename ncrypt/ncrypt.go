package ncrypt

import (
	"bytes"
	"crypto"
	"crypto/elliptic"
	"encoding/binary"
	"fmt"
	"golang.org/x/sys/windows"
	"log"
	"unsafe"
)

const (

	// Key storage properties
	NCRYPT_ALGORITHM_GROUP_PROPERTY = "Algorithm Group"
	NCRYPT_LENGTH_PROPERTY          = "Length"
	NCRYPT_KEY_TYPE_PROPERTY        = "Key Type"
	NCRYPT_UNIQUE_NAME_PROPERTY     = "Unique Name"
	NCRYPT_ECC_CURVE_NAME_PROPERTY  = "ECCCurveName"
	NCRYPT_IMPL_TYPE_PROPERTY       = "Impl Type"
	NCRYPT_PROV_HANDLE              = "Provider Handle"
	NCRYPT_PIN_PROPERTY             = "SmartCardPin"
	NCRYPT_SECURE_PIN_PROPERTY      = "SmartCardSecurePin"
	NCRYPT_READER_PROPERTY          = "SmartCardReader"
	NCRYPT_ALGORITHM_PROPERTY       = "Algorithm Name"
	NCRYPT_WINDOW_HANDLE_PROPERTY   = "HWND Handle"
	NCRYPT_PCP_USAGE_AUTH_PROPERTY  = "PCP_USAGEAUTH"
	// Key Storage Flags
	NCRYPT_MACHINE_KEY_FLAG = 0x00000001
	NCRYPT_SILENT_FLAG      = 0x40

	// Errors
	NTE_NOT_SUPPORTED         = uint32(0x80090029)
	NTE_NO_MEMORY             = uint32(0x8009000E)
	NTE_INVALID_PARAMETER     = uint32(0x80090027)
	NTE_INVALID_HANDLE        = uint32(0x80090026)
	NTE_BAD_FLAGS             = uint32(0x80090009)
	NTE_NO_MORE_ITEMS         = uint32(0x8009002A)
	NTE_BAD_KEYSET            = uint32(0x80090016)
	SCARD_W_CANCELLED_BY_USER = uint32(0x8010006E)

	// wincrypt.h constants
	acquireCached           = 0x1                                             // CRYPT_ACQUIRE_CACHE_FLAG
	acquireSilent           = 0x40                                            // CRYPT_ACQUIRE_SILENT_FLAG
	encodingX509ASN         = 1                                               // X509_ASN_ENCODING
	encodingPKCS7           = 65536                                           // PKCS_7_ASN_ENCODING
	certStoreProvSystem     = 10                                              // CERT_STORE_PROV_SYSTEM
	certStoreOpenExisting   = 0x00004000                                      // CERT_STORE_OPEN_EXISTING_FLAG
	certStoreCurrentUser    = uint32(certStoreCurrentUserID << compareShift)  // CERT_SYSTEM_STORE_CURRENT_USER
	certStoreLocalMachine   = uint32(certStoreLocalMachineID << compareShift) // CERT_SYSTEM_STORE_LOCAL_MACHINE
	certStoreCurrentUserID  = 1                                               // CERT_SYSTEM_STORE_CURRENT_USER_ID
	certStoreLocalMachineID = 2                                               // CERT_SYSTEM_STORE_LOCAL_MACHINE_ID
	infoIssuerFlag          = 4                                               // CERT_INFO_ISSUER_FLAG
	compareNameStrW         = 8                                               // CERT_COMPARE_NAME_STR_A
	compareShift            = 16                                              // CERT_COMPARE_SHIFT
	compareSHA1Hash         = 1                                               // CERT_COMPARE_SHA1_HASH
	compareCertID           = 16                                              // CERT_COMPARE_CERT_ID
	findIssuerStr           = compareNameStrW<<compareShift | infoIssuerFlag  // CERT_FIND_ISSUER_STR_W
	findHash                = compareSHA1Hash << compareShift                 // CERT_FIND_HASH
	findCertID              = compareCertID << compareShift                   // CERT_FIND_CERT_ID

	signatureKeyUsage = 0x80       // CERT_DIGITAL_SIGNATURE_KEY_USAGE
	ncryptKeySpec     = 0xFFFFFFFF // CERT_NCRYPT_KEY_SPEC

	BCRYPT_RSAPUBLIC_BLOB = "RSAPUBLICBLOB"
	BCRYPT_ECCPUBLIC_BLOB = "ECCPUBLICBLOB"

	// winerror.h constants
	CRYPT_E_NOT_FOUND                    = uint32(0x80092004)
	CRYPT_ACQUIRE_ALLOW_NCRYPT_KEY_FLAG  = uint32(0x00010000)
	CRYPT_ACQUIRE_PREFER_NCRYPT_KEY_FLAG = uint32(0x00020000)
	CRYPT_ACQUIRE_ONLY_NCRYPT_KEY_FLAG   = uint32(0x00040000)

	CERT_ID_ISSUER_SERIAL_NUMBER = uint32(1)
	CERT_ID_KEY_IDENTIFIER       = uint32(2)
	CERT_ID_SHA1_HASH            = uint32(3)

	CERT_NAME_STR_COMMA_FLAG = uint32(0x04000000)
	CERT_SIMPLE_NAME_STR     = uint32(1)
	CERT_X500_NAME_STR       = uint32(3)

	AT_KEYEXCHANGE = uint32(1)
	AT_SIGNATURE   = uint32(2)

	// Legacy CryptoAPI flags
	bCryptPadPKCS1 = uint32(2)

	// Magic numbers for public key blobs.
	RSA1Magic = 0x31415352 // "RSA1" BCRYPT_RSAPUBLIC_MAGIC
	ECS1Magic = 0x31534345 // "ECS1" BCRYPT_ECDSA_PUBLIC_P256_MAGIC
	ECS3Magic = 0x33534345 // "ECS3" BCRYPT_ECDSA_PUBLIC_P384_MAGIC
	ECS5Magic = 0x35534345 // "ECS5" BCRYPT_ECDSA_PUBLIC_P521_MAGIC

	ProviderMSSC       = "Microsoft Smart Card Key Storage Provider"
	ProviderMSPlatform = "Microsoft Platform Crypto Provider"

	ALG_RSA        = "RSA"
	ALG_ECDSA_P256 = "ECDSA_P256"
	ALG_ECDSA_P384 = "ECDSA_P384"
	ALG_ECDSA_P521 = "ECDSA_P521"
)

var (
	CurveNames = map[string]elliptic.Curve{
		ALG_ECDSA_P256: elliptic.P256(),
		ALG_ECDSA_P384: elliptic.P384(),
		ALG_ECDSA_P521: elliptic.P521(),
		"nistP256":     elliptic.P256(), // BCRYPT_ECC_CURVE_NISTP256
		"nistP384":     elliptic.P384(), // BCRYPT_ECC_CURVE_NISTP384
		"nistP521":     elliptic.P521(), // BCRYPT_ECC_CURVE_NISTP521
	}

	CurveMagicMap = map[string]uint32{
		"P-256": ECS1Magic,
		"P-384": ECS3Magic,
		"P-521": ECS5Magic,
	}

	// algIDs maps crypto.Hash values to bcrypt.h constants.
	HashAlgorithms = map[crypto.Hash]string{
		crypto.SHA1:   "SHA1",   // BCRYPT_SHA1_ALGORITHM
		crypto.SHA256: "SHA256", // BCRYPT_SHA256_ALGORITHM
		crypto.SHA384: "SHA384", // BCRYPT_SHA384_ALGORITHM
		crypto.SHA512: "SHA512", // BCRYPT_SHA512_ALGORITHM
	}

	AVAILABLE_ALGORITHMS = []string{ALG_ECDSA_P256, ALG_ECDSA_P384, ALG_ECDSA_P521, ALG_RSA}

	crypt32 = windows.MustLoadDLL("crypt32.dll")
	nCrypt  = windows.MustLoadDLL("ncrypt.dll")

	procCertFindCertificateInStore      = crypt32.MustFindProc("CertFindCertificateInStore")
	procCryptFindCertificateKeyProvInfo = crypt32.MustFindProc("CryptFindCertificateKeyProvInfo")
	procCertStrToName                   = crypt32.MustFindProc("CertStrToNameW")
	procNCryptCreatePersistedKey        = nCrypt.MustFindProc("NCryptCreatePersistedKey")
	procNCryptDeleteKey                 = nCrypt.MustFindProc("NCryptDeleteKey")
	procNCryptEnumKeys                  = nCrypt.MustFindProc("NCryptEnumKeys")
	procNCryptExportKey                 = nCrypt.MustFindProc("NCryptExportKey")
	procNCryptFinalizeKey               = nCrypt.MustFindProc("NCryptFinalizeKey")
	procNCryptFreeBuffer                = nCrypt.MustFindProc("NCryptFreeBuffer")
	procNCryptFreeObject                = nCrypt.MustFindProc("NCryptFreeObject")
	procNCryptOpenKey                   = nCrypt.MustFindProc("NCryptOpenKey")
	procNCryptOpenStorageProvider       = nCrypt.MustFindProc("NCryptOpenStorageProvider")
	procNCryptGetProperty               = nCrypt.MustFindProc("NCryptGetProperty")
	procNCryptSetProperty               = nCrypt.MustFindProc("NCryptSetProperty")
	procNCryptSignHash                  = nCrypt.MustFindProc("NCryptSignHash")
)

type BCRYPT_PKCS1_PADDING_INFO struct {
	pszAlgID *uint16
}

//CRYPTOAPI_BLOB -- https://learn.microsoft.com/en-us/previous-versions/windows/desktop/legacy/aa381414(v=vs.85)
type CRYPTOAPI_BLOB struct {
	len  uint32
	data uintptr
}

// CERT_ISSUER_SERIAL_NUMBER -- https://learn.microsoft.com/en-us/windows/win32/api/wincrypt/ns-wincrypt-cert_issuer_serial_number
type CERT_ISSUER_SERIAL_NUMBER struct {
	Issuer       CRYPTOAPI_BLOB
	SerialNumber CRYPTOAPI_BLOB
}

// CERT_ID - https://learn.microsoft.com/en-us/windows/win32/api/wincrypt/ns-wincrypt-cert_id
// TODO: might be able to merge these two types into one that uses interface{} instead
type CERT_ID_KEYIDORHASH struct {
	idChoice    uint32
	KeyIDOrHash CRYPTOAPI_BLOB
}

type CERT_ID_SERIAL struct {
	idChoice uint32
	Serial   CERT_ISSUER_SERIAL_NUMBER
}

type NCryptKeyName struct {
	Name          *uint16
	Algid         *uint16
	LegacyKeySpec uint32
	Flags         uint32
}

func errNoToStr(e uint32) string {
	switch e {
	case NTE_INVALID_PARAMETER:
		return "NTE_INVALID_PARAMETER"
	case NTE_BAD_FLAGS:
		return "NTE_BAD_FLAGS"
	case NTE_BAD_KEYSET:
		return "NTE_BAD_KEYSET"
	case NTE_NO_MORE_ITEMS:
		return "NTE_NO_MORE_ITEMS"
	case NTE_NOT_SUPPORTED:
		return "NTE_NOT_SUPPORTED"
	case SCARD_W_CANCELLED_BY_USER:
		return "User cancelled smartcard action"
	default:
		return fmt.Sprintf("0x%X", e)
	}
}

// wide returns a pointer to a uint16 representing the equivalent
// to a Windows LPCWSTR.
func wide(s string) *uint16 {
	w, _ := windows.UTF16PtrFromString(s)
	return w
}

func NCryptOpenStorageProvider(provider string) (uintptr, error) {
	var hProv uintptr
	// Open the provider, the last parameter is not used
	r, _, err := procNCryptOpenStorageProvider.Call(
		uintptr(unsafe.Pointer(&hProv)),
		uintptr(unsafe.Pointer(wide(provider))),
		0)
	if r == 0 {
		return hProv, nil
	}
	return hProv, fmt.Errorf("NCryptOpenStorageProvider returned %v: %v", errNoToStr(uint32(r)), err)
}

func NCryptFreeObject(h uintptr) error {
	r, _, err := procNCryptFreeObject.Call(h)
	if r == 0 {
		return nil
	}
	return fmt.Errorf("NCryptFreeObject returned %v: %v", errNoToStr(uint32(r)), err)
}

func NCryptFreeBuffer(pvInput uintptr) (err error) {
	_, _, e1 := procNCryptFreeBuffer.Call(
		pvInput,
		0,
		0,
	)

	if e1 != windows.Errno(0) {
		log.Printf("err is %v", e1)
		return e1
	}

	return nil
}

func NCryptDeleteKey(keyHandle uintptr, flags uint32) error {
	r, _, _ := procNCryptDeleteKey.Call(
		keyHandle,
		uintptr(flags))

	if r != 0 {
		return fmt.Errorf("NCryptDeleteKey returned %v", errNoToStr(uint32(r)))
	}

	return nil
}

func NCryptCreatePersistedKey(provisionerHandle uintptr, containerName string, algorithmName string, legacyKeySpec uint32, flags uint32) (uintptr, error) {
	var kh uintptr
	var kn uintptr = 0

	if containerName != "" {
		kn = uintptr(unsafe.Pointer(wide(containerName)))
	}

	r, _, err := procNCryptCreatePersistedKey.Call(
		provisionerHandle,
		uintptr(unsafe.Pointer(&kh)),
		uintptr(unsafe.Pointer(wide(algorithmName))),
		kn,
		uintptr(legacyKeySpec),
		uintptr(flags))
	if r != 0 {
		return 0, fmt.Errorf("NCryptCreatePersistedKey returned %v: %v", errNoToStr(uint32(r)), err)
	}

	return kh, nil
}

func NCryptOpenKey(provisionerHandle uintptr, containerName string, legacyKeySpec uint32, flags uint32) (uintptr, error) {
	var kh uintptr
	r, _, err := procNCryptOpenKey.Call(
		provisionerHandle,
		uintptr(unsafe.Pointer(&kh)),
		uintptr(unsafe.Pointer(wide(containerName))),
		uintptr(legacyKeySpec),
		uintptr(flags))
	if r != 0 {
		return 0, fmt.Errorf("NCryptOpenKey for container %q returned %v: %v", containerName, errNoToStr(uint32(r)), err)
	}

	return kh, nil
}

func NCryptFinalizeKey(keyHandle uintptr, flags uint32) error {
	r, _, err := procNCryptFinalizeKey.Call(keyHandle, uintptr(flags))
	if r != 0 {
		return fmt.Errorf("NCryptFinalizeKey returned %v: %v", errNoToStr(uint32(r)), err)
	}

	return nil
}

func NCryptSetProperty(keyHandle uintptr, propertyName string, propertyValue interface{}, flags uint32) error {

	if intVal, isInt := propertyValue.(uint32); isInt {
		r, _, err := procNCryptSetProperty.Call(
			keyHandle,
			uintptr(unsafe.Pointer(wide(propertyName))),
			uintptr(unsafe.Pointer(&intVal)),
			unsafe.Sizeof(intVal),
			uintptr(flags))
		if r != 0 {
			return fmt.Errorf("NCryptSetProperty \"%v\" returned %v: %v", propertyName, errNoToStr(uint32(r)), err)
		}

		return nil
	}

	if ptrVal, isPtr := propertyValue.(uintptr); isPtr {
		r, _, err := procNCryptSetProperty.Call(
			keyHandle,
			uintptr(unsafe.Pointer(wide(propertyName))),
			uintptr(unsafe.Pointer(&ptrVal)),
			unsafe.Sizeof(ptrVal),
			uintptr(flags))
		if r != 0 {
			return fmt.Errorf("NCryptSetProperty \"%v\" returned %v: %v", propertyName, errNoToStr(uint32(r)), err)
		}

		return nil
	}

	if byteVal, isBytes := propertyValue.([]byte); isBytes {
		r, _, err := procNCryptSetProperty.Call(
			keyHandle,
			uintptr(unsafe.Pointer(wide(propertyName))),
			uintptr(unsafe.Pointer(&byteVal[0])),
			uintptr(len(byteVal)),
			uintptr(flags))
		if r != 0 {
			return fmt.Errorf("NCryptSetProperty \"%v\" returned %v: %v", propertyName, errNoToStr(uint32(r)), err)
		}

		return nil
	}

	if strVal, isStr := propertyValue.(string); isStr {
		l := len(strVal)

		r, _, err := procNCryptSetProperty.Call(
			keyHandle,
			uintptr(unsafe.Pointer(wide(propertyName))),
			uintptr(unsafe.Pointer(wide(strVal))),
			uintptr(l),
			uintptr(flags))
		if r != 0 {
			return fmt.Errorf("NCryptSetProperty \"%v\" returned %X: %v", propertyName, errNoToStr(uint32(r)), err)
		}

		return nil
	}

	return fmt.Errorf("NCryptSetProperty %v invalid value %T", propertyName, propertyValue)
}

func NCryptSignHash(kh uintptr, digest []byte, hashID string) ([]byte, error) {
	var size uint32
	var padInfoPtr uintptr
	var flags uint32
	if hashID != "" {
		padInfo := BCRYPT_PKCS1_PADDING_INFO{pszAlgID: wide(hashID)}
		padInfoPtr = uintptr(unsafe.Pointer(&padInfo))
		flags = bCryptPadPKCS1
	} else {
		padInfoPtr = 0
		flags = 0
	}

	// Obtain the size of the signature
	r, _, err := procNCryptSignHash.Call(
		kh,
		padInfoPtr,
		uintptr(unsafe.Pointer(&digest[0])),
		uintptr(len(digest)),
		0,
		0,
		uintptr(unsafe.Pointer(&size)),
		uintptr(flags))
	if r != 0 {
		return nil, fmt.Errorf("NCryptSignHash returned %v during size check: %v", errNoToStr(uint32(r)), err)
	}

	// Obtain the signature data
	buf := make([]byte, size)
	r, _, err = procNCryptSignHash.Call(
		kh,
		padInfoPtr,
		uintptr(unsafe.Pointer(&digest[0])),
		uintptr(len(digest)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(size),
		uintptr(unsafe.Pointer(&size)),
		uintptr(flags),
	)
	if r != 0 {
		return nil, fmt.Errorf("NCryptSignHash returned %v during signing: %v", errNoToStr(uint32(r)), err)
	}

	return buf[:size], nil
}

func getProperty(kh uintptr, property *uint16) ([]byte, error) {
	var strSize uint32
	r, _, err := procNCryptGetProperty.Call(
		kh,
		uintptr(unsafe.Pointer(property)),
		0,
		0,
		uintptr(unsafe.Pointer(&strSize)),
		0,
		0)
	if r != 0 {
		return nil, fmt.Errorf("NCryptGetProperty(%v) returned %v during size check: %v", property, errNoToStr(uint32(r)), err)
	}

	buf := make([]byte, strSize)
	r, _, err = procNCryptGetProperty.Call(
		kh,
		uintptr(unsafe.Pointer(property)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(strSize),
		uintptr(unsafe.Pointer(&strSize)),
		0,
		0)
	if r != 0 {
		return nil, fmt.Errorf("NCryptGetProperty %v returned %v during export: %v", property, errNoToStr(uint32(r)), err)
	}

	return buf, nil
}

func NCryptGetPropertyHandle(kh uintptr, property string) (uintptr, error) {
	buf, err := getProperty(kh, wide(property))
	if err != nil {
		return 0, err
	}
	if len(buf) < 1 {
		return 0, fmt.Errorf("empty result")
	}
	return **(**uintptr)(unsafe.Pointer(&buf)), nil
}

func NCryptGetPropertyInt(kh uintptr, property string) (int, error) {
	buf, err := getProperty(kh, wide(property))

	if err != nil {
		return 0, err
	}
	if len(buf) < 1 {
		return 0, fmt.Errorf("empty result")
	}

	//TODO maybe someone wants to run this on ARM one day..so this might break
	return int(binary.LittleEndian.Uint32(buf)), nil
}

func NCryptGetPropertyStr(kh uintptr, property string) (string, error) {
	buf, err := getProperty(kh, wide(property))
	if err != nil {
		return "", err
	}
	uc := bytes.ReplaceAll(buf, []byte{0x00}, []byte(""))
	return string(uc), nil
}

func NCryptEnumKeys(provider uintptr, scope string, enumState uintptr, flags uint32) (*NCryptKeyName, uintptr, error) {
	scopePtr := uintptr(0)
	keyName := uintptr(0)

	if scope != "" {
		scopePtr = uintptr(unsafe.Pointer(wide(scope)))
	}

	r, _, err := procNCryptEnumKeys.Call(
		provider,
		scopePtr,
		uintptr(unsafe.Pointer(&keyName)),
		uintptr(unsafe.Pointer(&enumState)),
		uintptr(flags),
	)
	switch uint32(r) {
	case NTE_BAD_FLAGS:
		return nil, enumState, fmt.Errorf("NTE_BAD_FLAGS")
	case NTE_INVALID_HANDLE:
		return nil, enumState, fmt.Errorf("NTE_INVALID_HANDLE")
	case NTE_NO_MEMORY:
		return nil, enumState, fmt.Errorf("NTE_NO_MEMORY")
	case NTE_NO_MORE_ITEMS:
		_ = NCryptFreeBuffer(enumState)
		return nil, uintptr(0), nil
	case NTE_INVALID_PARAMETER:
		return nil, enumState, fmt.Errorf("NTE_INVALID_PARAMETER")
	}

	if r != 0 {
		return nil, enumState, err
	}

	ppKeyName := (**NCryptKeyName)(unsafe.Pointer(&keyName))

	return *ppKeyName, enumState, nil
}

func NCryptExportKey(kh uintptr, blobType string) ([]byte, error) {
	var size uint32
	// When obtaining the size of a public key, most parameters are not required
	r, _, err := procNCryptExportKey.Call(
		kh,
		0,
		uintptr(unsafe.Pointer(wide(blobType))),
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&size)),
		0)
	if r != 0 {
		return nil, fmt.Errorf("NCryptExportKey returned %v during size check: %v", errNoToStr(uint32(r)), err)
	}

	// Place the exported key in buf now that we know the size required
	buf := make([]byte, size)
	r, _, err = procNCryptExportKey.Call(
		kh,
		0,
		uintptr(unsafe.Pointer(wide(blobType))),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(size),
		uintptr(unsafe.Pointer(&size)),
		0)
	if r != 0 {
		return nil, fmt.Errorf("NCryptExportKey returned %v during export: %v", errNoToStr(uint32(r)), err)
	}
	return buf, nil
}

func FindCertificateInStore(store windows.Handle, enc, findFlags, findType uint32, para uintptr, prev *windows.CertContext) (*windows.CertContext, error) {
	h, _, err := procCertFindCertificateInStore.Call(
		uintptr(store),
		uintptr(enc),
		uintptr(findFlags),
		uintptr(findType),
		para,
		uintptr(unsafe.Pointer(prev)),
	)
	if h == 0 {
		// Actual error, or simply not found?
		if errno, ok := err.(windows.Errno); ok && uint32(errno) == CRYPT_E_NOT_FOUND {
			return nil, nil
		}
		return nil, err
	}
	return (*windows.CertContext)(unsafe.Pointer(h)), nil
}

func FreeCertContext(ctx *windows.CertContext) error {
	return windows.CertFreeCertificateContext(ctx)
}

func CryptFindCertificateKeyProvInfo(certContext *windows.CertContext) error {
	r, _, err := procCryptFindCertificateKeyProvInfo.Call(
		uintptr(unsafe.Pointer(certContext)),
		uintptr(CRYPT_ACQUIRE_PREFER_NCRYPT_KEY_FLAG),
		0,
	)

	if r == 0 {
		return fmt.Errorf("private key association failed: %v", err)
	}

	return nil
}

func CertStrToName(x500Str string) ([]byte, error) {
	var size uint32

	// Get the size of the data to be returned
	r, _, err := procCertStrToName.Call(
		uintptr(encodingX509ASN),
		uintptr(unsafe.Pointer(wide(x500Str))),
		uintptr(CERT_X500_NAME_STR|CERT_NAME_STR_COMMA_FLAG),
		0, // pvReserved
		0, // pbEncoded
		uintptr(unsafe.Pointer(&size)),
		0,
	)
	if r != 1 {
		return nil, fmt.Errorf("CertStrToName returned %v during size check: %w", errNoToStr(uint32(r)), err)
	}

	// Place the data in buf now that we know the size required
	buf := make([]byte, size)
	r, _, err = procCertStrToName.Call(
		uintptr(encodingX509ASN),
		uintptr(unsafe.Pointer(wide(x500Str))),
		uintptr(CERT_SIMPLE_NAME_STR),
		0, // pvReserved
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
	)
	if r != 1 {
		return nil, fmt.Errorf("CertStrToName returned %v during convert: %w", errNoToStr(uint32(r)), err)
	}
	return buf, nil
}

type NCryptKeyDescriptor struct {
	Container string
	Algorithm string
}

func ListKeysOnProvider(providerName string, cardReader string) ([]NCryptKeyDescriptor, error) {
	var err error

	var scope string
	if cardReader != "" {
		scope = fmt.Sprintf("\\\\.\\%s\\", cardReader)
	}

	prov, err := NCryptOpenStorageProvider(providerName)
	if err != nil {
		return nil, err
	}

	defer NCryptFreeObject(prov)

	var ret []NCryptKeyDescriptor
	enumState := uintptr(0)
	iterating := true
	for iterating {
		var key *NCryptKeyName
		key, enumState, err = NCryptEnumKeys(prov, scope, enumState, 0)

		if errno, ok := err.(windows.Errno); ok && uint32(errno) == 0 {
			return nil, nil
		}

		if err != nil {
			return nil, err
		}

		if key == nil {
			iterating = false
			continue
		}

		keyName := windows.UTF16PtrToString(key.Name)
		keyAlg := windows.UTF16PtrToString(key.Algid)

		ret = append(ret, NCryptKeyDescriptor{
			Container: keyName,
			Algorithm: keyAlg,
		})

		err = NCryptFreeBuffer(uintptr(unsafe.Pointer(key)))
		if err != nil {
			return nil, err
		}

		if enumState == 0 {
			iterating = false
		}
	}

	return ret, nil
}
