# Chaosbyte — operational plan

**Status:** companion to the moments doc; awaiting Daniel's sign-off
**Date:** 2026-05-12

This doc is the operational layer. Conceptual framing lives in `2026-05-12-moments.md`. Here every system component, every event, every animation is specified as: **what it is, when it fires, how it works, where it appears, why it exists, and when it stays silent.** Anything that can't fill all six columns gets cut.

---

## Principles (the test every entry must pass)

1. **Quiet baseline.** Default state of the room is still. Animation costs attention — it has to earn its turn.
2. **Each move carries function.** Visual change = signal. If there's nothing to signal, no motion.
3. **Decay is continuous.** Hard on/off transitions are a tell that an effect was bolted on. Everything fades in, fades out.
4. **Restraint scales with intimacy.** A whisper gets a whisper-treatment. An alert gets weight. The animation budget per moment matches the moment's emotional volume.
5. **Cohesion beats variety.** Same primitives, same glyphs, same color use across every surface. Vocabulary the user can learn.

---

## The substrate (one engine, layered)

```
   internal/typo
   ┌────────────────────────────────────────────────────────────┐
   │  Layout      — immutable per content, per-char addressable │
   │  State       — mutable per tick (reveal, drop, tint, etc.) │
   │  Primitives  — pure state mutators (Type, Cascade, Wipe…)  │
   │  Macros      — intention layer (Greet, Amplify, Mourn…)    │
   │  Render      — Layout + State → styled rows                │
   └────────────────────────────────────────────────────────────┘

   Sits BENEATH every UI surface. No surface bypasses it.
```

| What | Why exists |
|---|---|
| **Layout** | Per-glyph addressability. Without it we can't animate words, only lines. |
| **State** | Decouples animation from content. Same layout reused frame-to-frame with mutating state. |
| **Primitives** | The lowest-level vocabulary. Pure functions, testable, composable. |
| **Macros** | The public API. Screens speak in *intentions* (Greet, Amplify), not primitive sequences. |
| **Render** | The single hot path. Everything visible goes through here. |

**Restraint at this layer:** primitives don't fire themselves. Macros must be triggered. Triggers are events. No animation runs unprompted.

---

## The tag system (the AI substrate)

Tags are how directors leave marks on chat content. Multi-source, time-bounded, action-bearing.

```go
type InteractionTag struct {
    Source   TagSource    // AIMod | HumanMod | System | Author | ChannelRule
    SourceID string       // which mod nick, which rule name
    Kind     string       // question | url | code | build-fail | spotlight-suggest | ...
    Reason   string       // human-readable why (hover-revealed)
    Marker   rune         // margin glyph: ✦ ★ ⚡ ◇ ▸
    Window   time.Time    // when this tag expires
    Sticky   bool         // overrides Window if mod pins it
    Scope    TagScope     // Public | ModsOnly | SelfOnly
    Target   string       // for SelfOnly: the nick this is for
    Actions  []Action     // 1..9 keyed; what user can do while focused
}
```

| What | When fires | How | Where | Why | Stays silent when |
|---|---|---|---|---|---|
| **Tag attaches** | Director.Annotate returns a tag | Broker stores tag-with-message; broadcasts to scope subscribers | Margin column to left of chat line | AI/human/system intelligence surfaces possibility on a message | No director identifies anything worth flagging — most messages stay clean |
| **Marker renders** | Tag's Window > now AND user has Scope | Glyph in margin column, dim style | One column left of timestamp | Whisper that this line is "alive" without claiming the line | Tag expired or out of scope |
| **Focus reveals menu** | User Tab-cycles to tag OR cursor hovers OR scrolls-to | Inline menu `[1 echo 2 thread 3 react esc]` replaces the row below briefly | Just below the tagged message line | Affordance only appears on engagement — no clutter for non-interactors | User isn't engaging |
| **Action fires** | User presses keyed digit or clicks action | Action.Run mutates broker state, may trigger its own macro | Outcome animates in place (e.g., thread expands inline) | Interaction has a result that's part of the chat, not a popup | User cancels (esc) or window expires mid-engagement |
| **Tag expires** | now > Window AND not Sticky | Broker sweep on each tick removes tag, broadcasts removal | Marker fades over ~200ms then disappears | Time-bounding prevents pile-up; the room stays clean | Sticky=true (human mod pinned it) |

---

## The directors (who issues tags)

Five sources. Different rules per source.

