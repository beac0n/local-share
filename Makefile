FORCE: ;

build: FORCE
	go get -d ./...
	go build -ldflags "-s" -o build/local-share src/main/main.go

clean:
	rm -rf build
