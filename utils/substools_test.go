package utils

import (
	"reflect"
	"testing"
)

func TestSubtitleNames(t *testing.T) {
	tt := []struct {
		name    string
		streams []streams
		want    []string
	}{
		{
			name: "single named subtitle keeps label",
			streams: []streams{
				{
					CodecType: "subtitle",
					Tags: map[string]string{
						"title":    "English",
						"language": "eng",
					},
				},
			},
			want: []string{"English (eng)"},
		},
		{
			name: "duplicate language-only labels are numbered",
			streams: []streams{
				{
					CodecType: "subtitle",
					Tags: map[string]string{
						"language": "eng",
					},
				},
				{
					CodecType: "subtitle",
					Tags: map[string]string{
						"language": "eng",
					},
				},
			},
			want: []string{"1", "2"},
		},
		{
			name: "different labels keep title and language",
			streams: []streams{
				{
					CodecType: "video",
				},
				{
					CodecType: "subtitle",
					Tags: map[string]string{
						"title":    "SDH",
						"language": "eng",
					},
				},
				{
					CodecType: "subtitle",
					Tags: map[string]string{
						"language": "el",
					},
				},
			},
			want: []string{"SDH (eng)", "el"},
		},
		{
			name: "missing labels use subtitle index",
			streams: []streams{
				{
					CodecType: "subtitle",
				},
				{
					CodecType: "subtitle",
				},
			},
			want: []string{"1", "2"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := subtitleNames(tc.streams)
			if err != nil {
				t.Fatalf("subtitleNames() error = %v", err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("subtitleNames() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
