// Copyright (c) 2016-2017 Pani Networks
// All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package expression

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

func TestEval(t *testing.T) {
	cases := []struct {
		name       string
		expression string
		labels     map[string]string
		expected   bool
	}{
		{
			"ev1",
			"a==b",
			map[string]string{
				"a": "b",
			},
			true,
		},
		{
			"ev2",
			"a==b,c!=d",
			map[string]string{
				"a": "b",
			},
			true,
		},
		{
			"ev3",
			"a==b,c!=d",
			map[string]string{
				"a": "b",
				"c": "b",
			},
			true,
		},
		{
			"ev4",
			"a==b,c!=d",
			map[string]string{
				"a": "b",
				"c": "d",
			},
			false,
		},
		{
			"ev5",
			"c!=d",
			map[string]string{},
			true,
		},
		{
			"ev6",
			"c!=d,a=1,b==2",
			map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expression, err := ParseExpression(tc.expression)
			if err != nil {
				t.Errorf("failed to parse an expression %s", tc.expression)
				return
			}

			res := expression.Eval(tc.labels)
			if res != tc.expected {
				t.Errorf("%s got wrong result %t != %t", tc.name, res, tc.expected)
			}
		})
	}
}
