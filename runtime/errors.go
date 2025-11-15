package cbor

import (
	"errors"
	"reflect"
	"strconv"
)

const resumableDefault = false

var (
	// ErrShortBytes is returned when the
	// slice being decoded is too short to
	// contain the contents of the message
	ErrShortBytes error = errShort{}

	// ErrRecursion is returned when the maximum recursion limit is reached for an operation.
	// This should only realistically be seen on adversarial data trying to exhaust the stack.
	ErrRecursion error = errRecursion{}

	// ErrLimitExceeded is returned when a set limit is exceeded.
	// Limits can be set on the Reader to prevent excessive memory usage by adversarial data.
	ErrLimitExceeded error = errLimitExceeded{}

	// ErrMaxDepthExceeded is returned when skip recursion depth exceeds limit
	ErrMaxDepthExceeded error = errors.New("cbor: max depth exceeded")

	// ErrNotNil is returned when expecting nil
	ErrNotNil error = errors.New("cbor: not nil")

	// ErrInvalidUTF8 is returned when a text string contains invalid UTF-8
	ErrInvalidUTF8 error = errors.New("cbor: invalid UTF-8 in text string")

	// ErrDuplicateMapKey is returned when a map contains duplicate keys where
	// duplicates are not allowed (e.g., deterministic/strict decoding).
	ErrDuplicateMapKey error = errors.New("cbor: duplicate map key")

	// ErrIndefiniteForbidden is returned when an indefinite-length item is present
	// but strict/deterministic decoding forbids it.
	ErrIndefiniteForbidden error = errors.New("cbor: indefinite-length item not allowed in strict/deterministic mode")

	// ErrNonCanonicalInteger is returned when an integer is not encoded in the shortest form.
	ErrNonCanonicalInteger error = errors.New("cbor: non-canonical integer encoding")

	// ErrNonCanonicalLength is returned when a length (array/map/str/bytes) is not encoded in the shortest form.
	ErrNonCanonicalLength error = errors.New("cbor: non-canonical length encoding")

)

// Error is the interface satisfied
// by all of the errors that originate
// from this package.
type Error interface {
	error

	// Resumable returns whether
	// or not the error means that
	// the stream of data is malformed
	// and the information is unrecoverable.
	Resumable() bool
}

// contextError allows msgp Error instances to be enhanced with additional
// context about their origin.
type contextError interface {
	Error

	// withContext must not modify the error instance - it must clone and
	// return a new error with the context added.
	withContext(ctx string) error
}

// Cause returns the underlying cause of an error that has been wrapped
// with additional context.
func Cause(e error) error {
	out := e
	if e, ok := e.(errWrapped); ok && e.cause != nil {
		out = e.cause
	}
	return out
}

// Resumable returns whether or not the error means that the stream of data is
// malformed and the information is unrecoverable.
func Resumable(e error) bool {
	if e, ok := e.(Error); ok {
		return e.Resumable()
	}
	return resumableDefault
}

// WrapError wraps an error with additional context that allows the part of the
// serialized type that caused the problem to be identified. Underlying errors
// can be retrieved using Cause()
//
// The input error is not modified - a new error should be returned.
//
// ErrShortBytes is not wrapped with any context due to backward compatibility
// issues with the public API.
func WrapError(err error, ctx ...any) error {
	switch e := err.(type) {
	case errShort:
		return e
	case contextError:
		return e.withContext(ctxString(ctx))
	default:
		return errWrapped{cause: err, ctx: ctxString(ctx)}
	}
}

func addCtx(ctx, add string) string {
	if ctx != "" {
		return add + "/" + ctx
	} else {
		return add
	}
}

// errWrapped allows arbitrary errors passed to WrapError to be enhanced with
// context and unwrapped with Cause()
type errWrapped struct {
	cause error
	ctx   string
}

func (e errWrapped) Error() string {
	if e.ctx != "" {
		return e.cause.Error() + " at " + e.ctx
	} else {
		return e.cause.Error()
	}
}

func (e errWrapped) Resumable() bool {
	if e, ok := e.cause.(Error); ok {
		return e.Resumable()
	}
	return resumableDefault
}

// Unwrap returns the cause.
func (e errWrapped) Unwrap() error { return e.cause }

type errShort struct{}

func (e errShort) Error() string   { return "cbor: too few bytes left to read object" }
func (e errShort) Resumable() bool { return false }

type errRecursion struct{}

func (e errRecursion) Error() string   { return "cbor: recursion limit reached" }
func (e errRecursion) Resumable() bool { return false }

type errLimitExceeded struct{}

func (e errLimitExceeded) Error() string   { return "cbor: configured reader limit exceeded" }
func (e errLimitExceeded) Resumable() bool { return false }

// ArrayError is an error returned
// when decoding a fix-sized array
// of the wrong size
type ArrayError struct {
	Wanted uint32
	Got    uint32
	ctx    string
}

// Error implements the error interface
func (a ArrayError) Error() string {
	out := "cbor: wanted array of size " + strconv.Itoa(int(a.Wanted)) + "; got " + strconv.Itoa(int(a.Got))
	if a.ctx != "" {
		out += " at " + a.ctx
	}
	return out
}

// Resumable is always 'true' for ArrayErrors
func (a ArrayError) Resumable() bool { return true }

func (a ArrayError) withContext(ctx string) error { a.ctx = addCtx(a.ctx, ctx); return a }

