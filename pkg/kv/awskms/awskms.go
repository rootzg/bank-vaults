package awskms

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/banzaicloud/bank-vaults/pkg/kv"
)

type awsKMS struct {
	store      kv.Service
	kmsService *kms.KMS

	kmsID string
}

var _ kv.Service = &awsKMS{}

// NewWithSession creates a new kv.Service encrypted by AWS KMS with and existing AWS Session
func NewWithSession(sess *session.Session, store kv.Service, kmsID string) (kv.Service, error) {
	if kmsID == "" {
		return nil, fmt.Errorf("invalid kmsID specified: '%s'", kmsID)
	}

	return &awsKMS{
		store:      store,
		kmsService: kms.New(sess),
		kmsID:      kmsID,
	}, nil
}

// New creates a new kv.Service encrypted by AWS KMS
func New(store kv.Service, kmsID string) (kv.Service, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return NewWithSession(sess, store, kmsID)
}

func (a *awsKMS) decrypt(cipherText []byte) ([]byte, error) {
	out, err := a.kmsService.Decrypt(&kms.DecryptInput{
		CiphertextBlob: cipherText,
		EncryptionContext: map[string]*string{
			"Tool": aws.String("bank-vaults"),
		},
		GrantTokens: []*string{},
	})
	return out.Plaintext, err
}

func (a *awsKMS) Get(key string) ([]byte, error) {
	cipherText, err := a.store.Get(key)
	if err != nil {
		return nil, err
	}

	return a.decrypt(cipherText)
}

func (a *awsKMS) encrypt(plainText []byte) ([]byte, error) {

	out, err := a.kmsService.Encrypt(&kms.EncryptInput{
		KeyId:     aws.String(a.kmsID),
		Plaintext: plainText,
		EncryptionContext: map[string]*string{
			"Tool": aws.String("bank-vaults"),
		},
		GrantTokens: []*string{},
	})
	return out.CiphertextBlob, err
}

func (a *awsKMS) Set(key string, val []byte) error {
	cipherText, err := a.encrypt(val)

	if err != nil {
		return err
	}

	return a.store.Set(key, cipherText)
}

func (a *awsKMS) Test(key string) error {
	inputString := "test"

	err := a.store.Test(key)
	if err != nil {
		return fmt.Errorf("test of backend store failed: %s", err.Error())
	}

	cipherText, err := a.encrypt([]byte(inputString))
	if err != nil {
		return err
	}

	plainText, err := a.decrypt(cipherText)
	if err != nil {
		return err
	}

	if string(plainText) != inputString {
		return fmt.Errorf("encrypted and decryped text doesn't match: exp: '%v', act: '%v'", inputString, string(plainText))
	}

	return nil
}
