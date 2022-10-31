package keyman

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"math/big"
	"ncryptagent/ncrypt"
)

func unmarshalRSA(buf []byte) (*rsa.PublicKey, error) {
	// BCRYPT_RSA_BLOB -- https://learn.microsoft.com/en-us/windows/win32/api/bcrypt/ns-bcrypt-bcrypt_rsakey_blob
	header := struct {
		Magic         uint32
		BitLength     uint32
		PublicExpSize uint32
		ModulusSize   uint32
		UnusedPrime1  uint32
		UnusedPrime2  uint32
	}{}

	r := bytes.NewReader(buf)
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if header.Magic != ncrypt.RSA1Magic {
		return nil, fmt.Errorf("invalid header magic %x", header.Magic)
	}

	if header.PublicExpSize > 8 {
		return nil, fmt.Errorf("unsupported public exponent size (%d bits)", header.PublicExpSize*8)
	}

	// the exponent is in BigEndian format, so read the data into the right place in the buffer
	exp := make([]byte, 8)
	n, err := r.Read(exp[8-header.PublicExpSize:])

	if err != nil {
		return nil, fmt.Errorf("failed to read public exponent %w", err)
	}

	if n != int(header.PublicExpSize) {
		return nil, fmt.Errorf("failed to read correct public exponent size, read %d expected %d", n, int(header.PublicExpSize))
	}

	mod := make([]byte, header.ModulusSize)
	n, err = r.Read(mod)

	if err != nil {
		return nil, fmt.Errorf("failed to read modulus %w", err)
	}

	if n != int(header.ModulusSize) {
		return nil, fmt.Errorf("failed to read correct modulus size, read %d expected %d", n, int(header.ModulusSize))
	}

	pub := &rsa.PublicKey{
		N: new(big.Int).SetBytes(mod),
		E: int(binary.BigEndian.Uint64(exp)),
	}
	return pub, nil
}

func unmarshalECC(buf []byte, curve elliptic.Curve) (*ecdsa.PublicKey, error) {
	// BCRYPT_ECCKEY_BLOB -- https://learn.microsoft.com/en-us/windows/win32/api/bcrypt/ns-bcrypt-bcrypt_ecckey_blob
	header := struct {
		Magic uint32
		Key   uint32
	}{}

	r := bytes.NewReader(buf)
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if expectedMagic, ok := ncrypt.CurveMagicMap[curve.Params().Name]; ok {
		if expectedMagic != header.Magic {
			return nil, fmt.Errorf("elliptic curve blob did not contain expected magic")
		}
	}

	keyX := make([]byte, header.Key)
	n, err := r.Read(keyX)
	if err != nil {
		return nil, fmt.Errorf("failed to read key X %w", err)
	}

	if n != int(header.Key) {
		return nil, fmt.Errorf("failed to read key X size, read %d expected %d", n, int(header.Key))
	}

	keyY := make([]byte, header.Key)
	n, err = r.Read(keyY)
	if err != nil {
		return nil, fmt.Errorf("failed to read key Y %w", err)
	}

	if n != int(header.Key) {
		return nil, fmt.Errorf("failed to read key Y size, read %d expected %d", n, int(header.Key))
	}

	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(keyX),
		Y:     new(big.Int).SetBytes(keyY),
	}
	return pub, nil
}

func getPublicKey(kh uintptr) (crypto.PublicKey, error) {
	algGroup, err := ncrypt.NCryptGetPropertyStr(kh, ncrypt.NCRYPT_ALGORITHM_GROUP_PROPERTY)
	if err != nil {
		return nil, fmt.Errorf("unable to get NCRYPT_ALGORITHM_GROUP_PROPERTY: %w", err)
	}

	var pub crypto.PublicKey
	switch algGroup {
	case "ECDSA":
		buf, err := ncrypt.NCryptExportKey(kh, ncrypt.BCRYPT_ECCPUBLIC_BLOB)
		if err != nil {
			return nil, fmt.Errorf("failed to export ECC public key: %w", err)
		}
		curveName, err := ncrypt.NCryptGetPropertyStr(kh, ncrypt.NCRYPT_ALGORITHM_PROPERTY)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve ECC curve name: %w", err)
		}

		if _, ok := ncrypt.CurveNames[curveName]; !ok {
			fmt.Printf("Curve name not found, attempting to retrieve NCRYPT_ECC_CURVE_NAME_PROPERTY")
			curveName, err = ncrypt.NCryptGetPropertyStr(kh, ncrypt.NCRYPT_ECC_CURVE_NAME_PROPERTY)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve ECC curve name: %w", err)
			}
		}

		fmt.Printf("CurveName is %s\n", curveName)
		pub, err = unmarshalECC(buf, ncrypt.CurveNames[curveName])
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal ECC public key: %w", err)
		}
	case "RSA":
		buf, err := ncrypt.NCryptExportKey(kh, ncrypt.BCRYPT_RSAPUBLIC_BLOB)
		if err != nil {
			return nil, fmt.Errorf("failed to export %v public key: %w", algGroup, err)
		}
		pub, err = unmarshalRSA(buf)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal %v public key: %w", algGroup, err)
		}
	default:
		return nil, fmt.Errorf("unhandled algorithm group %v retrieved from key", algGroup)
	}

	return pub, nil
}
