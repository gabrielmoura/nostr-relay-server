# Changelog

## Unreleased

- Added optional browser-only encrypted DM import for NIP-17 gift wraps, NIP-04 DMs and NIP-17 relay lists. Enable `dm_import.enabled`; imports require NIP-42, preserve encrypted events unchanged, and only bypass kind allowlist/WoT for the authenticated owner.
