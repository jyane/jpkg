.PHONY: all clean install

all: jpkg proto/jpkg.pb.go

clean:
	rm proto/jpkg.pb.go jpkg
	rm -rf repos/

install:
	cp jpkg ~/bin/jpkg

jpkg: *.go proto/jpkg.pb.go
	go build -o jpkg

proto/jpkg.pb.go: proto/jpkg.proto
	protoc --go_out=. --go_opt=paths=source_relative proto/jpkg.proto
