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
        absProtoDir, _ := filepath.Abs(protoDir)
        importPaths = []string{absProtoDir}
    }
    
    compiler := protocompile.Compiler{
        Resolver: &protocompile.SourceResolver{
            ImportPaths: importPaths,
        },
    }
    
    // Make proto file path relative to import path for protocompile
    absProtoFile, _ := filepath.Abs(protoFilePath)
    relPath, _ := filepath.Rel(importPaths[0], absProtoFile)

    files, err := compiler.Compile(context.Background(), relPath)
    if err != nil {
        log.Fatal(err)
    }
    if files == nil || len(files) == 0 {
        log.Fatal("No files were compiled")
    }

    // Extract simple name from fully qualified name (e.g., "infra.iot_comms_poc.Ping" -> "Ping")
    simpleName := lookupType
    if lastDot := len(lookupType) - 1; lastDot >= 0 {
        for i := lastDot; i >= 0; i-- {
            if lookupType[i] == '.' {
                simpleName = lookupType[i+1:]
                break
            }
        }
    }

    msgDesc := files[0].Messages().ByName(protoreflect.Name(simpleName))
    if msgDesc == nil {
        log.Fatalf("Message type %s not found", lookupType)
    }

    return ProtoFile{msgDesc}
}

func (p *ProtoFile) Encode(jsonString string) []byte {
    log.Printf("DEBUG Encode: input JSON length=%d, first 200 chars=%s", len(jsonString), jsonString[:min(200, len(jsonString))])
    
    msg := dynamicpb.NewMessage(p.messageDesc)
    err := protojson.Unmarshal([]byte(jsonString), msg)
    if err != nil {
        log.Printf("ERROR: protojson.Unmarshal failed: %v", err)
        panic(err)
    }
    
    log.Printf("DEBUG: After unmarshal, message has %d fields set", len(msg.ProtoReflect().Descriptor().Fields()))
    
    data, err := proto.Marshal(msg)
    if err != nil {
        panic(err)
    }
    
    log.Printf("DEBUG: Encoded to %d bytes, first 20 hex: % x", len(data), data[:min(20, len(data))])
    return data
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