| Director | Marker | Trust | Window | Dismissable | Action skew |
|---|---|---|---|---|---|
| AI mod (Claude) | `✦` | suggestion | mod-decided (~5min default, scaled to confidence) | user can hide | exploratory: thread, react, echo, save, link |
| Human mod | `★` (`📌` if sticky) | authoritative | mod-set, may be sticky | only mod can clear | curatorial: pin, feature-in-spotlight, route-to-channel, mute, lock |
| System | `⚡` build, `▲` deploy, `↑` push | factual | event-bound (~30s transient, persistent for failures) | auto-expire | task: open-run, retry, link-PR |
| Author self-tag | `◇` | declarative | until edited | only author | meta: mark-question, mark-help-wanted, mark-soft, mark-WIP |
| Channel rule | `▸` | rule-based | rule-defined (auto-clear after N replies, etc.) | mod-config | routing: cross-post, summarize, archive |

| Director | When activates | How | Where it reads from | Why exists | Restraint |
|---|---|---|---|---|---|
| **AI mod** | Each new message + idle scans every ~30s | LLM call: system prompt + recent context → structured tags | Broker `#lobby` + active channels | The room's intelligence; sees patterns and surfaces moments humans miss | Confidence threshold — if uncertain, doesn't tag |
| **Human mod** | Mod issues `/pin`, `/feature`, `/route` slash commands | Slash handler creates HumanMod-source tag, broadcasts | Mod's intent | Authoritative curation; the human voice in the AI's room | Only fires when mod explicitly invokes |
| **System** | External webhook (CI, deploy, push) | Webhook receiver → System-source tag attached to a synthesized chat line | GitHub Actions, deploy pipelines, git push hooks | Facts about the room's external context — code state, build state | Doesn't tag opinions; only facts |
| **Author** | User runs `/me`, `/?` (mark question), `/wip` | Slash handler tags the user's most recent message | The author's intent | Self-organization; lets users opt in to AI attention | Only the author can tag their own; can't tag others |
| **Channel rule** | Each new message matches a pre-configured pattern | Mod-defined regex/rule fires synchronously on publish | Channel config | Convention enforcement; `#help` requires context, `#side-projects` auto-summarizes | Rules don't fire across channels |

**Cross-source conflict:**
- Markers stack in the margin column (`★✦⚡` = human + AI + system on same message)
- Action menu groups by source, highest trust first
- Human mod's actions can supersede/cancel AI tags
- System tags can't be silenced (they're facts)

---

## The chat scrollback (where most events land)

The single most important surface. Most of Chaosbyte happens here.

| What | When | How | Where | Why | Restraint |
|---|---|---|---|---|---|
| **Normal chat line** | User or remote chat arrives via broker | Plain Layout, no macro applied, instant render | Body row of scrollback | Most messages are just talking; signal-to-noise demands quiet defaults | Always (no animation on normal messages, ever) |
| **Join announcement** | ChatJoin event from broker | Greet macro on the join line (slow Type at 60ms/char) | Top of new arrival's line | A person entered the room — meaningful event worth marking | Already-seeded messages on lobby load don't re-greet |
| **Mod welcome on join** | Broker auto-publishes after ChatJoin | Settle macro on the mod's line (scramble then resolve) | Mod's line | The mod ACKNOWLEDGES the new arrival — relationship signal | Only fires once per join |
| **/me action** | User runs `/me <text>` | Settle macro on the full action line | The action line | User chose an expressive form; the form deserves the motion | Only on `/me`, not on normal text |
| **@mention** | Renderer detects `@nick` token in message | Tint the `@nick` cells accent + Pulse once on arrival | The mentioned word, inline | The mention is a relationship signal; emphasis lives ON the word | If the mention is your own past message reference (self-reference), no Pulse |
| **Tagged message marker** | Director attaches tag | Marker glyph in margin column, dim by default | Left margin column | Whisper of "this is alive" without occupying the line | Tag expired or out-of-scope |
| **Tagged message inline menu** | User Tab-focuses or hovers | Action menu replaces row beneath for ~3s | Below the message | Affordances only on engagement; no clutter otherwise | User isn't focused on it |
| **Reaction stain (heart)** | Anyone reacts ❤️ to a message | Persistent soft pink halo around the message body | Around the message text | The room's appreciation lives in the message itself, not as an emoji tail | Stain decays very slowly over hours; visible but not loud |
| **Reaction stain (fire)** | 🔥 reaction | Warm Tint + brief Pulse fade over 3s | Message body | Fire = "this is hot" — temperature is the metaphor | Stain settles to subtle warm hold |
| **Reaction stain (slump)** | 👎 reaction | Small persistent Drop offset (line sits 1 row lower) | Message position | Disapproval slumps the line physically | Stays until reacted-up by another user |
| **Reaction stain (chill)** | ☕ reaction | Muted Tint hold | Message body | Chill mood — message visually relaxes | Persistent low-intensity |
| **Temporal depth dimming** | Always | Each older message gets one step dimmer via Alpha | Whole scrollback | Reading flow becomes visible; you sense time without timestamps | N/A — always on |
| **@mention to YOU specifically** | Renderer detects current user's nick mentioned | Tint accent2 on your nick + persistent foreground until you scroll past | The mentioned `@you` cells | This is FOR you; eye must be summoned | Decays only after you've scrolled past (acknowledged) |
| **Reply-to-yours signal** | Someone replies to your message | Faint connection char drawn between parent and reply | Vertical column between parent and reply | Relationship visible structurally | If you've already seen both, no signal |
| **Old message decay** | Age > 30 minutes | Apply Quiet macro (alpha to 0.5, muted tint) | Old scrollback messages | Past conversation reads differently from current; depth of field | Active threads (replies still happening) stay bright |

