package main

import (
	"flag"
	"log"
	"os"

	"awoo.nl/hatchcert"
)

// TODO:
//
// hatchcert
//     Ensure all certificates listed in the configuration file are within the
//     desired validity period.
//
// hatchcert account
//     Perform account registration and key management.
//
//     -refresh     Forcefully unset saved registration and fetch/create it again
//     -rekey       Forcefully create new account key
//
// hatchcert issue [name]
//     Forcefully issue certificates, ignoring current validity.

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

	var err error
	var want []hatchcert.Cert
	hook := false

	switch opt := flag.Arg(0); opt {
	case "reconcile", "":
		hook = true
		want, err = hatchcert.ScanCerts(*path, conf.Certs)
		if err != nil {
			log.Println("ScanCerts:", err)
		}

		if len(want) == 0 {
			// Nothing to do
			return
		}

	case "issue":
		want = conf.Certs

	case "account":

	case "status":
		hatchcert.Active(*path, conf.Certs)
		return

	case "help":
		log.Fatal("Commands: reconcile issue account status")

	default:
		log.Fatalf("Unknown command: %v", opt)
	}

	account := hatchcert.Account(*path)
	if err := hatchcert.Setup(account, conf.ACME, conf.Email); err != nil {
		log.Fatalln(err)
	}

	if len(want) == 0 {
		return
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
	must(hatchcert.ChallengesDNS(account.Client, conf.Challenge.DNS))

	// Default action: create or refresh certs
	failed := false
	issued := false
	for _, req := range want {
		req.PreferredChain = conf.PreferredChain // FIXME
		err := hatchcert.Issue(account, req)
		if err != nil {
			failed = true
			log.Println("Failed to issue:", err)
		} else {
			issued = true
		}
	}

	if issued && hook {
		for _, hook := range conf.UpdateHooks {
			if err := hatchcert.Hook(hook); err != nil {
				log.Println("Failed to run update hook:", err)
				failed = true
			}
		}
	}

	if failed {
		os.Exit(1)
	}
}
