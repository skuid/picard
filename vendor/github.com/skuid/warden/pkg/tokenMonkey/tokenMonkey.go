/*
Package tokenMonkey sets up a visitor pattern for a JSON object, using json.Token
to evaluate the incoming JSON and allows for the user to visit objects along the
way.

Currently it only allows you to visit one key on the TraverseObj function and
it will output all other keys for that object without intervention.

For example, to parse a JSON object like this, but changing the value for each
object in this path:

	models[].data[].*

On this JSON:

	{
		"models": [
			{
				"canRetrieveMoreRecords": false,
				"data": [],
				"fields": [
					{
						"id": "address",
						"objectAlias": "o0",
						"queryId": "o0.address"
					},
					{
						"id": "city_id",
						"objectAlias": "o0",
						"queryId": "o0.city_id"
					}
				],
				"id": "Address",
				"sql": "select \"o0\".\"address\" as \"o0.address\", \"o0\".\"city_id\" as \"o0.city_id\" from \"address\" as \"o0\" where \"o0\".\"address\" = 'brian.newton@skuid.com' limit 2"
			},
			{
				"canRetrieveMoreRecords": false,
				"data": [
					{
						"address_id": 6,
						"first_name": "Patricia",
						"last_name": "Johnson"
					},
					{
						"address_id": 7,
						"first_name": "Linda",
						"last_name": "Williams"
					}
				],
				"fields": [
					{
						"id": "address_id",
						"objectAlias": "o0",
						"queryId": "o0.address_id"
					},
					{
						"id": "first_name",
						"objectAlias": "o0",
						"queryId": "o0.first_name"
					},
					{
						"id": "last_name",
						"objectAlias": "o0",
						"queryId": "o0.last_name"
					}
				],
				"id": "Customer",
				"sql": "select \"o0\".\"address_id\" as \"o0.address_id\", \"o0\".\"first_name\" as \"o0.first_name\", \"o0\".\"last_name\" as \"o0.last_name\" from \"customer\" as \"o0\" limit 2"
			}
		]
	}

You would do something like this:

	// Example of traversing an object, changing values all willy nilly
	json := tm.TraverseObj("models", func() interface{} {
		return tm.TraverseArray(func() interface{} {
			return tm.TraverseObj("data", func() interface{} {
				return tm.TraverseArray(func() interface{} {
					return tm.VisitObj(func(key string, dec *json.Decoder) (string, interface{}) {
						var v interface{}
						if tm.Err = dec.Decode(&v); tm.Err != nil {
							return "", nil
						}
						return key, fmt.Sprintf("Modified %v", v)
					})
				})
			})
		})
	})

	if tm.Err != nil {
		return tm.Err
	}

The variable 'json' will now hold a 'map[string]interface{}' with the changed
values
*/
package tokenMonkey

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

/*
Visitor functions are called whenever you "VisitObj" on an object. The function
will take a key and the value (as a json decoder). It expects a return of the
modified key and value (as interaface{}, who are we to judge?)

For example, you could use this function to change each value to "Modified %v"

	func modifyAValue(key string, dec *json.Decoder) (string, interface{}) {
		var v interface{}
		if tm.Err = dec.Decode(&v); tm.Err != nil {
			return "", nil
		}
		return key, fmt.Sprintf("Modified %v", v)
	}

"NOTE: " a tokenMonkey "JSONer" needs to be able to hold the error
*/
type Visitor func(key string, dec *json.Decoder) (string, interface{})

/*
TokenVisitor is an interface for traversing and visiting objects and arrays in
json
*/
type TokenVisitor interface {
	TraverseObj(string, func() interface{}) interface{}
	TraverseArray(func() interface{}) []interface{}
	VisitObj(Visitor) interface{}
}

/*
JSONer is a terrible name for an object that holds errors, decoder, and encoder
during the visitation of nodes in a JSON object
*/
type JSONer struct {
	Err error
	Dec *json.Decoder
	Enc *json.Encoder
}