// IntOverflow is returned when a call
// would downcast an integer to a type
// with too few bits to hold its value.
type IntOverflow struct {
	Value         int64 // the value of the integer
	FailedBitsize int   // the bit size that the int64 could not fit into
	ctx           string
}

// Error implements the error interface
func (i IntOverflow) Error() string {
	str := "cbor: " + strconv.FormatInt(i.Value, 10) + " overflows int" + strconv.Itoa(i.FailedBitsize)
	if i.ctx != "" {
		str += " at " + i.ctx
	}
	return str
}

// Resumable is always 'true' for overflows
func (i IntOverflow) Resumable() bool { return true }

func (i IntOverflow) withContext(ctx string) error { i.ctx = addCtx(i.ctx, ctx); return i }

// UintOverflow is returned when a call
// would downcast an unsigned integer to a type
// with too few bits to hold its value
type UintOverflow struct {
	Value         uint64 // value of the uint
	FailedBitsize int    // the bit size that couldn't fit the value
	ctx           string
}

// Error implements the error interface
func (u UintOverflow) Error() string {
	str := "cbor: " + strconv.FormatUint(u.Value, 10) + " overflows uint" + strconv.Itoa(u.FailedBitsize)
	if u.ctx != "" {
		str += " at " + u.ctx
	}
	return str
}

// Resumable is always 'true' for overflows
func (u UintOverflow) Resumable() bool { return true }

func (u UintOverflow) withContext(ctx string) error { u.ctx = addCtx(u.ctx, ctx); return u }

// InvalidTimestamp is returned when an invalid timestamp is encountered
type InvalidTimestamp struct {
	Nanos       int64 // value of the nano, if invalid
	FieldLength int   // Unexpected field length.
	ctx         string
}

// Error implements the error interface
func (u InvalidTimestamp) Error() (str string) {
	if u.Nanos > 0 {
		str = "msgp: timestamp nanosecond field value " + strconv.FormatInt(u.Nanos, 10) + " exceeds maximum allows of 999999999"
	} else if u.FieldLength >= 0 {
		str = "msgp: invalid timestamp field length " + strconv.FormatInt(int64(u.FieldLength), 10) + " - must be 4, 8 or 12"
	}
	if u.ctx != "" {
		str += " at " + u.ctx
	}
	return str
}

// Resumable is always 'true' for overflows
func (u InvalidTimestamp) Resumable() bool { return true }

func (u InvalidTimestamp) withContext(ctx string) error { u.ctx = addCtx(u.ctx, ctx); return u }

// UintBelowZero is returned when a call
// would cast a signed integer below zero
// to an unsigned integer.
type UintBelowZero struct {
	Value int64 // value of the incoming int
	ctx   string
}

// Error implements the error interface
func (u UintBelowZero) Error() string {
	str := "cbor: attempted to cast int " + strconv.FormatInt(u.Value, 10) + " to unsigned"
	if u.ctx != "" {
		str += " at " + u.ctx
	}
	return str
}

// Resumable is always 'true' for overflows
func (u UintBelowZero) Resumable() bool { return true }

func (u UintBelowZero) withContext(ctx string) error {
	u.ctx = ctx
	return u
}

// A TypeError is returned when a particular
// decoding method is unsuitable for decoding
// a particular MessagePack value.
type TypeError struct {
	Method  Type // Type expected by method
	Encoded Type // Type actually encoded

	ctx string
}

// Error implements the error interface
func (t TypeError) Error() string {
	out := "cbor: attempted to decode type " + quoteStr(t.Encoded.String()) + " with method for " + quoteStr(t.Method.String())
	if t.ctx != "" {
		out += " at " + t.ctx
	}
	return out
}

// Resumable returns 'true' for TypeErrors
func (t TypeError) Resumable() bool { return true }

func (t TypeError) withContext(ctx string) error { t.ctx = addCtx(t.ctx, ctx); return t }

// returns either InvalidPrefixError or
// TypeError depending on whether or not
// the prefix is recognized
func badPrefix(wantMajor uint8, gotMajor uint8) error {
	return InvalidPrefixError{Want: wantMajor, Got: gotMajor}
}

// InvalidPrefixError is returned when a bad encoding
// uses a major type that is not expected.
// This kind of error is unrecoverable.
type InvalidPrefixError struct {
	Want uint8
	Got  uint8
}

// Error implements the error interface
func (i InvalidPrefixError) Error() string {
	return "cbor: expected major type " + strconv.Itoa(int(i.Want)) + " but got " + strconv.Itoa(int(i.Got))
}

// Resumable returns 'false' for InvalidPrefixErrors
func (i InvalidPrefixError) Resumable() bool { return false }

// ErrUnsupportedType is returned when a bad argument is supplied to
// a function that accepts arbitrary values.
type ErrUnsupportedType struct {
	T reflect.Type

	ctx string
}

// Error implements error
func (e *ErrUnsupportedType) Error() string {
	out := "cbor: type " + quoteStr(e.T.String()) + " not supported"
	if e.ctx != "" {
		out += " at " + e.ctx
	}
	return out
}

// Resumable returns 'true' for ErrUnsupportedType
func (e *ErrUnsupportedType) Resumable() bool { return true }

func (e *ErrUnsupportedType) withContext(ctx string) error {
	o := *e
	o.ctx = addCtx(o.ctx, ctx)
	return &o
}
