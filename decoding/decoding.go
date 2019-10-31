/*
Package decoder is used to default to jsoniter for its decoder
*/
package decoding

import (
	"reflect"
	"strings"
	"unsafe"

	"github.com/modern-go/reflect2"
	jsoniter "github.com/plusplusben/json-iterator-go"
	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/stringutil"
)

// Config specifies options for the picard decoder
type Config struct {
	TagKey string
}

// GetDecoder returns a decoder that implements the standard encoding/json api
func GetDecoder(config *Config) jsoniter.API {
	if config == nil {
		config = &Config{}
	}
	if config.TagKey == "" {
		config.TagKey = "json"
	}
	api := jsoniter.Config{
		EscapeHTML:             true,
		SortMapKeys:            true,
		ValidateJsonRawMessage: true,
		OnlyTaggedField:        true,
		TagKey:                 config.TagKey,
	}.Froze()
	api.RegisterExtension(&picardExtension{jsoniter.DummyExtension{}, config})
	return api
}

// JsonIter Extension for metadata marshalling/unmarshalling
type picardExtension struct {
	jsoniter.DummyExtension
	config *Config
}

func (extension *picardExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		metadataTag, hasMetadataTag := binding.Field.Tag().Lookup(extension.config.TagKey)
		if hasMetadataTag {
			options := strings.Split(metadataTag, ",")[1:]
			if stringutil.StringSliceContainsKey(options, "omitretrieve") {
				// Don't do bindings for tags that have the "omitretrieve" option
				binding.ToNames = []string{}
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
