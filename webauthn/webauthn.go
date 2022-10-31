package webauthn

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	webauthn = syscall.MustLoadDLL("webauthn.dll")

	procWebAuthNGetApiVersionNumber = webauthn.MustFindProc("WebAuthNGetApiVersionNumber")
	//procWebAuthNIsUserVerifyingPlatformAuthenticatorAvailable = webauthn.MustFindProc("WebAuthNIsUserVerifyingPlatformAuthenticatorAvailable")
	procWebAuthNAuthenticatorMakeCredential = webauthn.MustFindProc("WebAuthNAuthenticatorMakeCredential")
	procWebAuthNAuthenticatorGetAssertion   = webauthn.MustFindProc("WebAuthNAuthenticatorGetAssertion")
	procWebAuthNFreeCredentialAttestation   = webauthn.MustFindProc("WebAuthNFreeCredentialAttestation")
	procWebAuthNFreeAssertion               = webauthn.MustFindProc("WebAuthNFreeAssertion")
	//procWebAuthNGetCancellationId                             = webauthn.MustFindProc("WebAuthNGetCancellationId")
	//procWebAuthNCancelCurrentOperation                        = webauthn.MustFindProc("WebAuthNCancelCurrentOperation")
	//procWebAuthNGetErrorName                                  = webauthn.MustFindProc("WebAuthNGetErrorName")
	//procWebAuthNGetW3CExceptionDOMError                       = webauthn.MustFindProc("WebAuthNGetW3CExceptionDOMError")
)

const (
	// Errors
	NTE_NOT_SUPPORTED         = uint32(0x80090029)
	NTE_INVALID_PARAMETER     = uint32(0x80090027)
	NTE_BAD_FLAGS             = uint32(0x80090009)
	NTE_NO_MORE_ITEMS         = uint32(0x8009002A)
	NTE_BAD_KEYSET            = uint32(0x80090016)
	SCARD_W_CANCELLED_BY_USER = uint32(0x8010006E)

	RP_ENTITY_INFORMATION_CURRENT_VERSION     = 1
	USER_ENTITY_INFORMATION_CURRENT_VERSION   = 1
	COSE_CREDENTIAL_PARAMETER_CURRENT_VERSION = 1
	CLIENT_DATA_CURRENT_VERSION               = 1
	ASSERTION_CURRENT_VERSION                 = 1
	CREDENTIAL_CURRENT_VERSION                = 1

	AUTHENTICATOR_MAKE_CREDENTIAL_OPTIONS_CURRENT_VERSION = 3
	AUTHENTICATOR_GET_ASSERTION_OPTIONS_CURRENT_VERSION   = 4

	CREDENTIAL_TYPE_PUBLIC_KEY = "public-key"

	COSE_KEY_PARAMETER_KTY = 1
	COSE_KEY_PARAMETER_ALG = 3

	//https://www.rfc-editor.org/rfc/rfc8152#section-13
	COSE_KEY_TYPE_OKP       = 1
	COSE_KEY_TYPE_EC2       = 2
	COSE_KEY_TYPE_SYMMETRIC = 4

	//https://www.rfc-editor.org/rfc/rfc8152#section-13.1
	COSE_KEY_CURVE_P256    = 1
	COSE_KEY_CURVE_P384    = 2
	COSE_KEY_CURVE_P521    = 3
	COSE_KEY_CURVE_X25519  = 4
	COSE_KEY_CURVE_X448    = 5
	COSE_KEY_CURVE_ED25519 = 7
	COSE_KEY_CURVE_ED448   = 8

	// https://tools.ietf.org/html/rfc8152#section-13.1.1
	COSE_KEY_PARAMETER_EC2_CRV     = -1
	COSE_KEY_PARAMETER_EC2_X_COORD = -2
	COSE_KEY_PARAMETER_EC2_Y_COORD = -3

	COSE_ALGORITHM_ECDSA_P256_WITH_SHA256 = -7
	COSE_ALGORITHM_EDDSA_ED25519          = -8
	COSE_ALGORITHM_ECDSA_P384_WITH_SHA384 = -35
	COSE_ALGORITHM_ECDSA_P521_WITH_SHA512 = -36

	COSE_ALGORITHM_RSASSA_PKCS1_V1_5_WITH_SHA256 = -257
	COSE_ALGORITHM_RSASSA_PKCS1_V1_5_WITH_SHA384 = -258
	COSE_ALGORITHM_RSASSA_PKCS1_V1_5_WITH_SHA512 = -259

	COSE_ALGORITHM_RSA_PSS_WITH_SHA256 = -37
	COSE_ALGORITHM_RSA_PSS_WITH_SHA384 = -38
	COSE_ALGORITHM_RSA_PSS_WITH_SHA512 = -39

	HASH_ALGORITHM_SHA_256 = "SHA-256"
	HASH_ALGORITHM_SHA_384 = "SHA-384"
	HASH_ALGORITHM_SHA_512 = "SHA-512"

	USER_VERIFICATION_REQUIREMENT_ANY         = 0
	USER_VERIFICATION_REQUIREMENT_REQUIRED    = 1
	USER_VERIFICATION_REQUIREMENT_PREFERRED   = 2
	USER_VERIFICATION_REQUIREMENT_DISCOURAGED = 3
)

