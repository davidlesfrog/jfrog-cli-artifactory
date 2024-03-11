package cryptolib

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"strings"
)

const (
	RSAKeyType       = "rsa"
	RSAKeyScheme     = "rsassa-pss-sha256"
	RSAPrivateKeyPEM = "RSA PRIVATE KEY"
)

// RSAPSSSignerVerifier is a dsse.SignerVerifier compliant interface to sign and
// verify signatures using RSA keys following the RSA-PSS scheme.
type RSAPSSSignerVerifier struct {
	keyID   string
	private *rsa.PrivateKey
	public  *rsa.PublicKey
}

// NewRSAPSSSignerVerifierFromSSLibKey creates an RSAPSSSignerVerifier from an
// SSLibKey.
func NewRSAPSSSignerVerifierFromSSLibKey(key *SSLibKey) (*RSAPSSSignerVerifier, error) {
	if len(key.KeyVal.Public) == 0 {
		return nil, ErrInvalidKey
	}

	_, publicParsedKey, err := decodeAndParsePEM([]byte(key.KeyVal.Public))
	if err != nil {
		return nil, fmt.Errorf("unable to create RSA-PSS signerverifier: %w", err)
	}

	puk, ok := publicParsedKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("couldnt convert to rsa public key")
	}

	if len(key.KeyVal.Private) > 0 {
		_, privateParsedKey, err := decodeAndParsePEM([]byte(key.KeyVal.Private))
		if err != nil {
			return nil, fmt.Errorf("unable to create RSA-PSS signerverifier: %w", err)
		}

		pkParsed, ok := privateParsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("couldnt convert to rsa private key")
		}

		return &RSAPSSSignerVerifier{
			keyID:   key.KeyID,
			public:  puk,
			private: pkParsed,
		}, nil
	}

	return &RSAPSSSignerVerifier{
		keyID:   key.KeyID,
		public:  puk,
		private: nil,
	}, nil
}

// Sign creates a signature for `data`.
func (sv *RSAPSSSignerVerifier) Sign(data []byte) ([]byte, error) {
	if sv.private == nil {
		return nil, ErrNotPrivateKey
	}

	hashedData := hashBeforeSigning(data, sha256.New())

	return rsa.SignPKCS1v15(rand.Reader, sv.private, crypto.SHA256, hashedData)
}

// Verify verifies the `sig` value passed in against `data`.
func (sv *RSAPSSSignerVerifier) Verify(data []byte, sig []byte) error {
	hashedData := hashBeforeSigning(data, sha256.New())

	if err := rsa.VerifyPKCS1v15(sv.public, crypto.SHA256, hashedData, sig); err != nil {
		return ErrSignatureVerificationFailed
	}

	return nil
}

// KeyID returns the identifier of the key used to create the
// RSAPSSSignerVerifier instance.
func (sv *RSAPSSSignerVerifier) KeyID() (string, error) {
	return sv.keyID, nil
}

// Public returns the public portion of the key used to create the
// RSAPSSSignerVerifier instance.
func (sv *RSAPSSSignerVerifier) Public() crypto.PublicKey {
	return sv.public
}

// LoadRSAPSSKeyFromBytes is a function that takes a byte array as input. This
// byte array should represent a PEM encoded RSA key, as PEM encoding is
// required.  The function returns an SSLibKey instance, which is a struct that
// holds the key data.
//
// Deprecated: use LoadKey() for all key types, RSA is no longer the only key
// that uses PEM serialization.
func LoadRSAPSSKeyFromBytes(contents []byte) (*SSLibKey, error) {
	pemData, keyObj, err := decodeAndParsePEM(contents)
	if err != nil {
		return nil, fmt.Errorf("unable to load RSA key from file: %w", err)
	}

	key := &SSLibKey{
		KeyType:             RSAKeyType,
		Scheme:              RSAKeyScheme,
		KeyIDHashAlgorithms: KeyIDHashAlgorithms,
		KeyVal:              KeyVal{},
	}

	pubKeyBytes, err := marshalAndGeneratePEM(keyObj)
	if err != nil {
		return nil, fmt.Errorf("unable to load RSA key from file: %w", err)
	}
	key.KeyVal.Public = strings.TrimSpace(string(pubKeyBytes))

	if _, ok := keyObj.(*rsa.PrivateKey); ok {
		key.KeyVal.Private = strings.TrimSpace(string(generatePEMBlock(pemData.Bytes, RSAPrivateKeyPEM)))
	}

	if len(key.KeyID) == 0 {
		keyID, err := calculateKeyID(key)
		if err != nil {
			return nil, fmt.Errorf("unable to load RSA key from file: %w", err)
		}
		key.KeyID = keyID
	}

	return key, nil
}

func marshalAndGeneratePEM(key interface{}) ([]byte, error) {
	var pubKeyBytes []byte
	var err error

	switch k := key.(type) {
	case *rsa.PublicKey:
		pubKeyBytes, err = x509.MarshalPKIXPublicKey(k)
	case *rsa.PrivateKey:
		pubKeyBytes, err = x509.MarshalPKIXPublicKey(k.Public())
	default:
		return nil, fmt.Errorf("unexpected key type: %T", k)
	}

	if err != nil {
		return nil, err
	}

	return generatePEMBlock(pubKeyBytes, PublicKeyPEM), nil
}
