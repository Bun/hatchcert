package main

import (
	"flag"
	"log"

	"awoo.nl/hatchcert"
)

// TODO:
//
// hatchcert account
//     -refresh     Forcefully unset saved registration and fetch/create it again
//     -rekey       Forcefully create new account key

func main() {
	path := flag.String("path", "/var/lib/acme", "Output directory")
	cfile := flag.String("conf", "/etc/hatchcert/config", "Config file")
	flag.Parse()

	conf := hatchcert.Conf(*cfile)
	if !conf.AcceptedTOS {
		log.Fatalln("You must accept the terms of service")
	}
	if conf.Email == "" {
		log.Fatalln("Email is required")
	}

	account := hatchcert.Account(*path)
	if err := hatchcert.Setup(account, conf.ACME, conf.Email); err != nil {
		log.Fatalln(err)
	}

	if len(conf.Challenge.HTTP)+len(conf.Challenge.DNS) == 0 {
		log.Fatalln("Cannot issue certificates without challenge method")
	}

	must := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	must(hatchcert.ChallengesHTTP(account.Client, conf.Challenge.HTTP))
	must(hatchcert.ChallengesDNS(account.Client, conf.Challenge.HTTP))

	// Default action: create or refresh certs
	err := hatchcert.Issue(
		account,
		conf.Certs)
	if err != nil {
		log.Fatalln("Failed to issue:", err)
	}
}