type COSEKey struct {
	Kty int      `cbor:"1,keyasint"`
	Alg int      `cbor:"3,keyasint,omitempty"`
	Crv int      `cbor:"-1,keyasint,omitempty"`
	X   [32]byte `cbor:"-2,keyasint,omitempty"`
	Y   [32]byte `cbor:"-3,keyasint,omitempty"`
}

type ECDSASignature struct {
}

type AuthenticatorData struct {
	RPIDHash               [32]byte
	Flags                  byte
	Counter                uint32
	AttestedCredentialData *AttestedCredentialData
}

type AttestedCredentialData struct {
	AAGUID          [16]byte
	CredentialIDLen uint16
	CredentialID    []byte
}

type AttestationStatement struct {
	Alg        int      `cbor:"alg"`
	Sig        []byte   `cbor:"sig"`
	X5c        [][]byte `cbor:"x5c,omitempty"`
	ECDAAKeyId []byte   `cbor:"ecdaaKeyId,omitempty"`
}

type AttestationObject struct {
	Fmt      string               `cbor:"fmt,omitempty"`
	AttStmt  AttestationStatement `cbor:"attStmt,omitempty"`
	AuthData []byte               `cbor:"authData,omitempty"`
}

type RP_ENTITY_INFORMATION struct {
	Version uint32
	Id      *uint16
	Name    *uint16
	Icon    *uint16
}

type USER_ENTITY_INFORMATION struct {
	Version     uint32
	IdLen       uint32
	Id          uintptr // maybe *byte and pass &arr[0] ?
	Name        *uint16
	Icon        *uint16
	DisplayName *uint16
}

type COSE_CREDENTIAL_PARAMETER struct {
	Version        uint32
	CredentialType *uint16
	Alg            int64
}

type COSE_CREDENTIAL_PARAMETERS struct {
	Count                uint32
	CredentialParameters uintptr
}

type CLIENT_DATA struct {
	Version              uint32
	ClientDataJSONLength uint32
	ClientDataJSON       uintptr
	HashAlgId            *uint16
}

type AUTHENTICATOR_MAKE_CREDENTIAL_OPTIONS_V3 struct {
	Version             uint32
	TimeoutMilliseconds uint32 // Time that the operation is expected to complete within.
	// This is used as guidance, and can be overridden by the platform.
	CredentialList                  CREDENTIALS // Credentials used for exclusion.
	Extensions                      EXTENSIONS  // Optional extensions to parse when performing the operation.
	AuthenticatorAttachment         uint32      // Optional. Platform vs Cross-Platform Authenticators.
	RequireResidentKey              bool        // Optional. Require key to be resident or not. Defaulting to FALSE.
	UserVerificationRequirement     uint32      // User Verification Requirement.
	AttestationConveyancePreference uint32      // Attestation Conveyance Preference.
	Flags                           uint32      // Reserved for future Use

	// The following fields have been added in WEBAUTHN_AUTHENTICATOR_MAKE_CREDENTIAL_OPTIONS_VERSION_2
	CancellationId *syscall.GUID // Cancellation Id - Optional - See WebAuthNGetCancellationId

	// The following fields have been added in WEBAUTHN_AUTHENTICATOR_MAKE_CREDENTIAL_OPTIONS_VERSION_3
	ExcludeCredentialList CREDENTIAL_LIST // Exclude Credential List. If present, "CredentialList" will be ignored.
}

type CREDENTIAL struct {
	Version        uint32
	IdLen          uint32
	Id             uintptr
	CredentialType *uint16
}

type CREDENTIALS struct {
	Count       uint32
	Credentials uintptr
}

type CREDENTIAL_EX struct {
	Version        uint32
	IdLen          uint32
	Id             uintptr
	CredentialType *uint16
	Transports     uint32
}

type CREDENTIAL_LIST struct {
	Count       uint32
	Credentials uintptr
}

type EXTENSION struct {
	ExtensionIdentifier *uint16
	ExtensionLen        uint32
	Extension           uintptr
}

