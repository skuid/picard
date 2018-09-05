package jsoniter

import (
	"fmt"
	"io"
	"strings"
	"unsafe"

	"github.com/modern-go/reflect2"
)

func decoderOfStruct(ctx *ctx, typ reflect2.Type) ValDecoder {
	bindings := map[string]*Binding{}
	structDescriptor := describeStruct(ctx, typ)
	for _, binding := range structDescriptor.Fields {
		for _, fromName := range binding.FromNames {
			old := bindings[fromName]
			if old == nil {
				bindings[fromName] = binding
				continue
			}
			ignoreOld, ignoreNew := resolveConflictBinding(ctx.frozenConfig, old, binding)
			if ignoreOld {
				delete(bindings, fromName)
			}
			if !ignoreNew {
				bindings[fromName] = binding
			}
		}
	}
	fields := map[string]*structFieldDecoder{}
	for k, binding := range bindings {
		fields[k] = binding.Decoder.(*structFieldDecoder)
	}

	if !ctx.caseSensitive() {
		for k, binding := range bindings {
			if _, found := fields[strings.ToLower(k)]; !found {
				fields[strings.ToLower(k)] = binding.Decoder.(*structFieldDecoder)
			}
		}
	}

	return createStructDecoder(ctx, typ, fields)
}

func createStructDecoder(ctx *ctx, typ reflect2.Type, fields map[string]*structFieldDecoder) ValDecoder {
	if ctx.disallowUnknownFields {
		return &generalStructDecoder{typ: typ, fields: fields, disallowUnknownFields: true}
	}
	switch len(fields) {
	case 0:
		return &skipObjectDecoder{typ}
	}
	return &generalStructDecoder{typ, fields, false}
}

type generalStructDecoder struct {
	typ                   reflect2.Type
	fields                map[string]*structFieldDecoder
	disallowUnknownFields bool
}

func (decoder *generalStructDecoder) Decode(ptr unsafe.Pointer, iter *Iterator) {
	for _, extension := range extensions {
		extension.StructDecodedHook(ptr, decoder.typ)
	}
	for _, extension := range iter.cfg.extraExtensions {
		extension.StructDecodedHook(ptr, decoder.typ)
	}
	if !iter.readObjectStart() {
		return
	}
	var c byte
	for c = ','; c == ','; c = iter.nextToken() {
		decoder.decodeOneField(ptr, iter)
	}
	if iter.Error != nil && iter.Error != io.EOF {
		iter.Error = fmt.Errorf("%v.%s", decoder.typ, iter.Error.Error())
	}
	if c != '}' {
		iter.ReportError("struct Decode", `expect }, but found `+string([]byte{c}))
	}
}

func (decoder *generalStructDecoder) decodeOneField(ptr unsafe.Pointer, iter *Iterator) {
	var field string
	var fieldDecoder *structFieldDecoder
	if iter.cfg.objectFieldMustBeSimpleString {
		fieldBytes := iter.ReadStringAsSlice()
		field = *(*string)(unsafe.Pointer(&fieldBytes))
		fieldDecoder = decoder.fields[field]
		if fieldDecoder == nil && !iter.cfg.caseSensitive {
			fieldDecoder = decoder.fields[strings.ToLower(field)]
		}
	} else {
		field = iter.ReadString()
		fieldDecoder = decoder.fields[field]
		if fieldDecoder == nil && !iter.cfg.caseSensitive {
			fieldDecoder = decoder.fields[strings.ToLower(field)]
		}
	}
	if fieldDecoder == nil {
		msg := "found unknown field: " + field
		if decoder.disallowUnknownFields {
			iter.ReportError("ReadObject", msg)
		}
		c := iter.nextToken()
		if c != ':' {
			iter.ReportError("ReadObject", "expect : after object field, but found "+string([]byte{c}))
		}
		iter.Skip()
		return
	}
	c := iter.nextToken()
	if c != ':' {
		iter.ReportError("ReadObject", "expect : after object field, but found "+string([]byte{c}))
	}
	for _, extension := range extensions {
		extension.StructFieldDecodedHook(fieldDecoder.field.Name(), ptr, decoder.typ)
	}
	for _, extension := range iter.cfg.extraExtensions {
		extension.StructFieldDecodedHook(fieldDecoder.field.Name(), ptr, decoder.typ)
	}

	fieldDecoder.Decode(ptr, iter)
}

type skipObjectDecoder struct {
	typ reflect2.Type
}

func (decoder *skipObjectDecoder) Decode(ptr unsafe.Pointer, iter *Iterator) {
	for _, extension := range extensions {
		extension.StructDecodedHook(ptr, decoder.typ)
	}
	for _, extension := range iter.cfg.extraExtensions {
		extension.StructDecodedHook(ptr, decoder.typ)
	}
	valueType := iter.WhatIsNext()
	if valueType != ObjectValue && valueType != NilValue {
		iter.ReportError("skipObjectDecoder", "expect object or null")
		return
	}
	iter.Skip()
}

type structFieldDecoder struct {
	field        reflect2.StructField
	fieldDecoder ValDecoder
}

func (decoder *structFieldDecoder) Decode(ptr unsafe.Pointer, iter *Iterator) {
	fieldPtr := decoder.field.UnsafeGet(ptr)
	decoder.fieldDecoder.Decode(fieldPtr, iter)
	if iter.Error != nil && iter.Error != io.EOF {
		iter.Error = fmt.Errorf("%s: %s", decoder.field.Name(), iter.Error.Error())
	}
}

type stringModeStringDecoder struct {
	elemDecoder ValDecoder
	cfg         *frozenConfig
}

func (decoder *stringModeStringDecoder) Decode(ptr unsafe.Pointer, iter *Iterator) {
	decoder.elemDecoder.Decode(ptr, iter)
	str := *((*string)(ptr))
	tempIter := decoder.cfg.BorrowIterator([]byte(str))
	defer decoder.cfg.ReturnIterator(tempIter)
	*((*string)(ptr)) = tempIter.ReadString()
}

type stringModeNumberDecoder struct {
	elemDecoder ValDecoder
}

func (decoder *stringModeNumberDecoder) Decode(ptr unsafe.Pointer, iter *Iterator) {
	c := iter.nextToken()
	if c != '"' {
		iter.ReportError("stringModeNumberDecoder", `expect ", but found `+string([]byte{c}))
		return
	}
	decoder.elemDecoder.Decode(ptr, iter)
	if iter.Error != nil {
		return
	}
	c = iter.readByte()
	if c != '"' {
		iter.ReportError("stringModeNumberDecoder", `expect ", but found `+string([]byte{c}))
		return
	}
}
