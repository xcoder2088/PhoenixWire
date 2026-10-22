# Phoenix Wire visual baseline

- Opening view follows PhoenixMail's compact payment window: one bordered frame,
  form on the left, quiet empty preview on the right. Mobile stacks those panels.
- The chosen Phoenix Wire wordmark and slogan remain above/in the window;
  Powered by Phoenix Mail stays in the page footer and on each payment card.
- Cards follow the PhoenixMail structure: phoenix/wordmark header, small status
  pill, centered amount, rounded white content panel and rounded action button.
- NIM uses the lavender/blue/pink pastel gradient and official Nimiq Iqons.
  A labeled visual preview uses a public sample address; a real prepared payment
  uses its own server-supplied public address and Nimicon.
- USDT Ethereum uses mint/seafoam pastels; USDT0 Polygon uses lavender pastels.
  Both show the Tether mark. Their sending capabilities remain unchanged.
- No payment-card QR or spending secret is introduced by this visual revision.
- A card appears only after preview creation or restoring an existing payment.
  The preview action inside the illustrated card is disabled.

Browser checks: 1280px desktop, 390px mobile, light/dark appearance, NIM Nimicon
and both USDT variants. Existing payment/OTP tests continue to exercise the same APIs.
