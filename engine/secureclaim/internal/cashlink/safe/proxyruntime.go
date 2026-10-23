// Copyright (C) 2026 Xcoder
// SPDX-License-Identifier: MIT
// Originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder.

package safe

import "encoding/hex"

// ProxyRuntimeCode is the code found at a deployed card: Safe 1.4.1's tiny proxy, which delegates everything to the singleton
// kept in storage slot 0. It is the same for every card. Obtained by deploying a card through the real factory (sim/); used
// by tools/simulate to pretend a card exists.
var ProxyRuntimeCode = func() []byte {
	b, err := hex.DecodeString("" +
		"608060405273ffffffffffffffffffffffffffffffffffffffff600054167fa619486e00000000000000000000000000" +
		"00000000000000000000000000000060003514156050578060005260206000f35b3660008037600080366000845af43d" +
		"6000803e60008114156070573d6000fd5b3d6000f3fea264697066735822122003d1488ee65e08fa41e58e888a986555" +
		"4c535f2c77126a82cb4c0f917f31441364736f6c63430007060033",
	)
	if err != nil {
		panic(err)
	}
	return b
}()
