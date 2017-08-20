// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/valpar source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package valpar

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"aahframework.org/test.v0/assert"
)

type testStruct struct{}
type testStruct1 struct{}

func TestValueParser(t *testing.T) {
	// check nil
	err := AddValueParser(nil, nil)
	assert.NotNil(t, err)
	assert.True(t, err == ErrTypeOrParserIsNil)

	// check already exists
	err = AddValueParser(timeType, func(key string, typ reflect.Type, params url.Values) (reflect.Value, error) {
		return reflect.Value{}, nil
	})
	assert.NotNil(t, err)
	assert.True(t, err == ErrValueParserIsAlreadyExists)

	// adding new
	err = AddValueParser(reflect.TypeOf(testStruct{}), func(key string, typ reflect.Type, params url.Values) (reflect.Value, error) {
		return reflect.Value{}, nil
	})
	assert.Nil(t, err)

	// getting value parser
	parser1, found1 := ValueParser(timeType)
	assert.True(t, found1)
	assert.NotNil(t, parser1)

	parser2, found2 := ValueParser(reflect.TypeOf(&time.Time{}))
	assert.True(t, found2)
	assert.NotNil(t, parser2)
	assert.Equal(t, fmt.Sprintf("%v", parser1), fmt.Sprintf("%v", parser2))

	parser, found := ValueParser(reflect.TypeOf((*int)(nil)))
	assert.True(t, found)
	assert.NotNil(t, parser)

	parser, found = ValueParser(reflect.TypeOf(testStruct1{}))
	assert.False(t, found)
	assert.Nil(t, parser)
}

