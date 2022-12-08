package process

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/brexhq/substation/config"
)

var replaceTests = []struct {
	name     string
	proc     Replace
	test     []byte
	expected []byte
	err      error
}{
	{
		"json",
		Replace{
			Options: ReplaceOptions{
				Old: "r",
				New: "z",
			},
			InputKey:  "foo",
			OutputKey: "foo",
		},
		[]byte(`{"foo":"bar"}`),
		[]byte(`{"foo":"baz"}`),
		nil,
	},
	{
		"json delete",
		Replace{
			Options: ReplaceOptions{
				Old: "z",
				New: "",
			},
			InputKey:  "foo",
			OutputKey: "foo",
		},
		[]byte(`{"foo":"fizz"}`),
		[]byte(`{"foo":"fi"}`),
		nil,
	},
	{
		"data",
		Replace{
			Options: ReplaceOptions{
				Old: "r",
				New: "z",
			},
		},
		[]byte(`bar`),
		[]byte(`baz`),
		nil,
	},
	{
		"data delete",
		Replace{
			Options: ReplaceOptions{
				Old: "r",
				New: "",
			},
		},
		[]byte(`bar`),
		[]byte(`ba`),
		nil,
	},
	{
		"data",
		Replace{
			Options: ReplaceOptions{
				New: "z",
			},
		},
		[]byte(`bar`),
		[]byte(`baz`),
		errMissingRequiredOptions,
	},
}

func TestReplace(t *testing.T) {
	ctx := context.TODO()
	capsule := config.NewCapsule()

	for _, test := range replaceTests {
		capsule.SetData(test.test)

		result, err := test.proc.Apply(ctx, capsule)
		if err != nil {
			if errors.Is(err, test.err) {
				continue
			}
			t.Error(err)
		}

		if !bytes.Equal(result.Data(), test.expected) {
			t.Errorf("expected %s, got %s", test.expected, result.Data())
		}
	}
}

func benchmarkReplace(b *testing.B, applicator Replace, test config.Capsule) {
	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		_, _ = applicator.Apply(ctx, test)
	}
}

func BenchmarkReplace(b *testing.B) {
	capsule := config.NewCapsule()
	for _, test := range replaceTests {
		b.Run(test.name,
			func(b *testing.B) {
				capsule.SetData(test.test)
				benchmarkReplace(b, test.proc, capsule)
			},
		)
	}
}
