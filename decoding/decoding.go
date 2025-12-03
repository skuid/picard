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
	TagKey    string
	OmitLogic func(*jsoniter.Binding) bool
}

// JsonIter Extension for metadata marshalling/unmarshalling
type picardExtension struct {
	jsoniter.DummyExtension
	config     *Config
	structDesc *jsoniter.StructDescriptor
}

func (extension *picardExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		if extension.config.OmitLogic != nil && extension.config.OmitLogic(binding) {
			binding.ToNames = []string{}
		}
	}
	extension.structDesc = structDescriptor
}

func (extension *picardExtension) DecorateDecoder(typ reflect2.Type, decoder jsoniter.ValDecoder) jsoniter.ValDecoder {
	if typ.Kind() == reflect.Struct {
		return &structDecoder{decoder, typ, extension.structDesc, extension.config}
	}
	return decoder
}

type structDecoder struct {
	valDecoder jsoniter.ValDecoder
	typ        reflect2.Type
	structDesc *jsoniter.StructDescriptor
	config     *Config
}

func (decoder *structDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	var obj any
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
		if reflect.TypeOf(obj).Kind() == reflect.Map && decoder.structDesc != nil {
			var fields []string
			fmap := obj.(map[string]any)
			for k := range fmap {
				for _, binding := range decoder.structDesc.Fields {
					sf, ok := binding.Field.Tag().Lookup(decoder.config.TagKey)
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
	if config.OmitLogic == nil {
		config.OmitLogic = func(binding *jsoniter.Binding) bool {
			metadataTag, hasMetadataTag := binding.Field.Tag().Lookup(config.TagKey)
			if hasMetadataTag {
				options := strings.Split(metadataTag, ",")[1:]
				return stringutil.StringSliceContainsKey(options, "omitretrieve")
			}
			return false
		}
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
