# Architecture

Telegram bot for saving goals. User makes a goal in a dialogue, logs deposits,
checks status.

Code has layers. Domain in middle, no Telegram/SQL/network deps. Around it:
bot, service, scenario, i18n, storage. Deps point inward. `internal/domain`
imports only stdlib.

    cmd/bot/main.go  ->  wiring only

    bot ----> service ----> domain <---- storage/postgres
     |           |
     v           v
    scenario -> i18n

## Layers

**domain** has `Goal`, `Deposit`, `Money`, `Status`, and math:
`ComputeStatus`, `ValidateGoal`, `ExchangeRate.Convert`. Declares repo
interfaces it needs. No `time.Now()` inside, `now` comes as parameter.

**service** is actions bot can do. Calls repo interfaces, calls pure domain
funcs, does currency conversion. `SavingsService.CreateGoal` validates before
`ComputeStatus` sees anything.

**scenario** is dialogue FSM. Graph of nodes plus one pure `Step`. Knows
nothing about goals. `service` reads collected vars and decides what they
mean. `ResultGoal` means save a goal, `ResultDeposit` and `ResultWithdraw`
mean save one signed deposit, `ResultNone` means clear state.

**bot** is thin. One handler, `b.route`, for all updates. Parses, resolves
intent, dispatches. Business rules stay out. `format.go` renders a `Status`
into the message text; `keyboards.go` builds the reply and inline keyboards;
`onCallback` handles inline-button presses.

**storage/postgres** is SQL. Implements domain interfaces.

**i18n** is string catalog for three languages.

**config** reads `.env` and env, validates, returns `Config`.

`cmd/bot/main.go` is config -> logger -> context -> pool -> migrate ->
services -> bot -> start. No logic.

## Money

`Money` is int64 minor units. No floats.

`Money` has no currency inside. Currency on record. Goal has `Currency` (what
we save for) and `SavingsCurrency` (what we save in). Deposit has amount, read
in goal `SavingsCurrency`.

Example: goal MacBook, EUR, targets 1500/1800/2100. Save in RUB. Deposits
5000, -1000, 10000. Targets converted to RUB through `ExchangeRate`. Same
currency on both sides -> `IdentityRate`, no math.

Deposits are signed. Withdrawal is negative row in same table. One `SUM()`.

Tier targets must be > 0. Zero target makes `saved >= target` always true,
marks that tier and all above as reached. Validation in service, before
`ComputeStatus`.

## Time

`ComputeStatus(goal, deposits, rate, now)`. `now` is parameter. Domain does
not call `time.Now()`. Test passes any date.

`Deadline` in UTC. `Timezone` is IANA name. Decides what "today" and "end of
month" mean for user. `startOfDay`, `daysUntil`, `monthsUntil` all use it.

Pace counts whole days. Anchor is start of creation day, reference is end of
current day, so the figures hold steady within a day and the day in progress
counts. Debt is `max(0, expected - saved)`, where expected is the target share
of days gone. `target * gone` passes int64 for large goals, so that product
runs through `math/big`.

## Dialogue

`Scenario` is list of `Node`. Node has text selectors, buttons, optional
`Parse` for free-text, map of next node per answer. `Step(node, answer)`
returns the next node id and the value kept.

Contract between scenario and service is `Var*` keys. Scenario writes answers
under them. `service.goalFromVars` reads them. Rename a key and both sides
break.

A rule that needs the clock stays out of the FSM. `Answer` checks the deadline
(at least tomorrow) at the deadline node, so a stale date asks the question
again and the answers already given stay. A zero amount in a money dialogue
cancels it: the service stores nothing and reports `ErrCancelled`, the bot
shows the cancel reply and the menu.

A withdrawal cannot take more than the goal holds. `Answer` checks the balance
(`SavingsService.Balance`) at the amount node, so an answer over the saved total
asks the amount again and reports `ErrOverdraw`; the bot shows the overdraw
reply and repeats the question. `Balance` is the sum of the goals signed
deposits, and a negative sum counts as nothing saved.

