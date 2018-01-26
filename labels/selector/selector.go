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

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type ExpressionType string

const (
	Eq      ExpressionType = "eq"
	NonEq   ExpressionType = "neq"
	Unknown ExpressionType = "unknown"
)

type SimpleExpression struct {
	Type ExpressionType
	Body string
}

func (e SimpleExpression) Key() string {
	var key string
	var idx int

	switch e.Type {
	case Eq:
		idx = strings.Index(e.Body, "=")
	case NonEq:
		idx = strings.Index(e.Body, "!=")
	case Unknown:
		// idx = 0
	}

	if idx >= 0 {
		key = e.Body[:idx]
	}

	return key
}

func (e SimpleExpression) Value() string {
	var key string
	var idx int
	var shift int

	switch e.Type {
	case Eq:
		shift = 2

		idx = strings.Index(e.Body, "==")
		if idx == -1 {
			idx = strings.Index(e.Body, "=")
			shift = 1
		}

		if idx == -1 {
			shift = 0
		}
	case NonEq:
		shift = 2
		idx = strings.Index(e.Body, "!=")
		if idx == -1 {
			shift = 0
		}
	case Unknown:
		// idx = 0
	}

	if idx+shift >= 0 {
		key = e.Body[idx+shift:]
	}

	return key
}

func (e SimpleExpression) testLabels(labels map[string]string) bool {
	switch e.Type {
	case Eq:
		val, ok := labels[e.Key()]
		if !ok {
			return false
		}
		if val != e.Value() {
			return false
		}

		return true
	case NonEq:
		val, ok := labels[e.Key()]
		if !ok {
			return true
		}
		if val != e.Value() {
			return true
		}

		return false
	}

	return false
}

func DetectSimpleExpressionType(expression string) (ExpressionType, error) {
	if strings.Contains(expression, ",") {
		return Unknown, errors.New("simple expresstion is invalid, must not contain commas (,) - " + expression)
	}

	neqs := strings.Count(expression, "!=")
	eqs := strings.Count(expression, "==")

	if neqs+eqs > 1 {
		return Unknown, errors.New("simple expression is invalid, must contain exactly one operator - " + expression)
	}

	// special case for "=" expressions
	if neqs+eqs == 0 {
		if strings.Count(expression, "=") == 1 {
			return Eq, nil
		}
	}

	if neqs == 1 {
		return NonEq, nil
	}

	if strings.Contains(expression, "===") {
		return Unknown, errors.New("simple expression is invalid, wrong operator (===) - " + expression)
	}

	if eqs == 1 {
		return Eq, nil
	}

	return Unknown, errors.New("failed to detect type of simple expression - " + expression)
}

type Expression []SimpleExpression

func (e Expression) Eval(labels map[string]string) bool {
	for _, exp := range e {
		if !exp.testLabels(labels) {
			return false
		}
	}

	return true
}

func ExpressionFromMap(labels map[string]string) Expression {
	var result Expression

	for k, v := range labels {
		result = append(result, SimpleExpression{
			Type: Eq,
			Body: fmt.Sprintf("%s=%s", k, v),
		})
	}

	return result
}

func ParseExpression(expression string) (Expression, error) {
	var result Expression

	simplets := strings.Split(expression, ",")
	for _, simplet := range simplets {
		expType, err := DetectSimpleExpressionType(simplet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse expression %v", expression)
		}

		result = append(result, SimpleExpression{Type: expType, Body: simplet})
	}

	return result, nil
}
