package hatchcert

import (
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/lego"
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
