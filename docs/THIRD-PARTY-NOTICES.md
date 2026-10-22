# Included client libraries

- internal/web/static/hub.js: official @nimiq/hub-api 1.14.0 bundle, reused unchanged
  from PhoenixMail; adjacent Apache-2.0 LICENSE-HUB-API preserved. API reference:
  https://www.nimiq.dev/hub/api-reference
- internal/web/static/iqons.js: existing official @nimiq/iqons 1.5.2 artwork/adapter
  from the imported engine, with adjacent LICENSE-IQONS. See the engine's
  web/NIMIQ-NOTICES.md for full upstream attribution/packaging notes.
- Go Authors portable Blake2b and its BSD license remain under the engine.
- JSDOM is used only for development tests; it is not part of the Go binary.

The MIT grant for original code does not replace these licenses or branding rights.
