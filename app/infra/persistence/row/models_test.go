package row

import (
	"reflect"
	"testing"
	"time"

	"benetnasch/app/domain/port"
)

func TestDomainRecordsDoNotCarryXORMetadata(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(port.TArticle{}),
		reflect.TypeOf(port.TCategory{}),
		reflect.TypeOf(port.TComment{}),
		reflect.TypeOf(port.TUserAuth{}),
		reflect.TypeOf(port.TUserInfo{}),
	}
	for _, recordType := range types {
		for fieldIndex := 0; fieldIndex < recordType.NumField(); fieldIndex++ {
			if tag := recordType.Field(fieldIndex).Tag.Get("xorm"); tag != "" {
				t.Fatalf("%s.%s unexpectedly has xorm tag %q", recordType.Name(), recordType.Field(fieldIndex).Name, tag)
			}
		}
	}
}

func TestArticleConversionPreservesDomainValuesAndRowMetadata(t *testing.T) {
	want := port.TArticle{
		Id:             42,
		UserId:         7,
		CategoryId:     9,
		ArticleTitle:   "boundary",
		ArticleContent: "content",
		Status:         1,
		CreateTime:     time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC),
	}

	converted := ToArticle(want)
	if got := FromArticle(converted); got != want {
		t.Fatalf("round trip changed article: got %#v, want %#v", got, want)
	}
	field, ok := reflect.TypeOf(converted).FieldByName("Id")
	if !ok || field.Tag.Get("xorm") == "" {
		t.Fatal("persistence row lost xorm metadata for auto-increment id")
	}
	if got := field.Tag.Get("xorm"); got != "autoincr not null pk unique INTEGER" {
		t.Fatalf("unexpected id metadata: %q", got)
	}
}

func TestRowConversionsPreserveSlices(t *testing.T) {
	input := []TPhoto{{Id: 1, AlbumId: 2, PhotoName: "one"}, {Id: 3, AlbumId: 4, PhotoName: "two"}}
	got := FromPhotos(input)
	if len(got) != len(input) || got[0].PhotoName != "one" || got[1].AlbumId != 4 {
		t.Fatalf("photo slice conversion lost values: %#v", got)
	}
}
