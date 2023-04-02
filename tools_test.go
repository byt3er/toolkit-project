package toolkit

import "testing"

func TestTool_RandomString(t *testing.T) {
	var testTools Tools
	s := testTools.RandonString(10)

	if len(s) != 10 {
		t.Error("Wrong length random string returned")
	}
}
