package protobuf

import (
	"context"
	"log"
	"path/filepath"

	"github.com/bufbuild/protocompile"
	"go.k6.io/k6/js/modules"
	"google.golang.org/protobuf/encoding/protojson"
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

// Load compiles a proto file and returns a ProtoFile for encoding/decoding messages
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
	if files == nil || len(files) == 0 {
		log.Fatal("No files were compiled")
	}

	// Extract simple name from fully qualified name (e.g., "infra.iot_comms_poc.Ping" -> "Ping")
	simpleName := lookupType
	for i := len(lookupType) - 1; i >= 0; i-- {
		if lookupType[i] == '.' {
			simpleName = lookupType[i+1:]
			break
		}
	}

	msgDesc := files[0].Messages().ByName(protoreflect.Name(simpleName))
	if msgDesc == nil {
		log.Fatalf("Message type %s not found", lookupType)
	}

	return ProtoFile{msgDesc}
}

// Encode converts JSON string to protobuf binary format
// Returns as Go string (binary safe) for compatibility with xk6-nats
func (p *ProtoFile) Encode(jsonString string) string {
	msg := dynamicpb.NewMessage(p.messageDesc)
	
	err := protojson.Unmarshal([]byte(jsonString), msg)
	if err != nil {
		log.Printf("ERROR: protojson.Unmarshal failed: %v", err)
		panic(err)
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}

	// Return as string to avoid UTF-8 corruption when passing to xk6-nats
	return string(data)
}

// Decode converts protobuf binary format to JSON string
func (p *ProtoFile) Decode(decodedBytes []byte) string {
	msg := dynamicpb.NewMessage(p.messageDesc)
	
	err := proto.Unmarshal(decodedBytes, msg)
	if err != nil {
		panic(err)
	}

	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		panic(err)
	}

	return string(jsonBytes)
}
