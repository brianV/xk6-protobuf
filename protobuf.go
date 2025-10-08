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
    // Show first 200 chars of JSON input
    jsonPreview := jsonString
    if len(jsonString) > 200 {
        jsonPreview = jsonString[:200]
    }
    log.Printf("DEBUG Encode: input JSON length=%d, first chars=%s", len(jsonString), jsonPreview)
    
    msg := dynamicpb.NewMessage(p.messageDesc)
    err := protojson.Unmarshal([]byte(jsonString), msg)
    if err != nil {
        log.Printf("ERROR: protojson.Unmarshal failed: %v", err)
        panic(err)
    }
    
    // Count how many fields are set
    fieldCount := 0
    msg.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
        log.Printf("DEBUG: Field %s (number %d) = %v", fd.Name(), fd.Number(), v)
        fieldCount++
        return true
    })
    log.Printf("DEBUG: After unmarshal, %d fields are set", fieldCount)
    
    data, err := proto.Marshal(msg)
    if err != nil {
        panic(err)
    }
    
    // Show first 20 bytes in hex
    hexLen := 20
    if len(data) < 20 {
        hexLen = len(data)
    }
    log.Printf("DEBUG: Encoded to %d bytes, first %d hex: % x", len(data), hexLen, data[:hexLen])
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
