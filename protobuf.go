package protobuf

import (
	"context"
	"log"
	"path/filepath"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/encoding/protojson"

	"go.k6.io/k6/js/modules"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func init() {
	modules.Register("k6/x/protobuf", new(Protobuf))
}

type Protobuf struct{}

type ProtoFile struct {
	messageDesc protoreflect.MessageDescriptor
}

func (p *Protobuf) Load(protoFilePath, lookupType string, importPaths ...string) ProtoFile {
	// Default import paths if none provided
	if len(importPaths) == 0 {
		protoDir := filepath.Dir(protoFilePath)
		absProtoDir, err := filepath.Abs(protoDir)
		if err != nil {
			absProtoDir = protoDir
		}
		importPaths = []string{absProtoDir}
	}

	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			ImportPaths: importPaths,
		},
	}

	// Make proto file path relative to import path for protocompile
	absProtoFile, err := filepath.Abs(protoFilePath)
	if err == nil {
		relPath, err := filepath.Rel(importPaths[0], absProtoFile)
		if err == nil {
			protoFilePath = relPath
		}
	}

	files, err := compiler.Compile(context.Background(), protoFilePath)
	if err != nil {
		log.Fatal(err)
	}
	if files == nil {
		log.Fatal("No files were passed as arguments")
	}
	if len(files) == 0 {
		log.Fatal("Zero files were parsed")
	}

	// Extract simple name from fully qualified name (e.g., "iot.test_messages.Ping" -> "Ping")
	simpleName := lookupType
	for i := len(lookupType) - 1; i >= 0; i-- {
		if lookupType[i] == '.' {
			simpleName = lookupType[i+1:]
			break
		}
	}

	messageDesc := files[0].Messages().ByName(protoreflect.Name(simpleName))
	if messageDesc == nil {
		log.Fatalf("Message type %s not found", lookupType)
	}

	return ProtoFile{messageDesc}
}

func (p *ProtoFile) Encode(data string) string {
	dynamicMessage := dynamicpb.NewMessage(p.messageDesc)

	err := protojson.Unmarshal([]byte(data), dynamicMessage)

	if err != nil {
		log.Fatal(err)
	}

	encodedBytes, err := proto.Marshal(dynamicMessage)
	if err != nil {
		log.Fatal(err)
	}

	return string(encodedBytes)
}

func (p *ProtoFile) Decode(decodedBytes []byte) string {

	decodedMessage := dynamicpb.NewMessage(p.messageDesc)

	err := proto.Unmarshal(decodedBytes, decodedMessage)
	if err != nil {
		log.Fatal(err)
	}

	marshalOptions := protojson.MarshalOptions{
		UseProtoNames: true,
	}

	jsonString, err := marshalOptions.Marshal(decodedMessage)
	if err != nil {
		log.Fatal(err)
	}

	return string(jsonString)
}
