package symbol

import "testing"

func TestParse_FutuAndMarketCode(t *testing.T) {
	tests := []struct {
		in, code, market, futu string
	}{
		{"HK.00700", "00700", "HK", "HK.00700"},
		{"700.HK", "00700", "HK", "HK.00700"},
		{"700", "00700", "HK", "HK.00700"},
		{"US.AAPL", "AAPL", "US", "US.AAPL"},
		{"AAPL", "AAPL", "US", "US.AAPL"},
		{"SG.D05", "D05", "SG", "SG.D05"},
		{"CC.BTC", "BTC", "CC", "CC.BTC"},
		{"CC.HSK.BTCUSD", "HSK.BTCUSD", "CC", "CC.HSK.BTCUSD"},
		{"BMS.1155", "1155", "BMS", "BMS.1155"},
		{"US..IXIC", ".IXIC", "US", "US..IXIC"},
		{"HK.HSImain", "HSImain", "HK", "HK.HSImain"},
		{"US.CLmain", "CLmain", "US", "US.CLmain"},
		{"US.LO2612P62x5", "LO2612P62x5", "US", "US.LO2612P62x5"},
		{"aapl", "AAPL", "US", "US.AAPL"},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.in, err)
		}
		if got.Code != tt.code || got.Market != tt.market || got.FutuCode() != tt.futu {
			t.Errorf("Parse(%q)=%+v futu=%s want code=%s market=%s futu=%s",
				tt.in, got, got.FutuCode(), tt.code, tt.market, tt.futu)
		}
	}
}