type EXTENSIONS struct {
	Count      uint32
	Extensions uintptr
}

type CREDENTIAL_ATTESTATION struct {
	Version              uint32  // Version of this structure, to allow for modifications in the future.
	FormatType           *uint16 // Attestation format type
	AuthenticatorDataLen uint32  // Size of cbAuthenticatorData.
	AuthenticatorData    uintptr // Authenticator data that was created for this credential.
	AttestationLen       uint32  // Size of CBOR encoded attestation information
	// 0 => encoded as CBOR null value.
	Attestation           uintptr // Encoded CBOR attestation information
	AttestationDecodeType uint32
	AttestationDecode     *[]byte //Depends on the AttestationDecodeType
	//  WEBAUTHN_ATTESTATION_DECODE_NONE
	//      NULL - not able to decode the CBOR attestation information
	//  WEBAUTHN_ATTESTATION_DECODE_COMMON
	//      PWEBAUTHN_COMMON_ATTESTATION;
	AttestationObjectLen uint32
	AttestationObject    uintptr // The CBOR encoded Attestation Object to be returned to the RP.
	CredentialIdLen      uint32
	CredentialId         uintptr // The CredentialId bytes extracted from the Authenticator Data.
	// Used by Edge to return to the RP.

	// Following fields have been added in WEBAUTHN_CREDENTIAL_ATTESTATION_VERSION_2
	Extensions EXTENSIONS

	// Following fields have been added in WEBAUTHN_CREDENTIAL_ATTESTATION_VERSION_3
	UsedTransport uint32 // One of the WEBAUTHN_CTAP_TRANSPORT_* bits will be set corresponding to
	// the transport that was used.
}

type AUTHENTICATOR_GET_ASSERTION_OPTIONS struct {
	Version             uint32 // Version of this structure, to allow for modifications in the future.
	TimeoutMilliseconds uint32 // Time that the operation is expected to complete within.
	// This is used as guidance, and can be overridden by the platform.
	CredentialList              CREDENTIALS // Allowed Credentials List.
	Extensions                  EXTENSIONS  // Optional extensions to parse when performing the operation.
	AuthenticatorAttachment     uint32      // Optional. Platform vs Cross-Platform Authenticators.
	UserVerificationRequirement uint32      // User Verification Requirement.
	Flags                       uint32      // Flags

	// The following fields have been added in WEBAUTHN_AUTHENTICATOR_GET_ASSERTION_OPTIONS_VERSION_2
	U2fAppId     *uint16 // Optional identifier for the U2F AppId. Converted to UTF8 before being hashed. Not lower cased.
	U2fAppIdUsed *bool   // If non-NULL, then, set to TRUE if the above pwszU2fAppid was used instead of PCWSTR pwszRpId;

	// The following fields have been added in WEBAUTHN_AUTHENTICATOR_GET_ASSERTION_OPTIONS_VERSION_3
	CancellationId *syscall.GUID // Cancellation Id - Optional - See WebAuthNGetCancellationId

	// The following fields have been added in WEBAUTHN_AUTHENTICATOR_GET_ASSERTION_OPTIONS_VERSION_4
	AllowCredentialList CREDENTIAL_LIST // Allow Credential List. If present, "CredentialList" will be ignored.
}

