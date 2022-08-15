/*
Package decoder is used to default to jsoniter for its decoder
*/
package decoding

import (
	"reflect"
	"sort"
	"strings"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
	"github.com/skuid/picard/metadata"
	"github.com/skuid/picard/stringutil"
)

// Config specifies options for the picard decoder
type Config struct {
	TagKey string
}

// JsonIter Extension for metadata marshalling/unmarshalling
type picardExtension struct {
	jsoniter.DummyExtension
	config     *Config
	structDesc *jsoniter.StructDescriptor
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
	extension.structDesc = structDescriptor
}

func (extension *picardExtension) DecorateDecoder(typ reflect2.Type, decoder jsoniter.ValDecoder) jsoniter.ValDecoder {
	if typ.Kind() == reflect.Struct {
		return &structDecoder{decoder, typ, extension.structDesc}
	}
	return decoder
}

type structDecoder struct {
	valDecoder jsoniter.ValDecoder
	typ        reflect2.Type
	structDesc *jsoniter.StructDescriptor
}

func (decoder *structDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	var obj interface{}
	iter.ReadVal(&obj)
	if obj != nil {
		// AC: I'm leaving the next line in as a comment because it took me hours of docs-reading to figure out
		// how to get the unsafe.Pointer into something my debugger could read.
		// ip := decoder.typ.UnsafeIndirect(ptr)

		buf, err := jsoniter.Marshal(obj)
		if err != nil {
			return
		}
		// first initialize the defined fields
		objectValue := reflect.NewAt(decoder.typ.Type1(), ptr)
		metadataField := metadata.GetMetadataValue(objectValue.Elem())
		if !metadata.HasDefinedFields(metadataField) {
			metadata.InitializeDefinedFields(metadataField)
		}

		// now get the field keys in the object
		if reflect.TypeOf(obj).Kind() == reflect.Map {
			var fields []string
			fmap := obj.(map[string]interface{})
			for k := range fmap {
				for _, binding := range decoder.structDesc.Fields {
					sf, ok := binding.Field.Tag().Lookup("json")
					if ok && sf == k {
						fields = append(fields, binding.Field.Name())
						break
					}
				}
			}
			sort.Strings(fields) // otherwise test comparison fails because the slice is in a random order. :facepalm:
			for _, f := range fields {
				metadata.AddDefinedField(metadataField, f)
			}
		}

		//  ┬──┬◡ﾉ(° -°ﾉ) newiter because we messed it up with iter.ReadVal()
		newiter := iter.Pool().BorrowIterator(buf)
		defer iter.Pool().ReturnIterator(newiter)

		// now do the normal
		decoder.valDecoder.Decode(ptr, newiter)
	}
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
	api.RegisterExtension(&picardExtension{
		config: config,
	})
	return api
}

//func (extension *picardExtension) StructDecodedHook(ptr unsafe.Pointer, typ reflect2.Type) {
//	objectValue := reflect.NewAt(typ.Type1(), ptr)
//	metadataField := metadata.GetMetadataValue(objectValue.Elem())
//	metadata.InitializeDefinedFields(metadataField)
//}
//
//func (extension *picardExtension) StructFieldDecodedHook(field string, ptr unsafe.Pointer, typ reflect2.Type) {
//	objectValue := reflect.NewAt(typ.Type1(), ptr)
//	metadataField := metadata.GetMetadataValue(objectValue.Elem())
//	metadata.AddDefinedField(metadataField, field)
//}