A money move ends with a phrase for its size. `domain.Mood` buckets the move: a
deposit by fixed amounts (5000 and 15000 whole units), a withdrawal by its share
of the saved total it comes out of (10 and 25 percent). `Answer` reports the mood
in `Outcome.Mood`, which also carries the next node and whether the dialogue runs
on. A stored goal, a cancel and `ResultNone` leave it `MoodNone`. i18n holds one
pool of phrases per mood and `TestCatalogComplete` forces all three languages;
`moodPhrase` in bot picks one at random and falls back to the plain done line.

`Result` says what a finished dialogue stores: `ResultGoal` saves a goal,
`ResultDeposit` and `ResultWithdraw` save one signed deposit, `ResultNone`
clears the position.

Inline buttons are the goal picker. `goalsKeyboard(prefix, goals)` in bot
writes `<prefix>:<goal id>`, where prefix is `dep`, `wit` or `st`. `onCallback`
reads it back and opens the matching money dialogue or renders status.
`tiersKeyboard` (`dep:min|ok|max`) is dormant, no caller yet.

## i18n

Catalogs: ru, en, rofl. ru/en from Telegram `language_code`. rofl only via
`users.ui_language`, per-user override.

`Messages` is struct with one field per string. `TestCatalogComplete` fails on
empty field, so adding a string forces all three languages. Reply keyboard
buttons match by localized label, so intent resolution needs user catalog.

## Routing

One handler. `b.route`. No per-command handlers.

Dispatch order: active dialogue first, command/reply-keyboard button second,
callback query third. Order in `handlers.go`.

Commands are plain text without slash: `status`, `privacy`, `data remove all`.
Admin commands take arguments: `send <scenario> <id|all>` and `broadcast <text>`.
Only `/start` has slash, Telegram sends it that way.

## Storage

Postgres. Tables: `users`, `goals`, `deposits`, `user_scenario_state`.
Deposits cascade on goal delete.

Migrations are goose, annotation-based, embedded. No root `migrations/` dir.
`go:embed` cannot reach parent dirs, so files sit next to package that runs
them. `Migrate` runs on startup, idempotent.

Driver is pgx/v5.

## Dependencies

Four direct: `go-telegram/bot`, `pgx/v5`, `goose/v3`, `godotenv`. Rest
indirect.

Module path is `lab042.ru/doomsdaycalc`. Owner domain serves GitHub repo. Do
not rename.

## Non-obvious

**Money without currency.** If `Money` held currency, every func touching it
would need the two in sync. Mixing becomes runtime error. Currency on record
forces conversion into one place: service, once.

**`now` as parameter.** Pure func takes `now` as argument. Test picks any
date. No mock.

**Signed deposits.** Separate withdrawals table doubles queries and indexes
for same result.

**FSM as data plus one function.** Adding question is adding node. Control
flow stays. `Step` keeps same size.

**Strings in i18n.** Three languages plus override. Inline strings break
`TestCatalogComplete`.

**One handler.** Per-command handlers duplicate dispatch. `resolveIntent` is
one place that maps update to action.

## Roadmap

Done: skeleton and config, domain types with migrations, the status math,
goal and deposit services with their handlers, the dialogue FSM, and the
broadcast.

Still open, roughly in order:

**Notifications.** A daily scheduler that reminds the user to log a deposit,
in the user language.

**Admin chats.** A mapping between admin chats and users (an `admin_messages`
table), so the operator can read what people write back and answer them.

**Currency.** Exchange rates from an API, with a cache and a background
refresher. `ExchangeRate` already sits between the two currencies;
`ComputeStatus` stays as it is.

**Privacy.** The `privacy` command shows the privacy policy text, and
`data remove all` drops every row of the user.

**Archive.** Write-only and age-encrypted, kept outside the domain DB. Scope
is incoming messages the bot cannot place, plus the admin chat. Outgoing bot
messages, commands and button presses stay out. The bot holds the public key
only, so a server compromise has ciphertext only.