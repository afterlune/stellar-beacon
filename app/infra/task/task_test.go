package task

import "testing"

func TestProvinceFromIPSource(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		want   string
		wantOK bool
	}{
		{name: "province suffix", value: "中国|华东|浙江省|杭州", want: "浙江", wantOK: true},
		{name: "ordinary region", value: "0|0|北京|北京", want: "北京", wantOK: true},
		{name: "malformed", value: "内网IP|内网IP", wantOK: false},
		{name: "empty region", value: "0|0| |unknown", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := provinceFromIPSource(tt.value)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("provinceFromIPSource(%q) = %q, %v; want %q, %v", tt.value, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