---

## The mod's voice (AI-as-margin, not AI-as-line)

The mod doesn't post chat lines that scroll past everyone. The mod ANNOTATES.

| What | When | How | Where | Why | Restraint |
|---|---|---|---|---|---|
| **Mod welcome (ARRIVAL)** | First-ever join (no prior history) | Settle macro on a single mod line | Top of chat | This is the one moment the mod IS a chat line — establishing presence | Returns to margin-only after; never duplicates |
| **Mod periodic prompt** | ~90s no chat + room has questions to surface | One short line, Type-in slow | One chat line | Sometimes the mod needs to talk; rare | Only fires on genuine room silence + something to say |
| **Mod margin marker on a message** | AI judges a message worth attention | `✦` in left margin (the tag system) | Margin column | The mod's voice 95% of the time — quiet pointer, not a speech | Doesn't tag unless confidence > threshold |
| **Mod calls out a user** | Mod wants to spotlight a participant | ATTENTION macro: their nick Pulses across all visible messages from them in last 60s | The user's nick anywhere it appears in scrollback | "I see you" without explicit chat line | One per user per 5min max |
| **Mod queues spotlight** | AI suggests N as next spotlight candidate | Spotlight tag with `✦ next?` reason and an action `[1 accept 2 skip]` on a relevant chat line | The line that prompted the suggestion | Mod surfaces spotlight candidates from real activity, not from a static rotation | Doesn't queue if user opted out of being spotlight |
| **Mod amplifies a moment** | A message gets engagement (reactions, replies) → mod confirms it matters | Amplify macro: brief Tint + Pulse + slight grow | The amplified message | Reinforces what the room is choosing to care about | Doesn't amplify without external signal (reactions, replies) — mod follows, doesn't lead taste |
| **Mod warns / dims** | Toxic content / off-topic / spam | Quiet macro: dim + muted tint on offending line | The line | De-emphasis without removal; user can still scroll back | Reversible; mod's call, not removal |

---

## Spotlight (presentation surface)

| What | When | How | Where | Why | Restraint |
|---|---|---|---|---|---|
| **Spotlight enters** | Engine state → presenting | CascadeTo macro on title row in place | Title row of spotlight screen | New thing arriving deserves a moment; cascade IS the arrival | Doesn't fire on every screen render — only on rotation |
| **Highlights bullets render** | Card first appears for a spotlight | Type macros staggered ~150ms apart per bullet | Card body | Sequential reveal helps reader take in points one at a time | Already-seen card on re-render: instant, no macro |
| **Timer ticks** | Each second of presenting | Digit Drop primitive on the changed digit | Status row "X:XX remaining" | Time feels like it's moving; weight on the digit that changed | Stable digits don't animate |
| **Last 10s** | Remaining < 10s | Timer cells Pulse subtly at 1Hz | Status row | Tension grows naturally as time runs out | Pulse rate doesn't increase further; tension is read, not pushed |
| **Last 3s** | Remaining < 3s | Scramble on the seconds digit | Status row | Final urgency; the digit literally agitates | Stops as soon as transition begins |
| **Opt-in window starts** | Engine → opt-in state | CascadeTo on "next: <project>" + Type-in on the prompt copy | Opt-in panel | Decision point — user is being asked | Doesn't auto-trigger again within the window |
| **Spotlight ends** | Presenting → transition | Mourn macro on title (slow Wipe right-to-left) | Title row | Departure has weight; the project doesn't just vanish | One-shot per rotation; doesn't replay |
| **Spotlight bleed (into chat)** | Active spotlight name appears in ANY chat message | Auto-tint those cells accent2 | Anywhere across all channels | The spotlight isn't a separate place — its energy radiates to wherever it's mentioned | Only the currently-active spotlight project name; not historical |

