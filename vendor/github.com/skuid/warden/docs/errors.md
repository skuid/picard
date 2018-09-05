## Warden Errors

In Warden, we whitelist errors that should be monitored. We do this by wrapping errors with additional context, like attributes and error class.

Not all 500 errors should be wrapped, only the ones that should notify a monitoring platform about error conditions in Warden's domain of responsibility. It is best to keep the errors opaque by wrapping them as soon as they happen.

## usage

errors.WrapError creates a new newrelic.Error type using the previous error message, attributes, and class.

### example

  if err != nil {
    errors.WrapError(
	    err,
	    errors.PicardClass,
	    map[string]interface{}{
	      "action": "FilterModel",
          "contextKey":  picardORMContextKey,
	    },
	    "",
    )
  }

When an error is passed through its callers and is ready to be handled, a type assertion is made. New Relic will then be notified through a web transaction.
