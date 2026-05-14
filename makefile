.PHONY: hatchcert
hatchcert:
	go build -v -o hatchcert ./cmd/hatchcert

test:
	go test ./...

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

	go build -v -o build/usr/bin/hatchcert ./cmd/hatchcert
	fakeroot dpkg-deb -z2 --build build/ "hatchcert-${deb_version}.deb"
