package amqptest

// revive:disable

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/url"
	"time"
)

// Assertions is an extracted interface for testify assertions object.
type Assertions interface {
	Condition(comp assert.Comparison, msgAndArgs ...interface{}) bool
	Conditionf(comp assert.Comparison, msg string, args ...interface{}) bool
	Contains(s interface{}, contains interface{}, msgAndArgs ...interface{}) bool
	Containsf(s interface{}, contains interface{}, msg string, args ...interface{}) bool
	DirExists(path string, msgAndArgs ...interface{}) bool
	DirExistsf(path string, msg string, args ...interface{}) bool
	ElementsMatch(listA interface{}, listB interface{}, msgAndArgs ...interface{}) bool
	ElementsMatchf(listA interface{}, listB interface{}, msg string, args ...interface{}) bool
	Empty(object interface{}, msgAndArgs ...interface{}) bool
	Emptyf(object interface{}, msg string, args ...interface{}) bool
	Equal(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool
	EqualError(theError error, errString string, msgAndArgs ...interface{}) bool
	EqualErrorf(theError error, errString string, msg string, args ...interface{}) bool
	EqualValues(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool
	EqualValuesf(expected interface{}, actual interface{}, msg string, args ...interface{}) bool
	Equalf(expected interface{}, actual interface{}, msg string, args ...interface{}) bool
	Error(err error, msgAndArgs ...interface{}) bool
	ErrorAs(err error, target interface{}, msgAndArgs ...interface{}) bool
	ErrorAsf(err error, target interface{}, msg string, args ...interface{}) bool
	ErrorIs(err error, target error, msgAndArgs ...interface{}) bool
	ErrorIsf(err error, target error, msg string, args ...interface{}) bool
	Errorf(err error, msg string, args ...interface{}) bool
	Eventually(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) bool
	Eventuallyf(condition func() bool, waitFor time.Duration, tick time.Duration, msg string, args ...interface{}) bool
	Exactly(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool
	Exactlyf(expected interface{}, actual interface{}, msg string, args ...interface{}) bool
	Fail(failureMessage string, msgAndArgs ...interface{}) bool
	FailNow(failureMessage string, msgAndArgs ...interface{}) bool
	FailNowf(failureMessage string, msg string, args ...interface{}) bool
	Failf(failureMessage string, msg string, args ...interface{}) bool
	False(value bool, msgAndArgs ...interface{}) bool
	Falsef(value bool, msg string, args ...interface{}) bool
	FileExists(path string, msgAndArgs ...interface{}) bool
	FileExistsf(path string, msg string, args ...interface{}) bool
	Greater(e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) bool
	GreaterOrEqual(e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) bool
	GreaterOrEqualf(e1 interface{}, e2 interface{}, msg string, args ...interface{}) bool
	Greaterf(e1 interface{}, e2 interface{}, msg string, args ...interface{}) bool
	HTTPBodyContains(handler http.HandlerFunc, method string, url string, values url.Values, str interface{}, msgAndArgs ...interface{}) bool
	HTTPBodyContainsf(handler http.HandlerFunc, method string, url string, values url.Values, str interface{}, msg string, args ...interface{}) bool
	HTTPBodyNotContains(handler http.HandlerFunc, method string, url string, values url.Values, str interface{}, msgAndArgs ...interface{}) bool
	HTTPBodyNotContainsf(handler http.HandlerFunc, method string, url string, values url.Values, str interface{}, msg string, args ...interface{}) bool
	HTTPError(handler http.HandlerFunc, method string, url string, values url.Values, msgAndArgs ...interface{}) bool
	HTTPErrorf(handler http.HandlerFunc, method string, url string, values url.Values, msg string, args ...interface{}) bool
	HTTPRedirect(handler http.HandlerFunc, method string, url string, values url.Values, msgAndArgs ...interface{}) bool
	HTTPRedirectf(handler http.HandlerFunc, method string, url string, values url.Values, msg string, args ...interface{}) bool
	HTTPStatusCode(handler http.HandlerFunc, method string, url string, values url.Values, statuscode int, msgAndArgs ...interface{}) bool
	HTTPStatusCodef(handler http.HandlerFunc, method string, url string, values url.Values, statuscode int, msg string, args ...interface{}) bool
	HTTPSuccess(handler http.HandlerFunc, method string, url string, values url.Values, msgAndArgs ...interface{}) bool
	HTTPSuccessf(handler http.HandlerFunc, method string, url string, values url.Values, msg string, args ...interface{}) bool
	Implements(interfaceObject interface{}, object interface{}, msgAndArgs ...interface{}) bool
	Implementsf(interfaceObject interface{}, object interface{}, msg string, args ...interface{}) bool
	InDelta(expected interface{}, actual interface{}, delta float64, msgAndArgs ...interface{}) bool
	InDeltaMapValues(expected interface{}, actual interface{}, delta float64, msgAndArgs ...interface{}) bool
	InDeltaMapValuesf(expected interface{}, actual interface{}, delta float64, msg string, args ...interface{}) bool
	InDeltaSlice(expected interface{}, actual interface{}, delta float64, msgAndArgs ...interface{}) bool
	InDeltaSlicef(expected interface{}, actual interface{}, delta float64, msg string, args ...interface{}) bool
	InDeltaf(expected interface{}, actual interface{}, delta float64, msg string, args ...interface{}) bool
	InEpsilon(expected interface{}, actual interface{}, epsilon float64, msgAndArgs ...interface{}) bool
	InEpsilonSlice(expected interface{}, actual interface{}, epsilon float64, msgAndArgs ...interface{}) bool
	InEpsilonSlicef(expected interface{}, actual interface{}, epsilon float64, msg string, args ...interface{}) bool
	InEpsilonf(expected interface{}, actual interface{}, epsilon float64, msg string, args ...interface{}) bool
	IsDecreasing(object interface{}, msgAndArgs ...interface{}) bool
	IsDecreasingf(object interface{}, msg string, args ...interface{}) bool
	IsIncreasing(object interface{}, msgAndArgs ...interface{}) bool
	IsIncreasingf(object interface{}, msg string, args ...interface{}) bool
	IsNonDecreasing(object interface{}, msgAndArgs ...interface{}) bool
	IsNonDecreasingf(object interface{}, msg string, args ...interface{}) bool
	IsNonIncreasing(object interface{}, msgAndArgs ...interface{}) bool
	IsNonIncreasingf(object interface{}, msg string, args ...interface{}) bool
	IsType(expectedType interface{}, object interface{}, msgAndArgs ...interface{}) bool
	IsTypef(expectedType interface{}, object interface{}, msg string, args ...interface{}) bool
	JSONEq(expected string, actual string, msgAndArgs ...interface{}) bool
	JSONEqf(expected string, actual string, msg string, args ...interface{}) bool
	Len(object interface{}, length int, msgAndArgs ...interface{}) bool
	Lenf(object interface{}, length int, msg string, args ...interface{}) bool
	Less(e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) bool
	LessOrEqual(e1 interface{}, e2 interface{}, msgAndArgs ...interface{}) bool
	LessOrEqualf(e1 interface{}, e2 interface{}, msg string, args ...interface{}) bool
	Lessf(e1 interface{}, e2 interface{}, msg string, args ...interface{}) bool
	Negative(e interface{}, msgAndArgs ...interface{}) bool
	Negativef(e interface{}, msg string, args ...interface{}) bool
	Never(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) bool
	Neverf(condition func() bool, waitFor time.Duration, tick time.Duration, msg string, args ...interface{}) bool
	Nil(object interface{}, msgAndArgs ...interface{}) bool
	Nilf(object interface{}, msg string, args ...interface{}) bool
	NoDirExists(path string, msgAndArgs ...interface{}) bool
	NoDirExistsf(path string, msg string, args ...interface{}) bool
	NoError(err error, msgAndArgs ...interface{}) bool
	NoErrorf(err error, msg string, args ...interface{}) bool
	NoFileExists(path string, msgAndArgs ...interface{}) bool
	NoFileExistsf(path string, msg string, args ...interface{}) bool
	NotContains(s interface{}, contains interface{}, msgAndArgs ...interface{}) bool
	NotContainsf(s interface{}, contains interface{}, msg string, args ...interface{}) bool
	NotEmpty(object interface{}, msgAndArgs ...interface{}) bool
	NotEmptyf(object interface{}, msg string, args ...interface{}) bool
	NotEqual(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool
	NotEqualValues(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool
	NotEqualValuesf(expected interface{}, actual interface{}, msg string, args ...interface{}) bool
	NotEqualf(expected interface{}, actual interface{}, msg string, args ...interface{}) bool
	NotErrorIs(err error, target error, msgAndArgs ...interface{}) bool
	NotErrorIsf(err error, target error, msg string, args ...interface{}) bool
	NotNil(object interface{}, msgAndArgs ...interface{}) bool
	NotNilf(object interface{}, msg string, args ...interface{}) bool
	NotPanics(f assert.PanicTestFunc, msgAndArgs ...interface{}) bool
	NotPanicsf(f assert.PanicTestFunc, msg string, args ...interface{}) bool
	NotRegexp(rx interface{}, str interface{}, msgAndArgs ...interface{}) bool
	NotRegexpf(rx interface{}, str interface{}, msg string, args ...interface{}) bool
	NotSame(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool
	NotSamef(expected interface{}, actual interface{}, msg string, args ...interface{}) bool
	NotSubset(list interface{}, subset interface{}, msgAndArgs ...interface{}) bool
	NotSubsetf(list interface{}, subset interface{}, msg string, args ...interface{}) bool
	NotZero(i interface{}, msgAndArgs ...interface{}) bool
	NotZerof(i interface{}, msg string, args ...interface{}) bool
	Panics(f assert.PanicTestFunc, msgAndArgs ...interface{}) bool
	PanicsWithError(errString string, f assert.PanicTestFunc, msgAndArgs ...interface{}) bool
	PanicsWithErrorf(errString string, f assert.PanicTestFunc, msg string, args ...interface{}) bool
	PanicsWithValue(expected interface{}, f assert.PanicTestFunc, msgAndArgs ...interface{}) bool
	PanicsWithValuef(expected interface{}, f assert.PanicTestFunc, msg string, args ...interface{}) bool
	Panicsf(f assert.PanicTestFunc, msg string, args ...interface{}) bool
	Positive(e interface{}, msgAndArgs ...interface{}) bool
	Positivef(e interface{}, msg string, args ...interface{}) bool
	Regexp(rx interface{}, str interface{}, msgAndArgs ...interface{}) bool
	Regexpf(rx interface{}, str interface{}, msg string, args ...interface{}) bool
	Same(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool
	Samef(expected interface{}, actual interface{}, msg string, args ...interface{}) bool
	Subset(list interface{}, subset interface{}, msgAndArgs ...interface{}) bool
	Subsetf(list interface{}, subset interface{}, msg string, args ...interface{}) bool
	True(value bool, msgAndArgs ...interface{}) bool
	Truef(value bool, msg string, args ...interface{}) bool
	WithinDuration(expected time.Time, actual time.Time, delta time.Duration, msgAndArgs ...interface{}) bool
	WithinDurationf(expected time.Time, actual time.Time, delta time.Duration, msg string, args ...interface{}) bool
	YAMLEq(expected string, actual string, msgAndArgs ...interface{}) bool
	YAMLEqf(expected string, actual string, msg string, args ...interface{}) bool
	Zero(i interface{}, msgAndArgs ...interface{}) bool
	Zerof(i interface{}, msg string, args ...interface{}) bool
}
