# P1-06d19: isolate legacy retirement queries

Base: 0703f634300a485e6fb547b354fe355ca32f14a0.

- Extract legacy address-hold, Create binding, claim and Pin collision SQL into a dedicated legacy query file. Scoped Pin collision has its own explicit predicate.
- Separate legacy maintenance foreign-count and pending-operation reads. Preserve digest labels and query bytes pending the schema transition.
- Retain the transactional empty-legacy-table guard; no table, runtime witness or safety check was removed.

Box evidence (9d374e18-196a-493d-bf1f-2d582589f04d):

- b4c9956000468cb0674ef42f4a5ab200: edit, exit zero.
- d999f7de4d798311c1de0789f9af29a4: exact query equivalence, focused tests and vet, exit zero.
- 345ac170ca0d116262d7117b5ab93a83: maintenance extraction, tests and vet, exit zero.
- 92fe03b2b3180f668809154f4ffd09bc: diff check, focused state race tests and full build, exit zero.

Independent CI remains required. No final P1 completion, legacy removal, alternate-pool acceptance, merge or release is claimed.
