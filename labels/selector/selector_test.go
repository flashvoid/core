package selector

import "testing"

func TestDetectSimpleExpressionType(t *testing.T) {
	cases := []struct {
		name       string
		expression string
		expected   ExpressionType
	}{
		{
			"eq1",
			"a==b",
			Eq,
		},
		{
			"eq2",
			"a=b",
			Eq,
		},
		{
			"neq",
			"a!=b",
			NonEq,
		},
		{
			"unknown1",
			"a===b",
			Unknown,
		},
		{
			"unknown2",
			"a!b",
			Unknown,
		},
		{
			"unknown3",
			"a,b",
			Unknown,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DetectSimpleExpressionType(tc.expression)
			if result != tc.expected {
				t.Errorf("%s got wrong result %s != %s, err=%s", tc.name, result, tc.expected, err)
			}
		})
	}
}

func TestKey(t *testing.T) {
	cases := []struct {
		name       string
		expression SimpleExpression
		expected   string
	}{
		{
			"key eq 1",
			SimpleExpression{Type: Eq, Body: "a==b"},
			"a",
		},
		{
			"key eq 2",
			SimpleExpression{Type: Eq, Body: "a=b"},
			"a",
		},
		{
			"key neq 1",
			SimpleExpression{Type: NonEq, Body: "a!=b"},
			"a",
		},
		{
			"key unk",
			SimpleExpression{Type: NonEq, Body: "a=b"},
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := tc.expression.Key()
			if key != tc.expected {
				t.Errorf("%s got wrong result %s != %s", tc.name, key, tc.expected)
			}
		})
	}
}

func TestVal(t *testing.T) {
	cases := []struct {
		name       string
		expression SimpleExpression
		expected   string
	}{
		{
			"val eq 1",
			SimpleExpression{Type: Eq, Body: "a==b"},
			"b",
		},
		{
			"val eq 2",
			SimpleExpression{Type: Eq, Body: "a=b"},
			"b",
		},
		{
			"val neq 1",
			SimpleExpression{Type: NonEq, Body: "a!=b"},
			"b",
		},
		{
			"val unk",
			SimpleExpression{Type: NonEq, Body: "a=b"},
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := tc.expression.Value()
			if key != tc.expected {
				t.Errorf("%s got wrong result %s != %s", tc.name, key, tc.expected)
			}
		})
	}
}

func TestTestLabels(t *testing.T) {
	cases := []struct {
		name       string
		expression SimpleExpression
		labels     map[string]string
		expected   bool
	}{
		{
			"eq1",
			SimpleExpression{Type: Eq, Body: "a==b"},
			map[string]string{
				"a": "b",
			},
			true,
		},
		{
			"eq2",
			SimpleExpression{Type: Eq, Body: "a=b"},
			map[string]string{
				"a": "b",
			},
			true,
		},
		{
			"eq3",
			SimpleExpression{Type: Eq, Body: "a=b"},
			map[string]string{
				"a": "c",
			},
			false,
		},
		{
			"eq4",
			SimpleExpression{Type: Eq, Body: "a=b"},
			map[string]string{
				"b": "c",
			},
			false,
		},
		{
			"neq1",
			SimpleExpression{Type: NonEq, Body: "a!=b"},
			map[string]string{
				"a": "b",
			},
			false,
		},
		{
			"neq2",
			SimpleExpression{Type: NonEq, Body: "a!=b"},
			map[string]string{
				"a": "c",
			},
			true,
		},
		{
			"neq3",
			SimpleExpression{Type: NonEq, Body: "a!=b"},
			map[string]string{},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := tc.expression.testLabels(tc.labels)
			if res != tc.expected {
				t.Errorf("%s got wrong result %t != %t", tc.name, res, tc.expected)
			}
		})
	}
}
