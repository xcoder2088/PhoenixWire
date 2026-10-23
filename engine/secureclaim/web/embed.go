// Package web embeds the pages the recipient of a USDC / USDT card needs.
//
// Copyright (C) 2026 Xcoder — MIT License (see ../LICENSE).
package web

import "embed"

// Claim holds the claim page, its protocol library, the camera QR reader and the images it shows.
//
//go:embed usdt.html cashlink.js jsqr.js usdc-coin.png usdt-coin.png phoenixmail-official-emblem.webp
var Claim embed.FS
