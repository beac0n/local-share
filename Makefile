FORCE: ;

build: FORCE
	go get -d ./...
	go build -ldflags "-s" -o build/go-remote src/main/main.go

clean:
	rm -rf build