/*
TraverseObj will expect the next json token to be an object. If so, it will look
for the key provided, and if it is currently processing that key/property it
will call "next()". Whatever is returned from next() will be the value for that
key.

So if you start with this JSON

	{
		"foo": "bar",
		"baz": "fuz"
	}

And you do this:

	json := tm.TraverseObj("foo", func() interface{} {
		return "I'm changed!"
	})

You'll get an object that looks like this:

	{
		"foo": "I'm changed!",
		"baz": "fuz"
	}

Notice how "baz" was processed but unchanged.

If you want to just dump the whole object, pass an empty string as the key
*/
func (tm *JSONer) TraverseObj(key string, next func() interface{}) interface{} {
	if tm.Err != nil {
		return nil
	}
	var t json.Token
	dec := tm.Dec

	t, tm.Err = dec.Token()
	if tm.Err != nil {
		return nil
	}

	_ /*delim*/, tm.Err = pdelim(t, '{')
	if tm.Err != nil {
		return nil
	}

	// We are in an object. Make sure we've already started the JSON
	obj := make(map[string]interface{})

	// Read props
	for dec.More() {
		t, tm.Err = dec.Token()
		if tm.Err != nil {
			return nil
		}

		prop := t.(string)

		// Test for keys we don't care about here
		if prop != key {
			var v interface{}
			if tm.Err = dec.Decode(&v); tm.Err != nil {
				return nil
			}

			obj[prop] = v
			continue
		}

		obj[prop] = next()
	}

	// Object end
	t, tm.Err = dec.Token()
	if tm.Err != nil {
		return nil
	}

	_ /*delim*/, tm.Err = pdelim(t, '}')
	if tm.Err != nil {
		return nil
	}
	// End object end

	return obj
}

/*
VisitObj allow you to inspect and modify each key/value pair from the current
object. See the Visitor notes above for an example of that
*/
func (tm *JSONer) VisitObj(visit Visitor) interface{} {
	if tm.Err != nil {
		return nil
	}
	var t json.Token
	dec := tm.Dec

	t, tm.Err = dec.Token()
	if tm.Err != nil {
		return nil
	}

	_ /*delim*/, tm.Err = pdelim(t, '{')
	if tm.Err != nil {
		return nil
	}

	obj := make(map[string]interface{})

	// Read props
	for dec.More() {
		t, tm.Err = dec.Token()
		if tm.Err != nil {
			return nil
		}

		prop := t.(string)

		key, val := visit(prop, dec)
		obj[key] = val
	}

	// Object end
	t, tm.Err = dec.Token()
	if tm.Err != nil {
		return nil
	}

	_ /*delim*/, tm.Err = pdelim(t, '}')
	if tm.Err != nil {
		return nil
	}
	// End object end

	return obj
}

/*
TraverseArray will expect the next token to be an array start, and will loop
through everything in the array, calling next() for each and adding the value
returned from each call to next() to the resulting array.
*/
func (tm *JSONer) TraverseArray(next func() interface{}) []interface{} {
	if tm.Err != nil {
		return nil
	}
	var t json.Token
	dec := tm.Dec

	// Array start
	t, tm.Err = dec.Token()
	if tm.Err != nil {
		return nil
	}

	_, tm.Err = pdelim(t, '[')
	if tm.Err != nil {
		return nil
	}
	// End array start

	arr := make([]interface{}, 0)
	// Next each array item
	for dec.More() {
		arr = append(arr, next())
	}

	// Array end
	t, tm.Err = dec.Token()
	if tm.Err != nil {
		return nil
	}

	_, tm.Err = pdelim(t, ']')
	if tm.Err != nil {
		return nil
	}

	return arr
}

func delimToBytes(delim json.Delim) []byte {
	rd := rune(delim)
	buf := make([]byte, utf8.RuneLen(rd))
	utf8.EncodeRune(buf, rd)
	return buf
}

func pdelim(t json.Token, expected json.Delim) (json.Delim, error) {
	delim, ok := t.(json.Delim)
	if !ok || delim != expected {
		return delim, fmt.Errorf("Unexpected delimiter. Got delim: %v", delim)
	}
	return delim, nil
}
