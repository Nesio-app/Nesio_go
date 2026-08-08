# Legacy parity scope

## Product boundary

This repository will rebuild the selected, user-facing behavior from
[`Nesio-workshop`](https://github.com/Nesio-app/Nesio-workshop) on the current
Go API, React client, and Python AI service. The legacy source, its product
documents, and its behavior tests are the implementation reference; its
Next.js, Supabase, and IndexedDB code is not copied as the runtime stack.

The current application remains server-authoritative. Offline support will add
a client-side cache and durable mutation queue that synchronizes with the Go
API, rather than reproducing the legacy app's large collection of unrelated
`localStorage` keys.

## Confirmed scope

### Build now

- Account, onboarding, settings, export, and deletion
- Today: focus tasks, reminders, recall, daily reports, and proactive cards
- Capture: text, voice, camera, files, and share links
- Memory: search, relations, rich detail, and Ask with citations
- Items: rooms, recognition, location, expiry, and where-is recall
- Offline cache, cross-device synchronization, backup, and restore
- Push notifications and recurring reminders
- Google Calendar and Gmail
- Contacts and relationship management
- Flomo synchronization
- Health: medication, lab documents, Apple Health, and mood
- Finance: transactions, budgets, receipts, and Plaid
- Places, timeline, and travel planning
- Family, chores, and rewards
- Wardrobe, outfits, and virtual try-on
- Pantry, recipes, shopping, and meal planning
- Fitness and training plans
- Tesla and asset management
- Insights: topics, retrospectives, plans, and life map
- Proactive AI guidance and mirror reports

### Deferred

- Music and playlists
- Back-office / operational dashboard

### Excluded

- A standalone Lab surface. Functionality that is currently only reachable
  through Lab must move into the appropriate primary product surface.
- External toolbox applications
- The separate memorial product under legacy `memory/`

### Not selected yet

The prior grouping included Notion and reader import alongside Flomo. Flomo is
in scope; Notion and reader import stay out of the implementation queue until
they are explicitly selected.

## Technical principles

1. **Preserve real user data.** Never persist browser `blob:` URLs. Assets need
   durable authenticated storage before being referenced by records.
2. **Use stable client identities.** Offline-created records require
   client-generated IDs, idempotent writes, conflict versions, and a mapping
   path for legacy `local_id` values.
3. **Make writes observable and recoverable.** Capture is saved locally before
   optional AI enrichment; queued work exposes its state and failures.
4. **Keep reminders timezone-safe.** User-intended wall-clock schedules,
   recurrence rules, notification delivery, and Today cards share one canonical
   reminder model.
5. **Keep AI grounded.** Ask, insights, and proactive cards cite persisted
   records and surface uncertainty rather than inventing user facts.
6. **Keep sensitive data compartmentalized.** Health, finance, credentials,
   and private assets must not be sent to AI or third parties without an
   explicit feature-level policy.

## Dependency order

### Foundation

1. Restore a clean build and fix currently broken core contracts.
2. Define canonical LifeNode, Signal, asset, relationship, reminder, and
   synchronization schemas.
3. Add data migration adapters for legacy node types, IDs, relations, assets,
   reminders, and timezone semantics.
4. Establish authenticated asset storage, idempotent write APIs, and a
   client-side IndexedDB cache/queue.

### Core loop

5. Build capture → memory → Today → Ask/recall end to end.
6. Add rich memory detail, search, relations, assets, citations, and reminders.
7. Replace placeholder or inert UI with live data and provide full
   error/offline states.

### Connected domains

8. Ship Calendar, Gmail, Flomo, contacts, and push delivery.
9. Ship health, finance, places/travel, family, wardrobe, cooking, fitness,
   and Tesla/assets in dependency order.
10. Add insights, proactive guidance, reports, and mirror reports once their
    data sources are real.

## Acceptance approach

- Legacy v1 product rules in
  `docs/design/v1-product-spec-2026-07.md` define the primary UX contract.
- Relevant legacy `scripts/*.test.mjs` behavior tests are translated into Go,
  frontend, or end-to-end tests as each module is rebuilt.
- Every completed module has a live API path, an interactive UI path, and a
  regression test; static domain tiles are not treated as implemented.
