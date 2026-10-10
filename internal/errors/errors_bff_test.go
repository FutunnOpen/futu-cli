package errors

import "testing"

func TestFromBFF_Success(t *testing.T) {
	if err := FromBFF(0, "success", nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestFromBFF_RetCodeMapping(t *testing.T) {
	tests := []struct {
		retCode int
		want    string
	}{
		{-3, CodeBadRequest},
		{-7, CodeInvalidSymbol},
		{-8, CodeUnsupported},
		{-9, CodePermission},
		{-10, CodeNoData},
		{-11, CodeRateLimit},
		{-5, CodeServerError},
	}
	for _, tt := range tests {
		err := FromBFF(tt.retCode, "boom", nil)
		cliErr, ok := err.(*CLIError)
		if !ok {
			t.Fatalf("ret_code %d: expected CLIError, got %T", tt.retCode, err)
		}
		if cliErr.Code != tt.want {
			t.Errorf("ret_code %d: code=%s want=%s", tt.retCode, cliErr.Code, tt.want)
		}
	}
}

func TestFromBFF_ErrorBlockOverrides(t *testing.T) {
	err := FromBFF(-3, "ret msg", &BFFError{Code: "invalid_symbol", Message: "bad symbol"})
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.Code != CodeInvalidSymbol {
		t.Errorf("code=%s want=%s", cliErr.Code, CodeInvalidSymbol)
	}
	if cliErr.Message != "bad symbol" {
		t.Errorf("message=%q", cliErr.Message)
	}
}
