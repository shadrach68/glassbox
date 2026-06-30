# Session Bookmarking

Saved debug sessions can be bookmarked with a human-readable name:

```bash
glassbox session save --name payroll-bug
glassbox session list
glassbox session load payroll-bug
```

`session load` is an alias for `session resume`. Lookups accept exact session
IDs, bookmark names, unique ID prefixes, and transaction hashes.

Bookmark names are stored with the saved session snapshot metadata and must be
unique.

When a session is saved, Glassbox validates the persisted snapshot before writing
it to the session store. This includes any audit-chain metadata
(`audit_hash`, `audit_signature`, `previous_session_hash`), so malformed or
incomplete chain state is rejected up front with actionable hints.
