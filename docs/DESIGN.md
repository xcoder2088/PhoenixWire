# PhoenixWire interface

The current root page is a compact Wire host around **PhoenixMail's own payment window**. It is not the older hand-built form.
Read PAYMENT-WINDOW.md before changing the mirrored window.

## Current sender page

- `index.html` provides branding, explicit links to the PhoenixWire presentation and PhoenixMail, language/theme controls,
  sender verification, payment launch button and the permanent Powered by Phoenix Mail footer.
- `wire-pay.js` owns the five-language host text, sender session, copied-window options and invitation watcher.
- `wire-pay.css` styles the host, navigation and overrides of the copied window. Buttons use the theme accent.
- Before confirmation, the email field explains that confirmation lasts 24 hours on this device. Once signed in, a short label
  repeats the duration next to the sender identity. Changing sender or clearing the session requires another confirmation.
- The payment window supplies its real form, preview/funding card and sender-history controls. A visible return button invokes
  its existing close control, preserving cleanup and locally saved cards. Switching assets recreates the window; Wire reapplies
  the return control to each replacement without changing pm/.
- USDC Polygon, USDT Polygon and USDT Ethereum are offered when enabled. All cards are protected. NIM and payment requests are
  hidden in this window because Wire does not yet support them there. “Send NIM” opens the classic interface.
- Sender cards use the existing emblem and PhoenixWire name through `o.brand`; card backgrounds follow the copied pastel palettes.

## Browser and home-screen identity

The tab/application title is **PhoenixWire**, one word. Favicon, 180px Apple touch icon and 192/512px manifest icons derive from
the existing approved phoenix emblem. The manifest uses browser display mode; no service worker, offline-payment claim or new
installation flow is introduced. The wide wordmark remains the page logo. Icon routes are explicitly allowed by the static router;
the CSP is unchanged. Browser/phone shortcut caches may require removing an old shortcut and adding it again to see a changed icon.

## Classic NIM interface

`classic.html` and its `app.js`, `flow.js`, `usdt-flow.js`, model and styles remain present. `/?claim=…` selects this page for NIM
recipients. Its Aurora, PhoenixFire and Ocean palettes, light/dark appearance, official Nimiq artwork and preview behavior are
retained. Shared icon metadata and navigation do not replace its payment logic.

## Verification

Use the real host with the copied payment-window code and fake server in tests/wire-pay.test.mjs. Check direct visits,
signed-out and signed-in states, the return control before/after asset changes, five languages and mobile layouts.
Keep pm/ untouched unless a documented marked change is unavoidable. Bump static loader version stamps when changing assets.
