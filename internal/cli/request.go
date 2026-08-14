package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// requestObject is the whole of what a caller may put on stdin:
//
//	{"system": "…who the reviewer is…", "task": "…what to review, this time…"}
//
// It is the cross-repository contract — the lx skills repository's
// `review-interfaces.md` builds exactly this shape — so the two field names
// are a one-way door in a way the Go type around them is not. The object is
// extensible by design: something that needs passing later gets a field here,
// not a flag (specs 08).
//
// The fields are plain strings rather than *string. A field's absence and a
// field's emptiness are the same refusal for the same reason — the binary has
// no default system prompt to fall back to — so nothing downstream would ever
// read the distinction a pointer preserves.
type requestObject struct {
	System string `json:"system"`
	Task   string `json:"task"`
}

// errNotARequestObject is what every unreadable stdin body reduces to. The
// decoder's own sentence is appended to it where it names something the
// caller can act on (an unknown field, a type mismatch), but the classifying
// question — "was this a request object at all?" — is answered by this
// sentinel and never by matching message text (CONVENTIONS § Error handling).
var errNotARequestObject = errors.New("stdin did not carry a JSON request object")

// decodeRequestObject parses the request object out of body and validates
// both of its fields, returning the message the single error: line carries.
//
// Three refusals, each of them the fail-closed reading of a state that would
// otherwise look like it worked:
//
//   - DisallowUnknownFields, so `{"systm":…}` is an error rather than a
//     request with no system prompt at all. This is specs 08's own wording,
//     and the reason a Go struct is decoded rather than a map.
//   - Anything after the object — a second object, trailing junk — is
//     refused. A caller that piped two requests would otherwise have the
//     second silently discarded, having been told nothing went wrong.
//   - Either field empty or whitespace-only is exit 2 and no substitution.
//     A default system prompt supplied here would be the binary owning the
//     prompt again through the back door, which is the one outcome specs 08
//     turned the recommendation down over.
//
// strings.TrimSpace decides emptiness only: what is returned — and what
// reaches the model — is the caller's own text, untouched.
func decodeRequestObject(body string) (requestObject, error) {
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()

	var request requestObject
	if err := decoder.Decode(&request); err != nil {
		return requestObject{}, fmt.Errorf("%w: %s", errNotARequestObject, decodeDetail(err))
	}
	// io.EOF is the only acceptable answer to "what follows the object?".
	// Decode into a json.RawMessage rather than checking decoder.More(),
	// because More reports false for trailing bytes that are not the start of
	// a value while Decode reports them as the syntax error they are.
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return requestObject{}, fmt.Errorf("%w: it carries more than one value, and nothing after the request object is read", errNotARequestObject)
	}

	if strings.TrimSpace(request.System) == "" {
		return requestObject{}, errors.New(`the request object's "system" is empty: the caller owns the system prompt and this binary substitutes none`)
	}
	if strings.TrimSpace(request.Task) == "" {
		return requestObject{}, errors.New(`the request object's "task" is empty: there is nothing to review`)
	}
	return request, nil
}

// expectedShape is what every diagnostic that cannot name a specific field
// says instead, so a caller piping the pre-request-object shape (raw task
// text) is told what to send rather than at which byte offset the parse died.
const expectedShape = `expected {"system": "…", "task": "…"}`

// unknownFieldPrefix is how encoding/json opens its unknown-field message.
// It exports no error type for that case — the value is a bare errorString —
// so naming the offending key means reading the rendered text. This is a
// *rendering* decision and never a classifying one: the branch below only
// chooses the wording, and a prefix that changes under a future Go release
// falls through to the decoder's own sentence, which still names the key the
// acceptance criteria require on the error line.
const unknownFieldPrefix = "json: unknown field "

// decodeDetail renders what encoding/json objected to, naming the caller's
// own key wherever the decoder knew one and never leaking a Go type name onto
// stderr.
func decodeDetail(err error) string {
	if name, found := strings.CutPrefix(err.Error(), unknownFieldPrefix); found {
		return fmt.Sprintf(`it carries an unknown field %s, and the fields are "system" and "task"`, name)
	}
	var mismatch *json.UnmarshalTypeError
	if errors.As(err, &mismatch) {
		if mismatch.Field == "" {
			return expectedShape
		}
		return fmt.Sprintf("its %q is a %s, and both fields are strings", mismatch.Field, mismatch.Value)
	}
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return expectedShape
	}
	return err.Error()
}
