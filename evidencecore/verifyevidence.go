package evidencecore

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"

	"github.com/jfrog/jfrog-cli-artifactory/evidencecore/cryptolib"
	"github.com/jfrog/jfrog-cli-artifactory/evidencecore/dsse"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
)

type EvidenceVerifyCommand struct {
	serverDetails *config.ServerDetails
	key           string
	evidenceName  string
}

func NewEvidenceVerifyCommand() *EvidenceVerifyCommand {
	return &EvidenceVerifyCommand{}
}

func (evc *EvidenceVerifyCommand) SetServerDetails(serverDetails *config.ServerDetails) *EvidenceVerifyCommand {
	evc.serverDetails = serverDetails
	return evc
}

func (evc *EvidenceVerifyCommand) SetKey(key string) *EvidenceVerifyCommand {
	evc.key = key
	return evc
}

func (evc *EvidenceVerifyCommand) SetEvidenceName(evidenceName string) *EvidenceVerifyCommand {
	evc.evidenceName = evidenceName
	return evc
}

func (evc *EvidenceVerifyCommand) CommandName() string {
	return "verify_create"
}

func (evc *EvidenceVerifyCommand) ServerDetails() (*config.ServerDetails, error) {
	return evc.serverDetails, nil
}

func (evc *EvidenceVerifyCommand) Run() error {
	// Load evidence from file system
	dsseFile, err := os.ReadFile(evc.evidenceName)
	if err != nil {
		return err
	}

	// We assume that the evidence name is a path, so we can assume that it is a local file
	key, err := os.ReadFile(evc.key)
	if err != nil {
		return err
	}
	// Load key from file
	loadedKey, err := cryptolib.LoadKey(key)
	if err != nil {
		return err
	}
	// Verify evidence with key
	dsseEnvelope := dsse.Envelope{}
	err = json.Unmarshal(dsseFile, &dsseEnvelope)
	if err != nil {
		return err
	}

	// Decode payload and key
	decodedPayload, err := base64.StdEncoding.DecodeString(dsseEnvelope.Payload)
	if err != nil {
		return err
	}
	decodedKey, err := base64.StdEncoding.DecodeString(dsseEnvelope.Signatures[0].Sig) // This stage we support only one signature
	if err != nil {
		return err
	}

	// Create PAE
	paeEnc := dsse.PAE(dsseEnvelope.PayloadType, decodedPayload)

	// create actual verifier
	switch loadedKey.KeyType {
	case cryptolib.ECDSAKeyType:
		ecdsaSinger, err := cryptolib.NewECDSASignerVerifierFromSSLibKey(loadedKey)
		if err != nil {
			return err
		}
		err = ecdsaSinger.Verify(paeEnc, decodedKey)
	case cryptolib.RSAKeyType:
		rsaSinger, err := cryptolib.NewRSAPSSSignerVerifierFromSSLibKey(loadedKey)
		if err != nil {
			return err
		}
		err = rsaSinger.Verify(paeEnc, decodedKey)
	case cryptolib.ED25519KeyType:
		ed25519Singer, err := cryptolib.NewED25519SignerVerifierFromSSLibKey(loadedKey)
		if err != nil {
			return err
		}
		err = ed25519Singer.Verify(paeEnc, decodedKey)
	default:
		return errors.New("unsupported key type")
	}
	return err
}
