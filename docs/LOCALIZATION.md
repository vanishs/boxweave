# Localization policy

BoxWeave uses **English (`en-US`) for all source code, comments, terminal messages, documentation, configuration descriptions and template policy text** in the current release.

This does not mean multilingual messages are implemented today. The project is organized so they can be added without changing stable protocol names:

- Human-facing text may be translated in future locale catalogs such as `locales/en-US/messages.json` and `locales/zh-CN/messages.json`.
- Locale identifiers should follow BCP 47, e.g. `en-US`, `zh-CN`, `fr-FR`.
- Stable machine identifiers (`boxweave`, config keys, file paths, MCP tool names, systemd units) remain **language-independent** and must never be translated.
- Unknown locales should fall back to `en-US` once localization is implemented.
- Logs should preserve parsable status codes independent of human translations.
- Security warnings must never be dropped or weakened in a translation.
- Do not add an advertised `--locale` option before message catalogs are wired into both server and client entry points.

Contributions in another language are welcome as proposed translations, but the canonical README and source comments remain English for international collaboration.
