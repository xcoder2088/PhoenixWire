module github.com/xcoder2088/PhoenixWire

go 1.22

toolchain go1.26.10

require github.com/xcoder2088/NimiqSecureClaim v0.0.0

require github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect

replace github.com/xcoder2088/NimiqSecureClaim => ./engine/secureclaim
