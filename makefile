.PHONY: hatchcert
hatchcert:
	go build -v -o hatchcert ./cmd/hatchcert

test:
	go test ./...

SOURCE_DATE_EPOCH=$(shell git log -1 --format=%ct)
deb_version=$(shell grep Version debian/control | cut '-d ' -f 2)

deb:
	rm -rf build/
	mkdir -p build/DEBIAN
	mkdir -p build/usr/bin
	mkdir -p build/etc/hatchcert
	mkdir -p build/usr/lib/systemd/system
	cp debian/control build/DEBIAN/control
	cp dist/hatchcert.cron build/etc/hatchcert/hatchcert.cron
	cp dist/config.example build/etc/hatchcert/config.example
	cp dist/update-hook build/etc/hatchcert/update-hook.example
	cp dist/hatchcert.service build/usr/lib/systemd/system/hatchcert.service
	cp dist/hatchcert.timer build/usr/lib/systemd/system/hatchcert.timer
	CGO_ENABLED=0 go build -v -trimpath -ldflags="-buildid= -X main.buildTime=$(SOURCE_DATE_EPOCH)" -o build/usr/bin/hatchcert ./cmd/hatchcert
	SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH) fakeroot dpkg-deb -z2 --build build/ "hatchcert-${deb_version}.deb"
