package decoding

import (
	"reflect"
	"strings"
	"unsafe"

	"github.com/modern-go/reflect2"
	jsoniter "github.com/plusplusben/json-iterator-go"
	"github.com/skuid/picard/metadata"
	"github.com/skuid/warden/pkg/mapvalue"
)

// Config specifies options for the picard decoder
type Config struct {
	TagKey string
}

var globalConfig *Config

// GetDecoder returns a decoder that implements the standard encoding/json api
func GetDecoder(config *Config) jsoniter.API {
	if config == nil {
		config = &Config{}
	}
	if config.TagKey == "" {
		config.TagKey = "json"
	}
	globalConfig = config
	api := jsoniter.Config{
		EscapeHTML:             true,
		SortMapKeys:            true,
		ValidateJsonRawMessage: true,
		OnlyTaggedField:        true,
		TagKey:                 config.TagKey,
	}.Froze()
	api.RegisterExtension(&picardExtension{jsoniter.DummyExtension{}})
	return api
}

// JsonIter Extension for metadata marshalling/unmarshalling
type picardExtension struct {
	jsoniter.DummyExtension
}

func (extension *picardExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		metadataTag, hasMetadataTag := binding.Field.Tag().Lookup(globalConfig.TagKey)
		if hasMetadataTag {
			options := strings.Split(metadataTag, ",")[1:]
			if mapvalue.StringSliceContainsKey(options, "omitretrieve") {
				// Don't do bindings for tags that have the "omitretrieve" option
				binding.ToNames = []string{}
				binding.FromNames = []string{}
			}
		}
	}
}

func (extension *picardExtension) StructDecodedHook(ptr unsafe.Pointer, typ reflect2.Type) {
	objectValue := reflect.NewAt(typ.Type1(), ptr)
	metadataField := metadata.GetMetadataValue(objectValue.Elem())
	metadata.InitializeDefinedFields(metadataField)
}

func (extension *picardExtension) StructFieldDecodedHook(field string, ptr unsafe.Pointer, typ reflect2.Type) {
	objectValue := reflect.NewAt(typ.Type1(), ptr)
	metadataField := metadata.GetMetadataValue(objectValue.Elem())
	metadata.AddDefinedField(metadataField, field)
}
