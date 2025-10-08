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
        
        // Convert to absolute path
        absProtoDir, err := filepath.Abs(protoDir)
        if err != nil {
            log.Printf("Failed to get absolute path for %s: %v", protoDir, err)
            absProtoDir = protoDir
        }
        
        log.Printf("DEBUG: protoFilePath=%s, protoDir=%s, absProtoDir=%s", protoFilePath, protoDir, absProtoDir)
        importPaths = []string{absProtoDir}
    }
    
    log.Printf("DEBUG: Using ImportPaths: %v", importPaths)
    
    compiler := protocompile.Compiler{
        Resolver: &protocompile.SourceResolver{
            ImportPaths: importPaths,
        },
    }
    
    // Make proto file path relative to the first import path
    absProtoFile, err := filepath.Abs(protoFilePath)
    if err == nil {
        relPath, err := filepath.Rel(importPaths[0], absProtoFile)
        if err == nil {
            log.Printf("DEBUG: Using relative proto path: %s", relPath)
            protoFilePath = relPath
        } else {
            log.Printf("DEBUG: Could not make relative path, using: %s", protoFilePath)
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

	return ProtoFile{files[0].Messages().ByName(protoreflect.Name(lookupType))}
}

func (p *ProtoFile) Encode(data string) []byte {
	dynamicMessage := dynamicpb.NewMessage(p.messageDesc)

	err := protojson.Unmarshal([]byte(data), dynamicMessage)

	if err != nil {
		log.Fatal(err)
	}

	encodedBytes, err := proto.Marshal(dynamicMessage)
	if err != nil {
		log.Fatal(err)
	}

	return encodedBytes
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
