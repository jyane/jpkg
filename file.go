package main

import (
	"fmt"
	"os"

	pb "github.com/jyane/jpkg/proto"
	"google.golang.org/protobuf/encoding/prototext"
)

func fileAvailable(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else {
		return false
	}
}

func writeJpkgLockFile(path string, l *pb.JpkgFile) error {
	options := prototext.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	data, err := options.Marshal(l)
	if err != nil {
		return fmt.Errorf("failed to encode to text proto file: %v", err)
	}
	err = os.WriteFile(path, data, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	return nil
}

func readJpkgFile(path string) (*pb.JpkgFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f := &pb.JpkgFile{}
	if err := prototext.Unmarshal(data, f); err != nil {
		return nil, err
	}
	return f, nil
}
