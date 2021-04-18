// Copyright 2017 GRAIL, Inc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// +build linux

package encryptiontest_test

var reliableGenerators = []generatorTestCase{
	{"ascendingDecimal", ascendingDecimal, true, false, false},
	{"ascendingHex", ascendingHex, true, false, false},
	{"ascendingBytes", ascendingBytes, true, false, false},
	{"pesudorand", pseudorand, false, true, false},
	{"cryptorand", cryptorand, false, true, true},

	// divx/zip require a precomputed random file to test with
	// due to lfs limits this file is no longer being servered by grailbio
	// The file may be downloaded via LFS from 39b3ca80f18 or earlier
	//{"divx", divx, true, false, false},
	//{"zip", zip, true, false, false},
}
