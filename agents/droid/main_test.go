package main

import (
	"errors"
	"testing"
)

func Test_splitJobName(t *testing.T) {
	testCases := []struct {
		Subject     string
		ProductName string
		RunID       string
		Err         error
	}{
		{
			"a-b-c",
			"a",
			"b",
			nil,
		},
		{
			"a123-b456-c789",
			"a123",
			"b456",
			nil,
		},
		{
			"",
			"",
			"",
			errors.New(`"" is not in format <product>-<runID>-<randomID>`),
		},
		{
			"a-b:c",
			"",
			"",
			errors.New(`"a-b:c" is not in format <product>-<runID>-<randomID>`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Subject, func(t *testing.T) {
			prod, run, err := splitJobName(tc.Subject)

			if tc.Err == nil {
				if err != nil {
					t.Error(err)
					return
				}
			} else {
				if err == nil {
					t.Logf("No error was encountered when %v was expected", tc.Err)
					t.Fail()
				} else if got, want := err.Error(), tc.Err.Error(); got != want {
					t.Logf("\n\tgot: \t%v\n\twant:\t%v", err, tc.Err)
					t.Fail()
				}
			}

			if prod != tc.ProductName {
				t.Logf("\n\tgot \t%q\n\twant:\t%q", prod, tc.ProductName)
				t.Fail()
			}

			if run != tc.RunID {
				t.Logf("\n\tgot \t%q\n\twant:\t%q", run, tc.RunID)
				t.Fail()
			}
		})
	}
}
