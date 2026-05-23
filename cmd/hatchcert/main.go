package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"awoo.nl/hatchcert"
)

// hatchcert
//     Ensure all certificates listed in the configuration file are within the
//     desired validity period.
//
// TODO:
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
	verbose := flag.Bool("v", false, "Always print log")
	flag.Parse()

	logbuf := hatchcert.SetupLogger(*verbose)

	conf, err := hatchcert.Conf(*cfile)
	if err != nil {
		logbuf.Emit()
		fmt.Fprintln(os.Stderr, "Configuration error:", err)
		os.Exit(1)
	}
	if !conf.AcceptedTOS {
		log.Fatalln("You must accept the terms of service")
	}
	if conf.Email == "" {
		log.Fatalln("Email is required")
	}

	var want []hatchcert.Cert
	hook := false

	switch opt := flag.Arg(0); opt {
	case "reconcile", "":
		hook = true
		want, err = hatchcert.ScanCerts(*path, conf.Certs)
		if err != nil {
			log.Println("ScanCerts:", err)
		}

	case "issue":
		want = hatchcert.LoadCerts(*path, conf.Certs)

		// Filter to only specified names if any args provided
		names := flag.Args()[1:]
		if len(names) > 0 {
			var filtered []hatchcert.Cert
			for _, n := range names {
				for _, w := range want {
					if w.Name == n {
						filtered = append(filtered, w)
					}
				}
			}
			want = filtered
		}

		// Skip any expiration checks
		for i, w := range want {
			w.Expired = true
			want[i] = w
		}

	case "account":
		hatchcert.AccountInfo(*path, conf)
		return

	case "status":
		hatchcert.Active(*path, conf.Certs)
		return

	case "help":
		log.Fatal("Commands: reconcile issue account status")

	default:
		log.Fatalf("Unknown command: %v", opt)
	}

	if len(conf.Solvers) == 0 {
		log.Fatalln("Cannot issue certificates without challenge method")
	}

	if err := os.MkdirAll(*path, 0755); err != nil {
		log.Fatalln(err)
	}

	failed := false
	h := hatchcert.New(*path, conf)
	if err := h.EnsureAccount(); err != nil {
		failed = true
		slog.Error("Failed to obtain ACME account",
			"err", err)
		goto uhoh
	}

	// Default action: create or refresh certs
	if len(want) == 0 {
		slog.Debug("No certificates to issue")
	} else {
		if conf.WebServer != "" {
			srv := http.Server{
				Addr:              conf.WebServer,
				Handler:           hatchcert.Mux,
				ReadTimeout:       time.Minute,
				ReadHeaderTimeout: time.Minute,
				WriteTimeout:      time.Minute,
				IdleTimeout:       time.Minute,
			}
			// TODO: avoid racing things and create listener first
			go func() {
				err := srv.ListenAndServe()
				if err != nil {
					log.Fatalln("WebServer setup failed:", err)
				}
			}()
		}

		issued := false
		for _, req := range want {
			if !req.Expired && !h.NeedsRenewal(req) {
				slog.Debug("Certificate does not need renewal",
					"domains", req.Domains)
				continue
			}
			err := h.Issue(req)
			if err != nil {
				failed = true
				log.Println("Failed to issue:", err)
			} else {
				slog.Info("Issued certificate",
					"domains", req.Domains)
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
	}

uhoh:
	if failed || *verbose {
		logbuf.Emit()
	}
	if failed {
		os.Exit(1)
	}
}
