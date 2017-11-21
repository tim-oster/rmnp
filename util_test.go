// Copyright 2017 Tim Oster. All rights reserved.
// Use of this source code is governed by the MIT license.
// More information can be found in the LICENSE file.

package rmnp

import "testing"

func TestUtilGreaterThanSequence(t *testing.T) {
	if greaterThanSequence(35000, 30000) != true {
		t.Error("Expected 33000 > 30000")
	}

	if greaterThanSequence(30000, 35000) != false {
		t.Error("Expected 30000 < 35000")
	}

	if greaterThanSequence(10, 35000) != true {
		t.Error("Expected 10 > 35000")
	}
}

func TestUtilGreaterThanOrder(t *testing.T) {
	if greaterThanOrder(140, 100) != true {
		t.Error("Expected 140 > 100")
	}

	if greaterThanOrder(100, 140) != false {
		t.Error("Expected 100 < 140")
	}

	if greaterThanOrder(10, 240) != true {
		t.Error("Expected 10 > 240")
	}
}

func TestUtilDifferenceSequence(t *testing.T) {
	if differenceSequence(50, 20) != 30 {
		t.Error("Expected diff(50, 20) = 30")
	}

	if differenceSequence(20, 50) != 30 {
		t.Error("Expected diff(20, 50) = 30")
	}

	if differenceSequence(65535 - 10, 20) != 30 {
		t.Error("Expected diff(65535 - 10, 20) = 30")
	}
}
