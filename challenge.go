package hatchcert

import (
	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/lego"
)

func ChallengesHTTP(client *lego.Client, ps []challenge.Provider) error {
	for _, p := range ps {
		if err := client.Challenge.SetHTTP01Provider(p); err != nil {
			return err
		}
	}
	return nil
}

func ChallengesDNS(client *lego.Client, ps []challenge.Provider) error {
	for _, p := range ps {
		if err := client.Challenge.SetDNS01Provider(p); err != nil {
			return err
		}
	}
	return nil
}
