/*
Package decoder is used to default to jsoniter for its decoder
*/
package decoding

import (
	"fmt"
	"reflect"
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
		// first initialize the defined fields
		objectValue := reflect.NewAt(decoder.typ.Type1(), ptr)
		metadataField := metadata.GetMetadataValue(objectValue.Elem())
		if !metadata.HasDefinedFields(metadataField) {
			metadata.InitializeDefinedFields(metadataField)
		}

		// now get the field keys in the object
		if reflect.TypeOf(obj).Kind() == reflect.Map {
			fmap := obj.(map[string]interface{})
			for k, _ := range fmap {
				for _, binding := range decoder.structDesc.Fields {
					sf, ok := binding.Field.Tag().Lookup("json")
					if ok && sf == k {
						metadata.AddDefinedField(metadataField, binding.Field.Name())
						break
					}
				}
			}
		}

		//  ┬──┬◡ﾉ(° -°ﾉ)
		buf, err := jsoniter.Marshal(obj)
		if err != nil {
			return
		}

		iter.ResetBytes(buf)

		// now do the normal
		decoder.valDecoder.Decode(ptr, iter)
		ip := decoder.typ.UnsafeIndirect(ptr)
		fmt.Printf("%v\n", ip)
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

//func (extension *picardExtension) DecorateDecoder(typ reflect2.Type, decoder jsoniter.ValDecoder) jsoniter.ValDecoder {
//	dec := &metadataDecoder{typ, decoder, extension}
//	extension.decoder = dec
//	return dec
//}
//
//type metadataDecoder struct {
//	typ        reflect2.Type
//	valDecoder jsoniter.ValDecoder
//	parent     *picardExtension
//}
//
//func (decoder *metadataDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
//	objectValue := reflect.NewAt(decoder.typ.Type1(), ptr)
//	metadataField := metadata.GetMetadataValue(objectValue.Elem())
//	k := metadataField.Kind()
//	if k == reflect.Struct {
//		if !metadata.HasDefinedFields(metadataField) {
//			metadata.InitializeDefinedFields(metadataField)
//		}
//		decoder.parent.currentVal = &metadataField
//	}
//	decoder.valDecoder.Decode(ptr, iter)
//}

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