type ASSERTION struct {
	Version              uint32     // Version of this structure, to allow for modifications in the future.
	AuthenticatorDataLen uint32     // Size of cbAuthenticatorData.
	AuthenticatorData    uintptr    // Authenticator data that was created for this assertion.
	SignatureLen         uint32     // Size of pbSignature.
	Signature            uintptr    // Signature that was generated for this assertion.
	Credential           CREDENTIAL // Credential that was used for this assertion.
	UserIdLen            uint32     // Size of User Id
	UserId               uintptr
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

func UintptrToBytes(u uintptr, l uint32) []byte {
	if u != 0 {
		us := make([]byte, l)
		for i, _ := range us {
			us[i] = *(*byte)(unsafe.Pointer(u + uintptr(i)))
		}
		return us
	}
	return nil

}

func LPCWSTR(s string) *uint16 {
	w, _ := syscall.UTF16PtrFromString(s)
	return w
}

func GetApiVersionNumber() (int32, error) {
	r, _, err := procWebAuthNGetApiVersionNumber.Call()

	if err != syscall.Errno(0) {
		return 0, fmt.Errorf("WebAuthNGetApiVersionNumber returned %v", err)
	}

	return int32(r), nil
}

//HRESULT
//WINAPI
//WebAuthNAuthenticatorMakeCredential(
//_In_        HWND                                                hWnd,
//_In_        PCWEBAUTHN_RP_ENTITY_INFORMATION                    pRpInformation,
//_In_        PCWEBAUTHN_USER_ENTITY_INFORMATION                  pUserInformation,
//_In_        PCWEBAUTHN_COSE_CREDENTIAL_PARAMETERS               pPubKeyCredParams,
//_In_        PCWEBAUTHN_CLIENT_DATA                              pWebAuthNClientData,
//_In_opt_    PCWEBAUTHN_AUTHENTICATOR_MAKE_CREDENTIAL_OPTIONS    pWebAuthNMakeCredentialOptions,
//_Outptr_result_maybenull_ PWEBAUTHN_CREDENTIAL_ATTESTATION      *ppWebAuthNCredentialAttestation);
func AuthenticatorMakeCredential(hwnd uintptr,
	entity_information RP_ENTITY_INFORMATION,
	user_entity_information USER_ENTITY_INFORMATION,
	cose_parameters COSE_CREDENTIAL_PARAMETERS,
	client_data CLIENT_DATA,
	make_credential_options AUTHENTICATOR_MAKE_CREDENTIAL_OPTIONS_V3) (*CREDENTIAL_ATTESTATION, error) {

	var result uintptr

	r, _, err := procWebAuthNAuthenticatorMakeCredential.Call(
		hwnd,
		uintptr(unsafe.Pointer(&entity_information)),
		uintptr(unsafe.Pointer(&user_entity_information)),
		uintptr(unsafe.Pointer(&cose_parameters)),
		uintptr(unsafe.Pointer(&client_data)),
		uintptr(unsafe.Pointer(&make_credential_options)),
		uintptr(unsafe.Pointer(&result)),
	)

	if err != syscall.Errno(0) {
		return nil, fmt.Errorf("WebAuthNGetApiVersionNumber returned %v", err)
	}

	if r != 0 {
		return nil, fmt.Errorf("%v", errNoToStr(uint32(r)))
	}

	return (*CREDENTIAL_ATTESTATION)(unsafe.Pointer(result)), nil
}

//HRESULT
//WINAPI
//WebAuthNAuthenticatorGetAssertion(
//_In_        HWND                                                hWnd,
//_In_        LPCWSTR                                             pwszRpId,
//_In_        PCWEBAUTHN_CLIENT_DATA                              pWebAuthNClientData,
//_In_opt_    PCWEBAUTHN_AUTHENTICATOR_GET_ASSERTION_OPTIONS      pWebAuthNGetAssertionOptions,
//_Outptr_result_maybenull_ PWEBAUTHN_ASSERTION                   *ppWebAuthNAssertion);
func AuthenticatorGetAssertion(hwnd uintptr, RpId string, client_data CLIENT_DATA, assertion_options AUTHENTICATOR_GET_ASSERTION_OPTIONS) (*ASSERTION, error) {
	rpIdPtr, _ := syscall.UTF16PtrFromString(RpId)
	var result uintptr
	r, _, err := procWebAuthNAuthenticatorGetAssertion.Call(
		hwnd,
		uintptr(unsafe.Pointer(rpIdPtr)),
		uintptr(unsafe.Pointer(&client_data)),
		uintptr(unsafe.Pointer(&assertion_options)),
		uintptr(unsafe.Pointer(&result)),
	)

	if err != syscall.Errno(0) {
		return nil, fmt.Errorf("WebAuthNGetApiVersionNumber returned %v", err)
	}

	if r != 0 {
		return nil, fmt.Errorf("%v", errNoToStr(uint32(r)))
	}

	return (*ASSERTION)(unsafe.Pointer(result)), nil
}

func FreeCredentialAttestation(attestation *CREDENTIAL_ATTESTATION) error {
	r, _, err := procWebAuthNFreeCredentialAttestation.Call(uintptr(unsafe.Pointer(attestation)))

	if err != syscall.Errno(0) {
		return fmt.Errorf("WebAuthNGetApiVersionNumber returned %v", err)
	}

	if r != 0 {
		return fmt.Errorf("%v", errNoToStr(uint32(r)))
	}

	return nil
}

func FreeAssertion(assertion *ASSERTION) error {
	r, _, err := procWebAuthNFreeAssertion.Call(uintptr(unsafe.Pointer(assertion)))

	if err != syscall.Errno(0) {
		return fmt.Errorf("WebAuthNGetApiVersionNumber returned %v", err)
	}

	if r != 0 {
		return fmt.Errorf("%v", errNoToStr(uint32(r)))
	}

	return nil
}

func GetErrorName() {
	//r, _, err := procWebAuthNGetErrorName.Call()
}
