# PhoenixMail Card Design System — approved visual direction

Reference: user-approved pastel glass card mockup (2026-10-19). Applies to all PhoenixMail-generated cards, including NIM send, payment request and SecureClaim.

## Non-negotiable visual criteria
- Preserve the user-approved composition: centered softly translucent rounded outer frame, inner pastel gradient payment card, prominent amount, sender identicon, pill-style alternate-card control for demo only, translucent white verification panel, large purple-gradient primary CTA and understated branded footer.
- Twelve pastel families: sky blue, lavender, mint, pink, peach, lemon, periwinkle, aqua, orchid, coral, pistachio, seafoam. Gradients may blend two/three pale colors, never dark/black/grey card backgrounds.
- Airy spacing, restrained 1px lavender outlines, large corner radii, delicate shadows and accessible contrast. Responsive down to mobile widths.
- **Use the actual PhoenixMail phoenix emblem**, not a generated phoenix, flame emoji, or rectangular black-background banner. Obtain or prepare a verified transparent-background brand asset before final branding is approved. Do not replace brand identity with approximations.
- Header must NOT contain “× Nimiq”. Footer: “Powered by PhoenixMail · NimiqSecureClaim”.
- Render actual official Nimiq identicon from the **verified funding sender address**; do not fabricate an address or identity. Until available, show an explicitly labeled placeholder.
- All cards generated in PhoenixMail should use shared CSS/tokens/components rather than independent inconsistent templates. Preserve email-client-safe HTML fallback where glass/blur/gradients aren't supported.

## Implementation sequence
1. Implement shared design tokens and responsive SecureClaim web card.
2. Add verified transparent PhoenixMail emblem and official sender Nimicon generator.
3. Build shared email-safe card renderer in PhoenixMail for send/request, preserving existing behavior and pastels.
4. Test across desktop/mobile and HTML email clients; check keyboard access, reduced motion, contrast and narrow screens.
5. Keep testnet-only, development-only payment flows until security and custody review.

This document describes an approved design target, not a claim that the production email templates are already migrated.
