# Items, classes, skills and enemies

Most gameplay data comes from the RPG Maker database. The server reads the same
`Classes.rvdata2`, `Skills.rvdata2`, `Items.rvdata2`, `Weapons.rvdata2`,
`Armors.rvdata2`, `Enemies.rvdata2` and `States.rvdata2` files as the client.

## Items

`RPG::Item` fields used by the server:

| Field | Use |
|---|---|
| `ID`, `Name`, `Description` | identity and client text |
| `Scope` | target selection (none, one ally, all allies, one enemy, all enemies, user) |
| `Occasion` | battle/menu/always validation |
| `HitType` | certain/physical/magical |
| `AnimationID` | client animation |
| `SuccessRate`, `Repeats` | hit resolution |
| `MPCost`, `TPCost` | resource cost |
| `Damage` | type, element, formula, variance, critical |
| `Effects` | recover HP/MP, states, buffs, growth, learn skill, common event |
| `Features` | passive features while equipped/active |
| `Range`, `AOE` | note tags used by area skills/items |
| `Level` | note tag: required level |
| `Soulbound` | note tag: cannot be traded/dropped |
| `AniIndex` | note tag: attack animation index |
| `ITypeID`, `Price`, `Consumable`, `KeyItem` | item behavior |

`Damage.Formula` is Lisp. The client does not evaluate it.

### Common item effects

- `Effect` code 11/12: recover HP/MP.
- code 21/22: add/remove state.
- code 31/32: add/remove buff.
- code 41: grow parameter.
- code 43: learn skill.
- code 44: common event (`DataID` = common event id).

The server validates ownership, stack limits, MP cost, level and occasion before
applying effects. Inventory changes are persisted.

## Weapons and armors

Both expose:

- `Params[8]` (RPG Maker parameter order);
- `Features`;
- `Price`;
- `ETypeID`/`WTypeID` or `ATypeID`;
- note tags: `Level`, `VIP`, `Soulbound`, `AniIndex`, `Paperdoll=Graphics/Index`;
- armor `Type` and `Sex` notes for slot/sex restrictions.

Equipping validates the slot, class/type, sex, level, VIP and soulbound rules.
Paperdoll data is sent to the client; the server only stores the values.

## Classes

| Field | Use |
|---|---|
| `Params` table | base parameters per level (0 mhp, 1 mmp, 2 atk, 3 def, 4 mat, 5 mdf, 6 agi, 7 luk) |
| `ExpParams` | exact RGSS3 `exp_for_level` curve |
| `Learnings` | skill id learned at level |
| `Features` | class features |
| note graphics | male/female sprite and face lists for character creation |

Character creation uses `Actors`/`Classes` and the shared limits to compute the
initial state. Level-up applies class parameter deltas and learnings on the
server.

## Skills

Skills share `UsableItem` and add:

- `STypeID` (skill type);
- `RequiredWTypeID1/2` (weapon requirements).

The server checks learned skills, level, MP, weapon requirements, range/AOE and
target validity. Projectile skills use the shared `range_skills` definitions in
`content/configs.toml`; IDs in `ProjectileSkills` are configured server-side.

## Enemies

| Field | Use |
|---|---|
| `Params[8]` | base stats |
| `Exp`, `Gold` | rewards |
| `DropItems` | kind/data id/denominator; `0` denominator disables |
| `Actions` | skill, condition, rating; AI chooses valid actions |
| `Features` | traits |
| note `Sight` | aggro/chase sight (default 6) |
| note `ReviveTime` | respawn seconds |
| note `DisableSwitchID` / `DisableVariableID` | increment/toggle on death |
| note `Escape` | flee at low HP |
| note `Boss` | boss behavior/name color |
| note `AniIndex` | attack animation index |

Enemy AI runs on the map actor: acquire target, chase, use actions, move with
the configured frequency, die/revive and drop items. Drops are map-local and
AOI-scoped.

## States

States are indexed by id. The server uses:

- `Restriction` (movement/action restrictions);
- `Priority`, `RemoveByRestriction`, `RemoveByDamage`, `ChanceByDamage`;
- `AutoRemovalTiming`, `MinTurns`, `MaxTurns`, `StepsToRemove`;
- `Features` (state rate/resist, parameter/element rates, attack states,
  regeneration, ex-parameters);
- `IconIndex` for client sync.

Buffs/debuffs are separate timed levels (-2..2). State/buff changes are sent to
the owner and to nearby players through AOI-scoped vitals/target updates.

## Validation checklist

1. IDs are unique and within the limits in `content/configs.toml`.
2. Damage formulas compile (`vxaos-formula check`).
3. Skill/item notes (`Range`, `AOE`, `Level`, `Soulbound`, `VIP`) are valid.
4. Drops reference existing items/weapons/armors.
5. Enemy actions reference existing skills.
6. State features use supported codes; unsupported features are ignored.
7. `vxaos-content check` passes.
8. `go test ./...` passes.
