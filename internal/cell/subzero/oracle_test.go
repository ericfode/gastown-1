package subzero

import (
	"testing"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

func TestEvalOracleJsonParse(t *testing.T) {
	oracle := &parser.OracleBlock{
		Statements: []*parser.OracleStmt{
			{Kind: "json_parse"},
		},
	}

	// Valid JSON
	if err := EvalOracle(oracle, `{"key": "value"}`); err != nil {
		t.Errorf("valid JSON should pass: %v", err)
	}

	// Invalid JSON
	if err := EvalOracle(oracle, "not json"); err == nil {
		t.Error("invalid JSON should fail")
	}
}

func TestEvalOracleKeysPresent(t *testing.T) {
	oracle := &parser.OracleBlock{
		Statements: []*parser.OracleStmt{
			{Kind: "json_parse"},
			{Kind: "keys_present", Args: []string{"output", `["message", "status"]`}},
		},
	}

	if err := EvalOracle(oracle, `{"message": "hi", "status": "ok"}`); err != nil {
		t.Errorf("all keys present should pass: %v", err)
	}

	if err := EvalOracle(oracle, `{"message": "hi"}`); err == nil {
		t.Error("missing key should fail")
	}
}

func TestEvalOracleAssertIn(t *testing.T) {
	oracle := &parser.OracleBlock{
		Statements: []*parser.OracleStmt{
			{Kind: "json_parse"},
			{Kind: "assert", Expr: `output.action in ["PROCEED", "REJECT"]`},
		},
	}

	if err := EvalOracle(oracle, `{"action": "PROCEED"}`); err != nil {
		t.Errorf("PROCEED should be in list: %v", err)
	}

	if err := EvalOracle(oracle, `{"action": "INVALID"}`); err == nil {
		t.Error("INVALID should not be in list")
	}
}

func TestEvalOracleAssertLen(t *testing.T) {
	oracle := &parser.OracleBlock{
		Statements: []*parser.OracleStmt{
			{Kind: "json_parse"},
			{Kind: "assert", Expr: `len(output.message) > 0`},
		},
	}

	if err := EvalOracle(oracle, `{"message": "hello"}`); err != nil {
		t.Errorf("non-empty message should pass: %v", err)
	}

	if err := EvalOracle(oracle, `{"message": ""}`); err == nil {
		t.Error("empty message should fail")
	}
}

func TestEvalOracleRejectIf(t *testing.T) {
	oracle := &parser.OracleBlock{
		Statements: []*parser.OracleStmt{
			{Kind: "json_parse"},
			{Kind: "reject", Expr: `output.action == "REJECT"`},
		},
	}

	if err := EvalOracle(oracle, `{"action": "PROCEED"}`); err != nil {
		t.Errorf("PROCEED should not be rejected: %v", err)
	}

	if err := EvalOracle(oracle, `{"action": "REJECT"}`); err == nil {
		t.Error("REJECT should be rejected")
	}
}

func TestEvalOracleNil(t *testing.T) {
	if err := EvalOracle(nil, "anything"); err != nil {
		t.Errorf("nil oracle should pass: %v", err)
	}
}