type sampleInfo struct {
	XMLName   xml.Name `xml:"Submit"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Email     string   `json:"email"`
	Number    int      `json:"number"`
}

func TestParserBodyJSON(t *testing.T) {
	jsonBytes := []byte(`{
	"first_name":"My firstname",
	"last_name": "My lastname",
	"email": "email@myemail.com",
	"number": 8253645635463
}`)

	val, err := Body("application/json", bytes.NewReader(jsonBytes), reflect.TypeOf(sampleInfo{}))
	assert.Nil(t, err)
	assert.NotNil(t, val)

	s := val.Interface().(sampleInfo)
	assert.Equal(t, "My firstname", s.FirstName)
	assert.Equal(t, "My lastname", s.LastName)
	assert.Equal(t, "email@myemail.com", s.Email)
	assert.Equal(t, 8253645635463, s.Number)

	// Error
	errJSONBytes := []byte(`{
	"first_name":"My firstname",
	"last_name": "My lastname",
	"email": "email@myemail.com",
	"number": 8253645635463,
}`)

	_, err = Body("application/json", bytes.NewReader(errJSONBytes), reflect.TypeOf(sampleInfo{}))
	assert.NotNil(t, err)
	assert.Equal(t, "invalid character '}' looking for beginning of object key string", err.Error())
}

func TestParserBodyXML(t *testing.T) {
	xmlBytes := []byte(`<Submit>
	<FirstName>My xml firstname</FirstName>
	<LastName>My xml lastname</LastName>
	<Email>myxml@email.com</Email>
	<Number>8253645635463</Number>
</Submit>`)

	val, err := Body("application/xml", bytes.NewReader(xmlBytes), reflect.TypeOf(&sampleInfo{}))
	assert.Nil(t, err)
	assert.NotNil(t, val)

	s := val.Interface().(*sampleInfo)
	assert.Equal(t, "My xml firstname", s.FirstName)
	assert.Equal(t, "My xml lastname", s.LastName)
	assert.Equal(t, "myxml@email.com", s.Email)
	assert.Equal(t, 8253645635463, s.Number)

	// Error
	errXMLBytes := []byte(`<Submit>
	<FirstName>My xml firstname</FirstName>
	<LastName>My xml lastname</LastName>
	<Email>myxml@email.com</Email>
	Number>8253645635463</Number>
</Submit>`)

	_, err = Body("application/xml", bytes.NewReader(errXMLBytes), reflect.TypeOf(&sampleInfo{}))
	assert.NotNil(t, err)
	assert.Equal(t, "XML syntax error on line 5: element <Submit> closed by </Number>", err.Error())
}

type sample struct {
	Int      int        `bind:"fint"`
	Int8     int8       `bind:"fint8"`
	Int16    int16      `bind:"fint16"`
	Int32    int32      `bind:"fint32"`
	Int64    int64      `bind:"fint64"`
	PInt     *int       `bind:"fpint"`
	PInt8    *int8      `bind:"fpint8"`
	PInt16   *int16     `bind:"fpint16"`
	PInt32   *int32     `bind:"fpint32"`
	PInt64   *int64     `bind:"fpint64"`
	Float32  float32    `bind:"ffloat32"`
	Float64  float64    `bind:"ffloat64"`
	PFloat32 *float32   `bind:"fpfloat32"`
	PFloat64 *float64   `bind:"fpfloat64"`
	UInt     uint       `bind:"fuint"`
	UInt8    uint8      `bind:"fuint8"`
	UInt16   uint16     `bind:"fuint16"`
	UInt32   uint32     `bind:"fuint32"`
	UInt64   uint64     `bind:"fuint64"`
	PUInt    *uint      `bind:"fpuint"`
	PUInt8   *uint8     `bind:"fpuint8"`
	PUInt16  *uint16    `bind:"fpuint16"`
	PUInt32  *uint32    `bind:"fpuint32"`
	PUInt64  *uint64    `bind:"fpuint64"`
	String   string     `bind:"fstring"`
	PString  *string    `bind:"fpstring"`
	Bool     bool       `bind:"fbool"`
	PBool    *bool      `bind:"fpbool"`
	Time     time.Time  `bind:"ftime"`
	PTime    *time.Time `bind:"fptime"`
	ISlice   []int      `bind:"fislice"`
	IPISlice []*int     `bind:"fipislice"`
	USlice   []uint     `bind:"fuslice"`
	IPUSlice []*uint    `bind:"fipuslice"`
	FSlice   []float32  `bind:"ffslice"`
	IPFSlice []*float32 `bind:"fipfslice"`
	SSlice   []string   `bind:"fsslice"`
	IPSSlice []*string  `bind:"fipsslice"`
}

func TestParserStruct(t *testing.T) {
	params, err := url.ParseQuery("fint=10002&fint8=127&fint16=3874&fint32=36437&fint64=3745343743874538&fpint=10002&fpint8=127&fpint16=3874&fpint32=36437&fpint64=3745343743874538&ffloat32=3.4747476&ffloat64=6.835483754873548735&fpfloat32=3.4747476&fpfloat64=6.835483754873548735&fuint=10002&fuint8=255&fuint16=3874&fuint32=36437&fuint64=3745343743874538&fpuint=10002&fpuint8=255&fpuint16=3874&fpuint32=36437&fpuint64=3745343743874538&fstring=safgsfdsdgj&fpstring=<script>javascript:</script>&fbool=true&fpbool=on&ftime=2017-08-20T05:53:45Z&fptime=2017-08-20T05:53:45-07:00&fislice=101&fislice=102&fislice=103&fislice=104&fipislice=101&fipislice=102&fipislice=103&fipislice=104&fuslice=101&fuslice=102&fuslice=103&fuslice=104&fipuslice=101&fipuslice=102&fipuslice=103&fipuslice=104&ffslice=1.243232&ffslice=6.343434&ffslice=9.5676576743625&fipfslice=1.243232&fipfslice=6.343434&fipfslice=9.5676576743625&fsslice=welcome1&fsslice=welcome2&fsslice=<script>welcome3</script>&fsslice=<script>welcome4</script>&fipsslice=welcome1&fipsslice=welcome2&fipsslice=<script>welcome3</script>&fipsslice=<script>welcome4</script>")
	assert.Nil(t, err)

	StructTagName = "bind"
	TimeFormats = []string{"2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05", "2006-01-02"}
	val, err := Struct("name", reflect.TypeOf(&sample{}), params)
	assert.Nil(t, err)

	s := val.Interface().(*sample)

	assert.Equal(t, 10002, s.Int)
	assert.Equal(t, int8(127), s.Int8)
	assert.Equal(t, int16(3874), s.Int16)
	assert.Equal(t, int32(36437), s.Int32)
	assert.Equal(t, int64(3745343743874538), s.Int64)

	assert.Equal(t, 10002, *s.PInt)
	assert.Equal(t, int8(127), *s.PInt8)
	assert.Equal(t, int16(3874), *s.PInt16)
	assert.Equal(t, int32(36437), *s.PInt32)
	assert.Equal(t, int64(3745343743874538), *s.PInt64)

	assert.Equal(t, uint(10002), s.UInt)
	assert.Equal(t, uint8(255), s.UInt8)
	assert.Equal(t, uint16(3874), s.UInt16)
	assert.Equal(t, uint32(36437), s.UInt32)
	assert.Equal(t, uint64(3745343743874538), s.UInt64)

	assert.Equal(t, uint(10002), *s.PUInt)
	assert.Equal(t, uint8(255), *s.PUInt8)
	assert.Equal(t, uint16(3874), *s.PUInt16)
	assert.Equal(t, uint32(36437), *s.PUInt32)
	assert.Equal(t, uint64(3745343743874538), *s.PUInt64)

	assert.True(t, s.Bool)
	assert.True(t, *s.PBool)

	assert.NotNil(t, s.Time)
	assert.NotNil(t, *s.PTime)

	assert.False(t, len(s.ISlice) == 0)
	assert.False(t, len(s.IPISlice) == 0)
	assert.False(t, len(s.USlice) == 0)
	assert.False(t, len(s.IPUSlice) == 0)
	assert.False(t, len(s.FSlice) == 0)
	assert.False(t, len(s.IPFSlice) == 0)
	assert.False(t, len(s.SSlice) == 0)
	assert.False(t, len(s.IPSSlice) == 0)

	assert.Equal(t, float32(1.243232), *s.IPFSlice[0])
	assert.Equal(t, "welcome1", *s.IPSSlice[0])
	assert.Equal(t, 101, *s.IPISlice[0])
	assert.Equal(t, uint(101), *s.IPUSlice[0])
}
