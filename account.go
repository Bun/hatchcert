package hatchcert

import (
	"crypto"
	"log"
	"path/filepath"

	"github.com/go-acme/lego/v3/lego"
	"github.com/go-acme/lego/v3/registration"
)

var (
	// LEDirectoryProduction URL to the Let's Encrypt production
	LEDirectoryProduction = "https://acme-v02.api.letsencrypt.org/directory"

	// LEDirectoryStaging URL to the Let's Encrypt staging
	LEDirectoryStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

type SavedAccount struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	AccountKey   string                 `json:"account_key"`
}

type AccountMeta struct {
	Path        string
	AccountFile string
	Key         crypto.PrivateKey
	SavedAccount

	Client *lego.Client
	Config *lego.Config
}

func Account(path string) *AccountMeta {
	ac := &AccountMeta{
		Path:        path,
		AccountFile: filepath.Join(path, "account"),
	}
	if exists(ac.AccountFile) {
		if err := unmarshal(ac.AccountFile, &ac.SavedAccount); err != nil {
			panic(err)
		}
		k, err := parseKey(ac.SavedAccount.AccountKey)
		if err != nil {
			panic(err)
		}
		ac.Key = k
	}
	return ac
}

func Setup(acct *AccountMeta, email string) error {
	store := false
	if acct.Key == nil {
		pk, pv, err := generatePrivateKey(DefaultKeyType)
		if err != nil {
			return err
		}
		store = true
		acct.SavedAccount.AccountKey = pv
		acct.Key = pk
	} else if acct.SavedAccount.Email != email {
		// ...
	}
	acct.SavedAccount.Email = email

	var err error
	acct.Config = lego.NewConfig(acct)
	acct.Config.CADirURL = LEDirectoryStaging
	acct.Config.UserAgent = "hatchcert+lego/0.0.1"
	acct.Client, err = lego.NewClient(acct.Config)
	if err != nil {
		panic(err)
	}

	if true {
		if acct.Registration == nil {
			reg, err := acct.Client.Registration.ResolveAccountByKey()
			if err != nil {
				// Perhaps urn:ietf:params:acme:error:accountDoesNotExist
				log.Printf("%T: %v", err, err)
			} else {
				acct.Registration = reg
				store = true
			}
		}
		if acct.Registration == nil {
			reg, err := acct.Client.Registration.Register(registration.RegisterOptions{
				TermsOfServiceAgreed: true,
			})
			if err != nil {
				return err
			}
			acct.Registration = reg
			store = true
		}
	}
	if store {
		if err := marshal(acct.AccountFile, acct.SavedAccount); err != nil {
			return err
		}
	}

	return nil
}

//
// Lego integration
//

func (am *AccountMeta) GetRegistration() *registration.Resource {
	return am.Registration
}

func (am *AccountMeta) GetEmail() string {
	return am.Email
}

func (am *AccountMeta) GetPrivateKey() crypto.PrivateKey {
	return am.Key
}