---

## Games (as conversational artifacts)

| What | When | How | Where | Why | Restraint |
|---|---|---|---|---|---|
| **`/blitz` starts** | User runs `/blitz` | Wipe-in on a game region at bottom of `#lobby` scrollback | Bottom region of the channel, 18 rows | Game is a moment IN the conversation, not a separate screen | Doesn't grab full screen; chat above stays readable |
| **Falling bars** | During game tick | Drop primitive per bar with gravity easing | Game region | Physical weight on falling text; the bar IS the moving thing | Bars vanish after they leave or hit floor; no afterimage |
| **Paddle motion** | User left/right input | Optimistic local position update + broker broadcast | Paddle row | Local feels instant; multiplayer sync stays consistent | No paddle "trail" — that would be backdrop art again |
| **Catch (paddle hits bar)** | Collision tick | Tint flash on bar cells (warm) → Wipe out + score Cascade in HUD | Bar location + HUD | The score increase IS the event; catch is acknowledged tactilely | Per-catch flash is brief (~100ms) |
| **Miss (bar floors)** | Bar reaches floor row | Tint flash red on bar + floor entry settles in place | Floor row | Failure visible; counts toward 3-misses-ends | Floor entry stays visible but doesn't pulse |
| **Score milestone** | Score crosses +50 | CascadeTo macro on score number | HUD score cell | Milestone deserves a beat | Only on multiples of 50; not every catch |
| **Timer last 10s** | Remaining < 10s | Pulse on timer cells at 1Hz | HUD timer | Tension | One-time threshold; no escalation |
| **Timer last 5s** | Remaining < 5s | Scramble on seconds digit | HUD timer | Final agitation | Replaces Pulse |
| **Game ends** | Timer expires or 3 missed | Wipe-out on game region + Settle macro on end card | Game region transitions to a permanent GameRecord chat message | Game ends as a CHAT POST — scrollback artifact, not modal | End card doesn't block; immediately becomes part of scrollback |
| **Game record in scrollback** | After game end | Persistent ChatGameRecord message kind; static unless replayed | The bottom of scrollback where the game was | Permanent record of room history; you can scroll back tomorrow and see it | No animation on rest state; only on `/replay` |
| **/replay <id>** | User runs replay | Layout sequence plays at recorded fps over the saved region | The original game's scrollback location | Re-watch capability; the record is a true record | Only plays for the requesting user (local-only render) |
| **/fork <id>** | User runs fork | New game starts with parent's state + variant rules; posts new GameRecord with `forked from <id>` reference | Bottom of current channel | Games beget games; tournaments emerge naturally | Parent record unchanged; fork is a new artifact |

---

## Header chip (screen + state indicator)

| What | When | How | Where | Why | Restraint |
|---|---|---|---|---|---|
| **Screen name** | Always | Static lipgloss render | Header bar, left | Identity — what screen am I on | N/A |
| **Screen switch** | Navigate event | CascadeTo macro on chip in place | Header bar | Old name dissolves into new; classic split-flap board | One-shot per switch; chip is otherwise static |
| **Unread indicator** | Channel has events since last view | Pulse on a single `•` glyph after the chip | Right of chip | Subtle dot that draws eye | Only when unread > 0; clears on view |
| **Connection state** | SSH heartbeat lapses (multi-user mode) | Tint chip color toward warn (orange) | The chip itself | Network health visible | Recovers automatically when heartbeat returns |

---

## Input area (where you type)

| What | When | How | Where | Why | Restraint |
|---|---|---|---|---|---|
| **Cursor blink** | Always while focused | Standard textinput cursor | Input line | Where will my next char go | N/A |
| **Submit clear** | User presses Enter | Wipe right-to-left on input contents | Input line | The text "leaves" rather than vanishing; transition to next state has weight | One-shot per submit |
| **Slash palette open** | User types `/` first | Palette items appear below input | Below input | Discoverability — what commands exist | Hidden when input doesn't start with `/` |
| **Tab autocomplete** | User presses Tab | Filled chars appear with no animation (just text) | Input line | Tab fills are user-initiated; no animation needed | N/A |
| **Submit error** | Slash command unknown or fails | Brief red Tint on input border | Input border | Feedback that something went wrong | One-shot, ~500ms |

