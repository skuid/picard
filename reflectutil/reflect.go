/* Package reflectutil contains basic go reflection utility funcs
 */
package reflectutil

import (
	"reflect"
)

// IsZeroValue returns true if the value provided is the zero value for its type
func IsZeroValue(v reflect.Value) bool {
	if v.CanInterface() {
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
	return false
}
