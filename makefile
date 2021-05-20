.PHONY: hatchcert
hatchcert:
	go build -v -o hatchcert ./cmd/hatchcert

test:
	go test ./...

deb:
	rm -rf build/
	mkdir -p build/DEBIAN
	mkdir -p build/usr/bin
	mkdir -p build/etc/hatchcert
	cp debian/control build/DEBIAN/control
	cp dist/config build/etc/hatchcert/config.example
	go build -v -o build/usr/bin/hatchcert ./cmd/hatchcert
	fakeroot dpkg-deb -z2 --build build/ hatchcert-0.2.deb