**Restraint at this surface:** typing itself doesn't animate. Each keystroke is just text. The submit is the only moment.

---

## Member list (the room's roster)

| What | When | How | Where | Why | Restraint |
|---|---|---|---|---|---|
| **Nick rendered** | User in channel | Static text with their assigned color | Member list pane | Identity | N/A |
| **New join** | ChatJoin event | Type macro into alphabetical position | New nick's row | Person arriving is visible structurally | One-shot |
| **Leave** | Disconnect | Mourn macro on the leaving nick (slow Wipe) | Leaving nick's row | Departure has weight | Gap closes after Wipe completes |
| **Active typing indicator** | User is typing in input | Nick Pulses at 1Hz | The typing user's row | Awareness of live activity | Only while typing; stops on send or input clear |
| **Idle dim** | No chat from this user in 5 min | Tint toward muted | Their row | Presence vs activity distinguished | Brightens on next message |
| **Mod-tagged user** | Mod calls out someone | Persistent low-intensity Tint accent on their row | Member list | "Mod has them in their thoughts" — relationship signal | Decays when mod's tag expires |

---

## Channel mood (the room's weather)

| What | When | How | Where | Why | Restraint |
|---|---|---|---|---|---|
| **Active mood** | Many messages in last 5min | Subtle background tint shift toward warm | Header chip + member list | Room temperature is readable | Doesn't bleed into chat content readability |
| **Quiet mood** | <2 messages in last 10min | Muted palette across chrome elements | Header + member list | Quietness is a state, communicated visually | Chat content stays bright (legibility wins) |
| **Late-night** | Local time 22:00-06:00 | Cool palette pull on chrome | Header + member list | The room shifts as the day shifts | Doesn't change message colors |
| **Channel signature** | Always | Each channel has a per-channel accent color tint on the chrome | Header chip color | `#rust-anonymous` reads orange, `#vibe-coding` lavender, etc. | Tint is faint; doesn't drown chat content |

---

## Cross-cutting principles

**Cohesion checklist** (each move must pass all five):

1. Does this move convey information the user could not get otherwise? — if no, cut it.
2. Could this same effect work for three different surfaces? — if no, it's bespoke noise.
3. Does it decay continuously rather than hard cut? — if no, fix the decay.
4. Does another effect already cover this? — if yes, consolidate.
5. Does this still read at one-third intensity? — if no, the effect is too loud.

**Vocabulary discipline:**

- Margin markers: `✦ ★ ⚡ ◇ ▸ • ↗` — that's the set, period. No new glyphs without dropping one.
- Colors: theme.Fg / Muted / Accent / Accent2 / OK / Warn / Like — the existing seven. No new colors.
- Easing: linear for instant, ease-out for arrivals, ease-in for departures, ease-in-out for transitions. Four curves max.
- Timing: 100ms (instant), 400ms (moment), 1.5s (full Macro), 5s (window), persistent — five durations. Pick from this set.

---

## What we will NOT build (anti-gimmick guardrails)

The list of things that *sound cool* but fail the cohesion test:

| Idea | Why we're not building it |
|---|---|
| Animated background field with cells warping | Daniel-validated: this is ambient art, not function |
| Type-in animation on every normal chat line | Adds noise to high-frequency surface; defeats quiet baseline |
| Particle effects on send | No information conveyed; pure decoration |
| Per-user custom emoji pools | Vocabulary inflation; cohesion breaks |
| Sound effects | Out of scope for terminal; would also break the "watching a quiet room" feel |
| Gradient text in chat bodies | Legibility cost > aesthetic gain |
| Bouncing cursors | No information; visual cost; nothing earned |
| Confetti on milestones | Not the tone we want |
| Floating reactions that drift up | Slack-style; we're not Slack |
| User-customizable animation curves | Cohesion premium; everyone in the same room sees the same vocabulary |
| Persistent toast notifications | Reactions-on-source handles this without out-of-band noise |
| Glittering tag markers | The marker is a whisper; glitter is the opposite of whisper |

---

## Phased delivery (operational order)

Each phase delivers one user-visible improvement. We ship in phase order; we don't skip ahead.

### P0 — Foundation + restraint

| Item | Acceptance criteria |
|---|---|
| `internal/typo` Layout/State/Render/Primitives/Macros | Tests pass; existing tests stay green |
| Chat scrollback uses typo for body render | Normal chat appears instantly; joins Greet; `/me` Settles |
| Rip out `internal/field` grid renderer | Backdrop is gone; idle room visually still |
| Temporal depth dim on old messages | Past chat reads dimmer; current bright |

**Done test:** open lobby, watch for 30s with no input. The room is *quiet*. Then trigger a join. The room *acknowledges*. Then go quiet again.

### P1 — The tag system

| Item | Acceptance criteria |
|---|---|
| `InteractionTag` type + Director interface | Tags broadcast through broker; expiry sweep runs each tick |
| Rules-v0 director | Tags `?`-questions and URL-bearing messages |
| Margin marker render | `✦` appears next to tagged messages |
| Tab-cycle focus through tagged messages | Pressing Tab navigates between active tags in scrollback |
| Inline action menu on focus | `[1 react 🔥  2 thread  esc]` appears under focused tagged message |
| Two action implementations: react and thread | Reactions stain source; thread starts inline child Layout |

**Done test:** seed chat with a question. Marker appears. Tab focuses it. Menu shows. React 🔥. Source message stains warm. Tab again. Tag expires after 5min and marker fades.

### P2 — Multi-source directors

| Item | Acceptance criteria |
|---|---|
| Human mod director | `/pin`, `/feature`, `/route` produce HumanMod-source tags |
| System director | `/test-build-fail` synthetic webhook produces System tag with build-fail kind |
| Author self-tag director | `/?`, `/wip` on own most-recent message tags it |
| Stack rendering | Multiple sources on same message stack glyphs in margin |
| Trust ordering in action menu | Highest-trust source's actions appear first |

**Done test:** human mod pins a question. AI mod auto-tags it. System tags a related build fail. Three markers stack. Menu shows pin actions first.

### P3 — Spotlight bleed + chat semantics

| Item | Acceptance criteria |
|---|---|
| Spotlight bleed | Active spotlight project name auto-tints accent2 wherever it appears in chat |
| @mention parsing | Self-mentions become persistent foreground until scrolled past |
| Content marker parsing | `?`, `!`, `*emphasis*`, `_italic_` get inline treatment |
| Old message Quiet decay | Messages >30min apply Quiet macro |

**Done test:** post "lazygit is great" while lazygit is spotlit. The word lazygit tints. Mention yourself. Your nick stays bright until scrolled past.

### P4 — Games as artifacts

| Item | Acceptance criteria |
|---|---|
| Bricks rebuilt on typo | Paddle/bars/score/HUD all use typo primitives |
| Bricks renders in scrollback region | No fullscreen takeover; 18-row region at bottom of `#lobby` |
| GameRecord ChatMessage kind | End-of-game writes a permanent record |
| /replay implementation | Re-plays the recorded timeline |
| /fork implementation | Forks state, posts variant record |

**Done test:** play a blitz. End. Scroll back. Run /replay on the record. Watch it again.

### P5 — Mod as director (LLM)

| Item | Acceptance criteria |
|---|---|
| Claude API client + system prompt | Director.Annotate uses Claude with tool-use |
| Tag justification | Hovering a tagged message shows the AI's reason |
| Confidence threshold | Low-confidence tags don't fire |
| Negative-feedback loop | User dismissals weight future Claude calls |

**Done test:** Claude tags a question, suggests a related repo, explains why. User accepts the suggestion. Mod adapts.

### P6 — Atmosphere + Dev hooks

| Item | Acceptance criteria |
|---|---|
| Time-of-day chrome palette | Late-night reads cool, daytime reads warm |
| Channel signature | Each channel's chrome tints its accent |
| CI webhook receiver | Build pass/fail produces System tags on synthesized lines |
| PR open/merge from `gh` CLI poll | Tags surface in `#side-projects` or designated dev channel |

---

## Sign-off checklist before shipping each phase

For every item in the phase, ask:

- [ ] What does the user see / do?
- [ ] When does it trigger? (Specific event)
- [ ] How is it implemented? (Which macro/primitive)
- [ ] Where on screen does it appear?
- [ ] Why does this exist? (What does it serve?)
- [ ] When does it stay silent? (Anti-gimmick check)

If any column is empty or filler, the item gets cut or rewritten.

---

**Next move:** assuming sign-off, P0 starts now. Phase boundary is one commit. Daniel's call after each phase whether to continue.
