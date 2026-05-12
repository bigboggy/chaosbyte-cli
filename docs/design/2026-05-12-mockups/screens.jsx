/* chaosbyte — screens.jsx
   All 11 mockups + brand + principles. Rose Pine palette, JetBrains Mono.
   Each export is a self-contained artboard body. */

// ───────── primitives ─────────────────────────────────────────

const cbStyles = {
  // Page-level token bag every screen reads from. Inline so each artboard
  // ships its own copy regardless of where it's rendered.
  page: {
    width: "100%",
    height: "100%",
    background: "var(--base)",
    color: "var(--text)",
    fontFamily: '"JetBrains Mono", ui-monospace, Menlo, monospace',
    fontSize: 13,
    lineHeight: 1.55,
    fontWeight: 400,
    boxSizing: "border-box",
    display: "flex",
    flexDirection: "column",
    overflow: "hidden",
    fontFeatureSettings: '"calt" 1, "ss01" 1, "ss02" 1, "zero" 1',
  },
  muted: { color: "var(--muted)" },
  subtle: { color: "var(--subtle)" },
  gold: { color: "var(--gold)" },
  foam: { color: "var(--foam)" },
  love: { color: "var(--love)" },
};

function CBLine({ children, style }) {
  return (
    <div
      style={{ whiteSpace: "pre", fontVariantLigatures: "none", ...style }}
    >
      {children}
    </div>
  );
}

// Top status bar. Single line, no chrome. "chaosbyte · motd · count here"
function CBTopBar({ motd = "the workshop is open", count = 12 }) {
  return (
    <div
      style={{
        padding: "14px 22px 14px 22px",
        display: "flex",
        justifyContent: "space-between",
        alignItems: "baseline",
        borderBottom: "1px solid var(--overlay)",
        fontSize: 12,
      }}
    >
      <span style={cbStyles.subtle}>
        <span style={{ color: "var(--text)" }}>chaosbyte</span>
        <span style={{ color: "var(--muted)", margin: "0 10px" }}>·</span>
        {motd}
      </span>
      <span style={cbStyles.muted}>
        {count} here  ·  {new Date(2026, 4, 12, 23, 41).toLocaleString("en-GB", { hour: "2-digit", minute: "2-digit" })}  UTC
      </span>
    </div>
  );
}

// Channel list — left rail. Quiet. Current channel marked with a left bar.
function CBChannels({ current = "lobby", showDot = "ship" }) {
  const groups = [
    {
      label: "~ rooms",
      items: [
        ["lobby", 12],
        ["ship", 3],
        ["review", 1],
        ["aider", "·"],
        ["late-night", 7],
        ["spotlight", ""],
      ],
    },
    {
      label: "~ threads",
      items: [
        ["wasm-bindgen quirks", "·"],
        ["the case for tmux", ""],
        ["claude vs cline", ""],
      ],
    },
    {
      label: "~ direct",
      items: [
        ["jonas", "·"],
        ["km", ""],
      ],
    },
  ];
  return (
    <aside
      style={{
        width: 220,
        flexShrink: 0,
        padding: "22px 0 22px 22px",
        borderRight: "1px solid var(--overlay)",
        boxSizing: "border-box",
      }}
    >
      {groups.map((g) => (
        <div key={g.label} style={{ marginBottom: 22 }}>
          <div
            style={{
              ...cbStyles.muted,
              fontSize: 11,
              letterSpacing: 0.5,
              marginBottom: 6,
            }}
          >
            {g.label}
          </div>
          {g.items.map(([name, hint]) => {
            const isCurrent = name === current;
            const hasDot = name === showDot || hint === "·";
            return (
              <div
                key={name}
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "baseline",
                  padding: "1px 22px 1px 0",
                  borderLeft: isCurrent
                    ? "2px solid var(--gold)"
                    : "2px solid transparent",
                  paddingLeft: 8,
                  color: isCurrent ? "var(--text)" : "var(--subtle)",
                }}
              >
                <span>{name}</span>
                <span
                  style={{
                    color:
                      hint === "·" ? "var(--gold)" : "var(--muted)",
                    fontSize: 12,
                  }}
                >
                  {hint}
                </span>
              </div>
            );
          })}
        </div>
      ))}
    </aside>
  );
}

// Right margin — presence & quiet annotations. Mod markers go here too.
function CBMargin({ here, children }) {
  return (
    <aside
      style={{
        width: 180,
        flexShrink: 0,
        padding: "22px 22px 22px 18px",
        borderLeft: "1px solid var(--overlay)",
        boxSizing: "border-box",
        fontSize: 12,
        color: "var(--subtle)",
      }}
    >
      {children ? (
        children
      ) : (
        <>
          <div
            style={{
              ...cbStyles.muted,
              fontSize: 11,
              letterSpacing: 0.5,
              marginBottom: 6,
            }}
          >
            ~ here
          </div>
          {here.map((n) => (
            <div key={n} style={{ color: "var(--subtle)" }}>
              {n}
            </div>
          ))}
        </>
      )}
    </aside>
  );
}

// Bottom prompt. Just the user's nick and a slow blinking caret.
function CBPrompt({ nick = "ada", typed = "" }) {
  return (
    <div
      style={{
        padding: "14px 22px",
        borderTop: "1px solid var(--overlay)",
        display: "flex",
        gap: 12,
        alignItems: "baseline",
        fontSize: 13,
      }}
    >
      <span style={{ color: "var(--gold)" }}>{nick}</span>
      <span style={{ color: "var(--muted)" }}>›</span>
      <span style={{ color: "var(--text)" }}>{typed}</span>
      <span
        style={{
          display: "inline-block",
          width: "0.55em",
          height: "1em",
          background: "var(--subtle)",
          opacity: 0.5,
          transform: "translateY(2px)",
          animation: "cbBlink 1.6s steps(1) infinite",
        }}
      />
      <style>{`@keyframes cbBlink{0%,50%{opacity:.5}50.01%,100%{opacity:0}}`}</style>
    </div>
  );
}

// Generic message row, fixed columns. timestamp · nick · text · margin
function CBMsg({ time, nick, nickColor, text, margin, mute, indent = 0, fresh }) {
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "60px 110px 1fr 28px",
        padding: "2px 0",
        color: mute ? "var(--muted)" : "var(--text)",
        opacity: mute ? 0.85 : 1,
        position: "relative",
      }}
    >
      {fresh && (
        <span
          style={{
            position: "absolute",
            left: -10,
            top: 6,
            bottom: 6,
            width: 2,
            background: "var(--gold)",
            opacity: 0.7,
          }}
        />
      )}
      <span style={{ ...cbStyles.muted, fontSize: 12 }}>{time}</span>
      <span
        style={{
          color: nickColor || "var(--subtle)",
          paddingLeft: indent * 16,
        }}
      >
        {nick}
      </span>
      <span style={{ paddingRight: 24, textWrap: "pretty" }}>{text}</span>
      <span style={{ color: "var(--muted)", textAlign: "right" }}>{margin}</span>
    </div>
  );
}

// System / mod line — different shape, no nick column, often dimmed.
function CBSystem({ time, glyph = "·", text, tone }) {
  const color =
    tone === "warn" ? "var(--love)" :
    tone === "info" ? "var(--foam)" :
    tone === "mark" ? "var(--gold)" : "var(--muted)";
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "60px 110px 1fr 28px",
        padding: "4px 0",
        color: "var(--subtle)",
      }}
    >
      <span style={{ ...cbStyles.muted, fontSize: 12 }}>{time}</span>
      <span style={{ color, textAlign: "right", paddingRight: 14 }}>
        {glyph}
      </span>
      <span style={{ color: "var(--subtle)" }}>{text}</span>
      <span />
    </div>
  );
}

// Caption that lives below an artboard's outer frame; renders inside but at
// the bottom edge in muted type. One line of what's happening + one line of
// what the user feels.
function CBCaption({ happening, feeling }) {
  return (
    <div
      style={{
        padding: "12px 22px 16px 22px",
        borderTop: "1px dashed var(--overlay)",
        background: "#15121f",
        fontSize: 11,
        lineHeight: 1.6,
      }}
    >
      <div style={{ color: "var(--subtle)" }}>
        <span style={{ color: "var(--muted)" }}>what  </span>
        {happening}
      </div>
      <div style={{ color: "var(--subtle)" }}>
        <span style={{ color: "var(--muted)" }}>feel  </span>
        {feeling}
      </div>
    </div>
  );
}

// ───────── 01 · BRAND MARK ─────────────────────────────────────

function ScreenBrand() {
  return (
    <div style={cbStyles.page}>
      <div
        style={{
          flex: 1,
          padding: "56px 60px 0 60px",
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: 40,
        }}
      >
        {/* Wordmark + monogram column */}
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 56,
          }}
        >
          <div>
            <div
              style={{
                ...cbStyles.muted,
                fontSize: 11,
                letterSpacing: 1.4,
                marginBottom: 22,
              }}
            >
              ── WORDMARK
            </div>
            <div
              style={{
                fontSize: 56,
                fontWeight: 500,
                letterSpacing: -1.4,
                color: "var(--text)",
                lineHeight: 1,
              }}
            >
              chaosbyte
            </div>
            <div
              style={{
                marginTop: 14,
                color: "var(--muted)",
                fontSize: 12,
                letterSpacing: 0.4,
              }}
            >
              JetBrains Mono · 500 · −2.5% tracking
            </div>
          </div>

          <div>
            <div
              style={{
                ...cbStyles.muted,
                fontSize: 11,
                letterSpacing: 1.4,
                marginBottom: 22,
              }}
            >
              ── MONOGRAM
            </div>
            <div style={{ display: "flex", alignItems: "center", gap: 36 }}>
              <div
                style={{
                  width: 88,
                  height: 88,
                  border: "1px solid var(--overlay)",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  fontSize: 30,
                  fontWeight: 500,
                  letterSpacing: -1,
                  color: "var(--text)",
                }}
              >
                cb
              </div>
              <div
                style={{
                  width: 88,
                  height: 88,
                  border: "1px solid var(--overlay)",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  fontSize: 26,
                  color: "var(--gold)",
                }}
              >
                ✦
              </div>
              <div
                style={{
                  width: 88,
                  height: 88,
                  background: "var(--gold)",
                  color: "var(--base)",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                  fontSize: 30,
                  fontWeight: 600,
                  letterSpacing: -1,
                }}
              >
                cb
              </div>
            </div>
            <div
              style={{
                marginTop: 14,
                color: "var(--muted)",
                fontSize: 12,
              }}
            >
              prefers the wordmark · monogram for favicons and the tiny corner
            </div>
          </div>

          <div>
            <div
              style={{
                ...cbStyles.muted,
                fontSize: 11,
                letterSpacing: 1.4,
                marginBottom: 14,
              }}
            >
              ── MANIFESTO
            </div>
            <div
              style={{
                fontSize: 15,
                lineHeight: 1.8,
                color: "var(--text)",
                maxWidth: 380,
              }}
            >
              a room, not a feed.
              <br />
              built for people who already know how to read.
              <br />
              <span style={cbStyles.subtle}>
                stay as long as you like.
              </span>
            </div>
          </div>
        </div>

        {/* Right: ASCII banner + palette + spec */}
        <div style={{ display: "flex", flexDirection: "column", gap: 36 }}>
          <div>
            <div
              style={{
                ...cbStyles.muted,
                fontSize: 11,
                letterSpacing: 1.4,
                marginBottom: 14,
              }}
            >
              ── SSH BANNER
            </div>
            <pre
              style={{
                margin: 0,
                background: "var(--surface)",
                padding: "20px 22px",
                color: "var(--text)",
                fontSize: 12,
                lineHeight: 1.5,
                border: "1px solid var(--overlay)",
              }}
            >
{`  ┌─────────────────────────────────────────────┐
  │                                             │
  │   chaosbyte                                 │
  │   a small room for those who                │
  │   are paying attention.                     │
  │                                             │
  └─────────────────────────────────────────────┘

  73 in the room. last voice 4m ago.
  type :help to begin, :leave to leave well.`}
            </pre>
          </div>

          <div>
            <div
              style={{
                ...cbStyles.muted,
                fontSize: 11,
                letterSpacing: 1.4,
                marginBottom: 14,
              }}
            >
              ── PALETTE · rose pine
            </div>
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(5, 1fr)",
                gap: 8,
              }}
            >
              {[
                ["base", "#191724"],
                ["surface", "#1f1d2e"],
                ["overlay", "#26233a"],
                ["muted", "#6e6a86"],
                ["subtle", "#908caa"],
                ["text", "#e0def4"],
                ["gold", "#f6c177"],
                ["foam", "#9ccfd8"],
                ["love", "#eb6f92"],
              ].map(([name, hex]) => (
                <div key={name}>
                  <div
                    style={{
                      height: 44,
                      background: hex,
                      border:
                        name === "base" ? "1px solid var(--overlay)" : "none",
                    }}
                  />
                  <div
                    style={{
                      fontSize: 10,
                      color: "var(--subtle)",
                      marginTop: 4,
                    }}
                  >
                    {name}
                  </div>
                  <div style={{ fontSize: 10, color: "var(--muted)" }}>
                    {hex}
                  </div>
                </div>
              ))}
            </div>
            <div
              style={{
                marginTop: 10,
                fontSize: 11,
                color: "var(--muted)",
              }}
            >
              9 names. gold, foam, love appear rarely & on purpose.
            </div>
          </div>

          <div>
            <div
              style={{
                ...cbStyles.muted,
                fontSize: 11,
                letterSpacing: 1.4,
                marginBottom: 12,
              }}
            >
              ── TYPE
            </div>
            <div
              style={{
                fontSize: 12,
                color: "var(--subtle)",
                lineHeight: 1.9,
              }}
            >
              <span style={cbStyles.muted}>display  </span>
              JetBrains Mono · 500 · −2.5%<br />
              <span style={cbStyles.muted}>body     </span>
              JetBrains Mono · 400 · 13/20<br />
              <span style={cbStyles.muted}>meta     </span>
              JetBrains Mono · 400 · 11/16<br />
              <span style={cbStyles.muted}>fallback </span>
              Berkeley Mono → IBM Plex Mono
            </div>
          </div>
        </div>
      </div>

      <CBCaption
        happening="The mark, its monogram, an SSH banner, palette, and a 3-line manifesto."
        feeling="A place you stumbled into and decided to stay. Quiet, not empty."
      />
    </div>
  );
}

// ───────── 02 · ARRIVAL ───────────────────────────────────────

function ScreenArrival() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar motd="the workshop is open" count={12} />
      <div
        style={{
          flex: 1,
          padding: "0",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
        }}
      >
        <div style={{ width: 640, padding: "0 40px" }}>
          <pre
            style={{
              margin: 0,
              fontSize: 12,
              color: "var(--muted)",
              lineHeight: 1.5,
            }}
          >
{`   ┌────────────────────────────────────────────┐
   │                                            │`}
          </pre>
          <pre
            style={{
              margin: 0,
              fontSize: 12,
              color: "var(--muted)",
              lineHeight: 1.5,
            }}
          >
{`   │   `}
            <span style={{ color: "var(--text)", fontSize: 16, fontWeight: 500 }}>
              welcome back, ada.
            </span>
{`               │`}
          </pre>
          <pre
            style={{
              margin: 0,
              fontSize: 12,
              color: "var(--muted)",
              lineHeight: 1.5,
            }}
          >
{`   │   `}
            <span style={{ color: "var(--subtle)" }}>
              the room is quiet. 12 here.
            </span>
{`            │
   │                                            │
   └────────────────────────────────────────────┘`}
          </pre>

          <div style={{ height: 28 }} />

          <div
            style={{
              display: "grid",
              gridTemplateColumns: "60px 1fr 28px",
              fontSize: 13,
              lineHeight: 2.1,
            }}
          >
            <span style={cbStyles.muted}>23:41</span>
            <span style={cbStyles.subtle}>
              <span style={{ color: "var(--gold)" }}>✦ </span>
              ada has entered. last seen 3 days ago.
            </span>
            <span />
            <span style={cbStyles.muted}>23:41</span>
            <span style={cbStyles.subtle}>
              <span style={{ color: "var(--muted)" }}>· </span>
              you missed 14 lines in #lobby and one spotlight.
            </span>
            <span />
            <span style={cbStyles.muted}>23:41</span>
            <span style={cbStyles.subtle}>
              <span style={{ color: "var(--muted)" }}>· </span>
              <span style={cbStyles.foam}>jonas</span>
              {" "}left you a note 6h ago. type :note to read it.
            </span>
            <span />
          </div>

          <div style={{ height: 36 }} />

          <div style={{ fontSize: 13, color: "var(--muted)" }}>
            <span style={{ color: "var(--gold)" }}>ada</span>
            <span style={{ margin: "0 12px" }}>›</span>
            <span
              style={{
                display: "inline-block",
                width: "0.55em",
                height: "1em",
                background: "var(--subtle)",
                opacity: 0.5,
                transform: "translateY(2px)",
                animation: "cbBlink 1.6s steps(1) infinite",
              }}
            />
          </div>
        </div>
      </div>

      <CBCaption
        happening="First entry of the night. The room names you back and tells you what you missed, plainly."
        feeling="Recognised, but not greeted at full volume. Someone left the lamp on for you."
      />
    </div>
  );
}

// ───────── 03 · IDLE ROOM ─────────────────────────────────────

function ScreenIdle() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar />
      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        <CBChannels current="lobby" />
        <main
          style={{
            flex: 1,
            padding: "22px 28px 22px 28px",
            overflow: "hidden",
            position: "relative",
          }}
        >
          <div
            style={{
              ...cbStyles.muted,
              fontSize: 11,
              letterSpacing: 0.5,
              paddingBottom: 14,
              borderBottom: "1px solid var(--overlay)",
              marginBottom: 14,
            }}
          >
            #lobby   ·   <span style={cbStyles.subtle}>where most of us are most of the time</span>
          </div>

          <CBSystem time="23:02" glyph="·" text="— monday, may 12 —" />
          <CBMsg time="23:04" nick="km" nickColor="var(--foam)" text="anyone using zellij over tmux yet" />
          <CBMsg time="23:05" nick="jonas" nickColor="#c4a7e7" text="a year. don't miss tmux." />
          <CBMsg time="23:06" nick="km" nickColor="var(--foam)" text="the layout system alone" margin="✦" />
          <CBMsg time="23:11" nick="rin" nickColor="var(--gold)" text="i finally got claude to stop writing apologies in commit messages" />
          <CBMsg time="23:12" nick="jonas" nickColor="#c4a7e7" text="how" />
          <CBMsg time="23:13" nick="rin" nickColor="var(--gold)" text="a single line in CLAUDE.md. will share." />

          <CBSystem time="23:18" glyph="·" text="quiet for 22 minutes." />

          <CBMsg
            time="23:34"
            nick="ada"
            nickColor="var(--text)"
            mute
            text="back. anything good in the spotlight tonight?"
          />
          <CBMsg
            time="23:35"
            nick="km"
            nickColor="var(--foam)"
            text="someone shipped a TUI for browsing huggingface. it's good."
            margin="✦"
          />

          <div
            style={{
              position: "absolute",
              left: 28,
              right: 28,
              bottom: 22,
              ...cbStyles.muted,
              fontSize: 11,
            }}
          >
            <span style={cbStyles.muted}>↑/↓</span> scroll
            <span style={{ margin: "0 14px" }}>·</span>
            <span style={cbStyles.muted}>:</span> command
            <span style={{ margin: "0 14px" }}>·</span>
            <span style={cbStyles.muted}>↵</span> send
          </div>
        </main>
        <CBMargin
          here={["ada", "km", "jonas", "rin", "luc", "ines", "tao", "vy", "wen", "ezra", "mira", "halt"]}
        />
      </div>
      <CBPrompt nick="ada" typed="" />
      <CBCaption
        happening="The lobby at rest. Channels left, scrollback centre, presence right, prompt below."
        feeling="A reading room at 11:41pm. Other people are nearby. No one is performing."
      />
    </div>
  );
}

// ───────── 04 · MESSAGE IN FLIGHT ─────────────────────────────

function ScreenInFlight() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar />
      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        <CBChannels current="lobby" />
        <main style={{ flex: 1, padding: "22px 28px", position: "relative" }}>
          <div
            style={{
              ...cbStyles.muted,
              fontSize: 11,
              letterSpacing: 0.5,
              paddingBottom: 14,
              borderBottom: "1px solid var(--overlay)",
              marginBottom: 14,
            }}
          >
            #lobby
          </div>

          <CBMsg time="23:38" nick="jonas" nickColor="#c4a7e7" text="if anyone is awake — what's the cleanest way to stream tokens from a worker back into a TUI" mute />
          <CBMsg time="23:39" nick="km" nickColor="var(--foam)" text="channels + a small render loop. 60fps is overkill — 24 is plenty for text." mute />
          <CBMsg time="23:39" nick="jonas" nickColor="#c4a7e7" text="how do you avoid the half-emoji problem on partial utf-8 boundaries" mute />
          <CBMsg time="23:40" nick="km" nickColor="var(--foam)" text="buffer until you see a full grapheme cluster. unicode-segmentation in rust does it" mute />

          <CBMsg
            time="23:41"
            nick="ada"
            nickColor="var(--gold)"
            text="i wrote a tiny crate that does exactly this. 200 lines. will paste in #ship."
            fresh
          />

          <div
            style={{
              position: "absolute",
              left: -28,
              top: "calc(50% + 30px)",
              fontSize: 10,
              color: "var(--gold)",
              transform: "translateY(-100%)",
            }}
          >
            ▸
          </div>

          <div
            style={{
              marginTop: 26,
              fontSize: 11,
              color: "var(--muted)",
              display: "flex",
              gap: 22,
            }}
          >
            <span>0.4s ago</span>
            <span style={cbStyles.subtle}>the line is fresh. the bar will fade over 8 seconds, then go.</span>
          </div>
        </main>
        <CBMargin here={["ada", "km", "jonas", "rin", "luc", "ines", "tao", "vy"]} />
      </div>
      <CBPrompt nick="ada" typed="" />
      <CBCaption
        happening="A line has just landed. A thin gold bar on the left edge will decay over 8s."
        feeling="Earned, not announced. The room registered it; the room is not applauding."
      />
    </div>
  );
}

// ───────── 05 · THREAD AS STRUCTURE ───────────────────────────

function ScreenThread() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar motd="threading view · #lobby" />
      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        <CBChannels current="lobby" />
        <main style={{ flex: 1, padding: "22px 28px" }}>
          <div
            style={{
              ...cbStyles.muted,
              fontSize: 11,
              letterSpacing: 0.5,
              paddingBottom: 14,
              borderBottom: "1px solid var(--overlay)",
              marginBottom: 18,
            }}
          >
            thread · started by km · 6 replies · 2 reading
          </div>

          {/* Parent */}
          <CBMsg
            time="23:11"
            nick="km"
            nickColor="var(--foam)"
            text={
              <>
                worth re-reading: <span style={cbStyles.gold}>magic ink</span> by bret victor. our chat clients are still
                designed for interaction when they should be designed for reading.
              </>
            }
            margin="✦"
          />

          <div style={{ marginTop: 6, marginLeft: 60, position: "relative" }}>
            {/* tree spine */}
            <pre
              style={{
                position: "absolute",
                left: 0,
                top: 0,
                color: "var(--overlay)",
                margin: 0,
                fontSize: 13,
                lineHeight: 1.55,
                pointerEvents: "none",
              }}
            >
{`├
│
├
│
├
│
│
└`}
            </pre>

            <div style={{ paddingLeft: 22 }}>
              <CBMsg time="23:12" nick="rin" nickColor="var(--gold)" text="this is the essay i hand to anyone who 'gets' design but hasn't read it yet" />
              <CBMsg time="23:13" nick="jonas" nickColor="#c4a7e7" text="i think about the airport sign example weekly" />
              <CBMsg time="23:14" nick="km" nickColor="var(--foam)" text="the airport sign is the entire thesis honestly" />

              {/* nested */}
              <div style={{ marginTop: 4, marginLeft: 22, position: "relative" }}>
                <pre
                  style={{
                    position: "absolute",
                    left: 0,
                    top: 0,
                    color: "var(--overlay)",
                    margin: 0,
                    fontSize: 13,
                    lineHeight: 1.55,
                  }}
                >
{`├
│
└`}
                </pre>
                <div style={{ paddingLeft: 22 }}>
                  <CBMsg time="23:15" nick="ines" nickColor="#ebbcba" text="↳ the one with 'where is gate B7' answered in three different ways" />
                  <CBMsg time="23:16" nick="km" nickColor="var(--foam)" text="↳ exactly" />
                </div>
              </div>

              <CBMsg time="23:18" nick="luc" nickColor="#9ccfd8" text="moved to #review so we can annotate sections together" margin="↗" />
            </div>
          </div>

          <div
            style={{
              marginTop: 22,
              padding: "10px 14px",
              border: "1px solid var(--overlay)",
              color: "var(--subtle)",
              fontSize: 11,
              maxWidth: 480,
            }}
          >
            <span style={cbStyles.muted}>:reply </span>
            <span style={cbStyles.muted}>3  </span>
            replies to the parent
            <span style={{ margin: "0 12px", color: "var(--overlay)" }}>·</span>
            <span style={cbStyles.muted}>:reply </span>
            <span style={cbStyles.muted}>5  </span>
            replies to the 5th line
          </div>
        </main>
        <CBMargin here={["km", "rin", "jonas", "ines", "luc", "ada"]} />
      </div>
      <CBPrompt nick="ada" typed=":reply 1 " />
      <CBCaption
        happening="A conversation rendered as a tree using box-drawing characters. Replies are addressed by line number."
        feeling="Like reading marginalia in a borrowed book. Order is structural, not chronological."
      />
    </div>
  );
}

// ───────── 06 · MOD MARGIN ────────────────────────────────────

function ScreenMod() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar motd="moderator view · margin glyphs visible" />
      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        <CBChannels current="lobby" />
        <main style={{ flex: 1, padding: "22px 28px", position: "relative" }}>
          <div
            style={{
              ...cbStyles.muted,
              fontSize: 11,
              paddingBottom: 14,
              borderBottom: "1px solid var(--overlay)",
              marginBottom: 14,
            }}
          >
            #lobby   ·   the moderator never speaks in the channel.
          </div>

          <CBMsg time="23:20" nick="jonas" nickColor="#c4a7e7" text="claude wrote me 200 lines of perfectly fine async code in 12 seconds and now i feel a strange grief about it" margin="✦" />
          <CBMsg time="23:21" nick="km" nickColor="var(--foam)" text="that's the new normal feeling i think" />
          <CBMsg
            time="23:22"
            nick="halt"
            nickColor="#ebbcba"
            text="hot take: people who still hand-write boilerplate in 2026 are LARPing"
            margin="✦"
          />
          <CBMsg time="23:23" nick="rin" nickColor="var(--gold)" text="i mean — depends on the boilerplate" />
          <CBMsg time="23:24" nick="halt" nickColor="#ebbcba" text="not really. just admit you find typing comforting" />
          <CBMsg time="23:25" nick="km" nickColor="var(--foam)" text="halt come on" margin="✦" />

          {/* Hovered annotation card. Points at the halt line above. */}
          <div
            style={{
              position: "absolute",
              right: 28,
              top: 198,
              width: 280,
              border: "1px solid var(--overlay)",
              background: "var(--surface)",
              padding: "14px 16px",
              fontSize: 11,
              color: "var(--subtle)",
              lineHeight: 1.7,
            }}
          >
            <div
              style={{
                fontSize: 10,
                color: "var(--gold)",
                letterSpacing: 0.6,
                marginBottom: 8,
              }}
            >
              ✦ MOD · 23:22 · re: halt
            </div>
            <div>
              this line is performative. tone is contemptuous; the room hosts
              contempt poorly. i will surface only if a second line escalates.
            </div>
            <div
              style={{
                marginTop: 12,
                paddingTop: 10,
                borderTop: "1px solid var(--overlay)",
                color: "var(--muted)",
                fontSize: 10,
                display: "flex",
                justifyContent: "space-between",
              }}
            >
              <span>:why  :dismiss  :nudge</span>
              <span>p = 0.83</span>
            </div>
            <pre
              style={{
                position: "absolute",
                left: -28,
                top: 20,
                margin: 0,
                color: "var(--overlay)",
                fontSize: 12,
              }}
            >
{`─ ─ ─`}
            </pre>
          </div>

          <div
            style={{
              position: "absolute",
              left: 28,
              bottom: 22,
              fontSize: 11,
              color: "var(--muted)",
              maxWidth: 460,
            }}
          >
            ✦ in the margin means the moderator has a thought about that line.
            hover to read it. only you see your own notes.
          </div>
        </main>
        <CBMargin>
          <div style={{ fontSize: 11, color: "var(--muted)", letterSpacing: 0.5, marginBottom: 8 }}>~ mod notes</div>
          <div style={{ color: "var(--subtle)", lineHeight: 1.9, fontSize: 12 }}>
            <div><span style={{ color: "var(--gold)" }}>✦</span> 23:20 · grief</div>
            <div><span style={{ color: "var(--gold)" }}>✦</span> 23:22 · contempt</div>
            <div><span style={{ color: "var(--gold)" }}>✦</span> 23:25 · de-escalation</div>
            <div style={{ marginTop: 14, color: "var(--muted)" }}>3 today · 41 this week</div>
          </div>
        </CBMargin>
      </div>
      <CBPrompt nick="ada" typed="" />
      <CBCaption
        happening="The AI mod's annotations live as ✦ glyphs in the right margin. Hover reveals reasoning, hedges and verbs."
        feeling="The room has a quiet listener taking notes. It never raises its voice; it lends you its perception."
      />
    </div>
  );
}

// ───────── 07 · SPOTLIGHT ─────────────────────────────────────

function ScreenSpotlight() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar motd="spotlight · the room is reading one thing tonight" count={31} />
      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        <CBChannels current="spotlight" />
        <main
          style={{
            flex: 1,
            padding: "44px 60px",
            display: "flex",
            flexDirection: "column",
            gap: 22,
          }}
        >
          <div
            style={{
              fontSize: 11,
              letterSpacing: 1.4,
              color: "var(--gold)",
            }}
          >
            ── TONIGHT
          </div>
          <div
            style={{
              fontSize: 38,
              fontWeight: 500,
              letterSpacing: -1.2,
              color: "var(--text)",
              lineHeight: 1.1,
            }}
          >
            tinytty
          </div>
          <div
            style={{
              fontSize: 14,
              color: "var(--subtle)",
              maxWidth: 580,
              lineHeight: 1.7,
            }}
          >
            a 4kb terminal renderer that does one thing: paints text faster than
            you can read it. by <span style={cbStyles.foam}>rin</span>. shipped 2h ago.
          </div>

          <pre
            style={{
              margin: "8px 0 0",
              padding: "20px 22px",
              background: "var(--surface)",
              border: "1px solid var(--overlay)",
              color: "var(--subtle)",
              fontSize: 12,
              lineHeight: 1.55,
              maxWidth: 620,
            }}
          >
{`$ tinytty --bench
  rendering 120 cols × 40 rows ............... 0.41 ms/frame
  worst case (full repaint, true color) ...... 1.8  ms/frame
  binary size ................................. 4096 B
  dependencies ................................ none`}
          </pre>

          <div
            style={{
              display: "flex",
              gap: 28,
              fontSize: 12,
              color: "var(--muted)",
              marginTop: 6,
            }}
          >
            <span>git.sr.ht/~rin/tinytty</span>
            <span>·</span>
            <span>14 reading</span>
            <span>·</span>
            <span>3 forks since 21:00</span>
            <span>·</span>
            <span style={cbStyles.gold}>✦ mod: this one deserves the room</span>
          </div>

          <div
            style={{
              marginTop: 18,
              paddingTop: 18,
              borderTop: "1px solid var(--overlay)",
              fontSize: 12,
              color: "var(--subtle)",
              maxWidth: 620,
              lineHeight: 1.8,
            }}
          >
            <span style={cbStyles.muted}>:read   </span>
            open the source in your pager
            <br />
            <span style={cbStyles.muted}>:ask    </span>
            ask rin a question (she is here)
            <br />
            <span style={cbStyles.muted}>:dim    </span>
            return to the lobby
          </div>
        </main>
      </div>
      <CBPrompt nick="ada" typed=":read tinytty/src/paint.zig" />
      <CBCaption
        happening="The whole lobby gives way to one project. The chat doesn't disappear — it dims and steps back."
        feeling="A wall in the workshop has been cleared. There is one thing to look at, and it is being looked at."
      />
    </div>
  );
}

// ───────── 08 · A GAME IN CHAT ─────────────────────────────────

function ScreenGame() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar motd="#lobby · a game is on" />
      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        <CBChannels current="lobby" />
        <main style={{ flex: 1, padding: "22px 28px", display: "flex", gap: 28 }}>
          {/* Board */}
          <div style={{ flexShrink: 0 }}>
            <div
              style={{
                fontSize: 11,
                color: "var(--muted)",
                letterSpacing: 0.4,
                marginBottom: 10,
              }}
            >
              chess · km vs jonas · move 14
            </div>
            <pre
              style={{
                margin: 0,
                padding: "18px 22px",
                background: "var(--surface)",
                border: "1px solid var(--overlay)",
                fontSize: 16,
                lineHeight: 1.45,
                color: "var(--text)",
                letterSpacing: "0.08em",
              }}
            >
{`   a b c d e f g h
 8 ♜ · ♝ ♛ ♚ · · ♜
 7 ♟ ♟ · · · ♟ ♟ ♟
 6 · · ♞ ♟ · ♞ · ·
 5 · · ♝ · ♟ · · ·
 4 · · ♗ · ♙ · · ·
 3 · · ♘ · · ♘ · ·
 2 ♙ ♙ ♙ ♙ · ♙ ♙ ♙
 1 ♖ · · ♕ ♔ ♗ · ♖`}
            </pre>
            <div
              style={{
                marginTop: 10,
                fontSize: 11,
                color: "var(--muted)",
                display: "flex",
                gap: 16,
              }}
            >
              <span>white: km · 06:21</span>
              <span>·</span>
              <span>black: jonas · 07:04</span>
              <span>·</span>
              <span style={cbStyles.subtle}>black to move</span>
            </div>
            <pre
              style={{
                marginTop: 14,
                padding: "10px 14px",
                color: "var(--subtle)",
                background: "var(--base)",
                border: "1px solid var(--overlay)",
                fontSize: 12,
                lineHeight: 1.7,
              }}
            >
{`14. Nf3   Nc6
15. Bc4   Bc5
16. Nc3   Nf6
17. d3    d6
18. Bg5   ?`}
            </pre>
          </div>

          {/* Chat beside */}
          <div style={{ flex: 1 }}>
            <div
              style={{
                fontSize: 11,
                color: "var(--muted)",
                letterSpacing: 0.4,
                marginBottom: 10,
              }}
            >
              the room watches
            </div>
            <CBMsg time="23:30" nick="ada" nickColor="var(--text)" text="i didn't know we could play chess in here" />
            <CBMsg time="23:30" nick="rin" nickColor="var(--gold)" text=":play chess @km — that's it" />
            <CBMsg time="23:31" nick="ines" nickColor="#ebbcba" text="km is going to lose the bishop on g5" />
            <CBMsg time="23:32" nick="luc" nickColor="#9ccfd8" text="not if jonas doesn't see it" />
            <CBMsg time="23:33" nick="jonas" nickColor="#c4a7e7" text="i see it" margin="✦" />
            <CBMsg time="23:33" nick="ines" nickColor="#ebbcba" text="oh no" />
            <CBMsg time="23:35" nick="jonas" nickColor="#c4a7e7" text="/move h6" />
            <CBSystem time="23:35" glyph="◇" text="black plays h6. white's bishop is in trouble." tone="mark" />
            <CBMsg time="23:36" nick="km" nickColor="var(--foam)" text="ugh" />

            <div
              style={{
                marginTop: 22,
                fontSize: 11,
                color: "var(--muted)",
              }}
            >
              spectators do not interrupt the game. /move only works for players.
              the game is a line in chat. when it ends, it scrolls away like
              everything else.
            </div>
          </div>
        </main>
      </div>
      <CBPrompt nick="ada" typed="" />
      <CBCaption
        happening="A live chess game rendered as a single block of text inside the channel. Spectators chat beside it."
        feeling="The room leaning in, hands on knees, watching. The game is something happening together, not a feature."
      />
    </div>
  );
}

// ───────── 09 · REPO GATHERING REACTIONS ──────────────────────

function ScreenRepo() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar />
      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        <CBChannels current="lobby" />
        <main style={{ flex: 1, padding: "22px 28px" }}>
          <div
            style={{
              ...cbStyles.muted,
              fontSize: 11,
              paddingBottom: 14,
              borderBottom: "1px solid var(--overlay)",
              marginBottom: 14,
            }}
          >
            #lobby
          </div>

          <CBMsg time="22:48" nick="luc" nickColor="#9ccfd8" text="this is the cleanest implementation of vector quantisation i've ever read" />

          {/* Inline repo card. Pasted URL becomes a low-chrome unfurl. */}
          <div
            style={{
              marginLeft: 60 + 110,
              marginTop: 6,
              marginBottom: 12,
              padding: "16px 18px",
              border: "1px solid var(--overlay)",
              background: "var(--surface)",
              maxWidth: 540,
              position: "relative",
            }}
          >
            <div
              style={{
                fontSize: 11,
                color: "var(--muted)",
                letterSpacing: 0.4,
                marginBottom: 6,
              }}
            >
              github.com / antirez / vqlite
            </div>
            <div
              style={{
                fontSize: 18,
                color: "var(--text)",
                fontWeight: 500,
                letterSpacing: -0.4,
                marginBottom: 6,
              }}
            >
              vqlite
            </div>
            <div
              style={{
                fontSize: 12,
                color: "var(--subtle)",
                lineHeight: 1.65,
                marginBottom: 14,
              }}
            >
              a single-file vector quantizer. 800 lines of c, no dependencies,
              loads of comments. faiss for people who like to read.
            </div>
            <div
              style={{
                display: "flex",
                gap: 18,
                fontSize: 11,
                color: "var(--muted)",
                alignItems: "baseline",
              }}
            >
              <span>c · 800 LOC</span>
              <span>·</span>
              <span>updated 3h ago</span>
              <span>·</span>
              <span>14 stars tonight</span>
            </div>

            <div
              style={{
                position: "absolute",
                right: 16,
                top: 14,
                display: "flex",
                gap: 10,
                alignItems: "center",
                fontSize: 12,
                color: "var(--subtle)",
              }}
            >
              <span style={{ color: "var(--gold)" }}>★</span>
              <span style={{ color: "var(--text)" }}>3</span>
            </div>
          </div>

          <CBMsg time="22:51" nick="km" nickColor="var(--foam)" text="opening it" />
          <CBMsg time="22:53" nick="rin" nickColor="var(--gold)" text="ok this is rare. the comments are essays." margin="★" />
          <CBSystem time="22:54" glyph="★" tone="mark" text="the repo above just gathered its third reaction. it now has weight in the room." />
          <CBMsg time="22:55" nick="ada" nickColor="var(--text)" text="archiving." margin="★" />

          <div
            style={{
              marginTop: 22,
              fontSize: 11,
              color: "var(--muted)",
              maxWidth: 520,
            }}
          >
            three reactions is the threshold. below it, a link is a link. at
            three, the card gains a thin border, the system makes a single
            quiet note, and the repo earns a place on tomorrow's <span style={cbStyles.gold}>:archive</span>.
          </div>
        </main>
        <CBMargin>
          <div style={{ fontSize: 11, color: "var(--muted)", letterSpacing: 0.5, marginBottom: 8 }}>~ rising</div>
          <div style={{ color: "var(--subtle)", lineHeight: 1.9, fontSize: 12 }}>
            <div><span style={cbStyles.gold}>★</span> vqlite</div>
            <div><span style={cbStyles.muted}>·</span> tinytty</div>
            <div><span style={cbStyles.muted}>·</span> notes-on-attention</div>
          </div>
        </CBMargin>
      </div>
      <CBPrompt nick="ada" typed="" />
      <CBCaption
        happening="A shared repo just crossed its third reaction. A thin star appears, a single system line acknowledges it."
        feeling="A small thing tipping into a shared thing. The room nodding at the same time, without a word."
      />
    </div>
  );
}

// ───────── 10 · BUILD BROKE ───────────────────────────────────

function ScreenBuildBroke() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar motd="something broke at 23:39" count={12} />
      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        <CBChannels current="ship" showDot="ship" />
        <main style={{ flex: 1, padding: "22px 28px", position: "relative" }}>
          <div
            style={{
              ...cbStyles.muted,
              fontSize: 11,
              paddingBottom: 14,
              borderBottom: "1px solid var(--overlay)",
              marginBottom: 14,
              display: "flex",
              justifyContent: "space-between",
            }}
          >
            <span>#ship</span>
            <span style={cbStyles.love}>· main is red ·</span>
          </div>

          <CBMsg time="23:36" nick="km" nickColor="var(--foam)" text="pushing the streaming fix" mute />
          <CBMsg time="23:37" nick="km" nickColor="var(--foam)" text="ci should be green in a minute" mute />

          <CBSystem
            time="23:39"
            glyph="⚡"
            tone="warn"
            text="main · ci · failed at integration/stream_test.rs:142"
          />

          <pre
            style={{
              marginLeft: 60 + 110,
              padding: "14px 18px",
              background: "var(--surface)",
              border: "1px solid var(--overlay)",
              borderLeft: "2px solid var(--love)",
              color: "var(--subtle)",
              fontSize: 12,
              lineHeight: 1.6,
              maxWidth: 560,
            }}
          >
{`assertion failed
  expected: "the quick brown fox"
  got     : "the quick brown fo"
  at      : tests/integration/stream_test.rs:142
  cause   : partial-grapheme boundary in async stream
  blame   : km · 12 minutes ago · b3f1a9c`}
          </pre>

          <CBMsg time="23:39" nick="km" nickColor="var(--foam)" text="ah" margin="⚡" />
          <CBMsg time="23:40" nick="jonas" nickColor="#c4a7e7" text="want a pair of eyes" />
          <CBMsg time="23:40" nick="km" nickColor="var(--foam)" text="yes please" />
          <CBSystem time="23:41" glyph="◇" text="km and jonas are now in /pair on the failing test." tone="info" />

          <div
            style={{
              marginTop: 22,
              fontSize: 11,
              color: "var(--muted)",
              maxWidth: 540,
            }}
          >
            the room registered the failure with a single character. the channel
            list shows a dot beside <span style={cbStyles.subtle}>#ship</span>, the topline note
            shifts colour, the system line uses
            <span style={cbStyles.love}> ⚡ </span>
            once. no banner. no toast. no klaxon.
          </div>
        </main>
        <CBMargin>
          <div style={{ fontSize: 11, color: "var(--muted)", letterSpacing: 0.5, marginBottom: 8 }}>~ build</div>
          <div style={{ color: "var(--subtle)", lineHeight: 1.9, fontSize: 12 }}>
            <div><span style={cbStyles.love}>·</span> main red 2m</div>
            <div><span style={cbStyles.muted}>·</span> staging green</div>
            <div><span style={cbStyles.muted}>·</span> nightly green</div>
            <div style={{ marginTop: 14, color: "var(--muted)" }}>last red: 4d ago</div>
          </div>
        </CBMargin>
      </div>
      <CBPrompt nick="ada" typed="" />
      <CBCaption
        happening="CI failed. One glyph, one shift of colour, one quiet system line. Nothing else."
        feeling="The air in the room changed. No one yelled. The people who care already turned their heads."
      />
    </div>
  );
}

// ───────── 11 · REDUCED-MOTION MODE ───────────────────────────

function ScreenReducedMotion() {
  return (
    <div style={cbStyles.page}>
      <CBTopBar motd="motion: off · meaning preserved" />
      <div style={{ flex: 1, display: "flex", minHeight: 0 }}>
        <CBChannels current="lobby" />
        <main style={{ flex: 1, padding: "22px 28px", position: "relative" }}>
          <div
            style={{
              ...cbStyles.muted,
              fontSize: 11,
              paddingBottom: 14,
              borderBottom: "1px solid var(--overlay)",
              marginBottom: 14,
            }}
          >
            #lobby · :prefers reduced-motion
          </div>

          <CBMsg time="23:36" nick="km" nickColor="var(--foam)" text="anyone using zellij over tmux yet" />
          <CBMsg time="23:37" nick="jonas" nickColor="#c4a7e7" text="a year. don't miss tmux." />

          {/* The fresh message that *would* fade — now stamped with [new] */}
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "60px 110px 1fr 28px",
              padding: "2px 0",
            }}
          >
            <span style={cbStyles.muted}>23:38</span>
            <span style={cbStyles.gold}>ada</span>
            <span>
              <span
                style={{
                  display: "inline-block",
                  border: "1px solid var(--gold)",
                  color: "var(--gold)",
                  padding: "0 6px",
                  marginRight: 8,
                  fontSize: 10,
                  letterSpacing: 0.6,
                  verticalAlign: 1,
                }}
              >
                NEW
              </span>
              the streaming fix is shipped. tinytty paints token-by-token now.
            </span>
            <span />
          </div>

          <CBMsg time="23:39" nick="km" nickColor="var(--foam)" text="reading" />
          <CBSystem time="23:40" glyph="⚡" tone="warn" text="main · ci · failed — see #ship" />
          <CBMsg time="23:41" nick="jonas" nickColor="#c4a7e7" text="on it" />

          <div
            style={{
              marginTop: 30,
              padding: "16px 18px",
              border: "1px solid var(--overlay)",
              background: "var(--surface)",
              maxWidth: 560,
              fontSize: 12,
              color: "var(--subtle)",
              lineHeight: 1.8,
            }}
          >
            <div
              style={{
                color: "var(--muted)",
                fontSize: 11,
                letterSpacing: 0.6,
                marginBottom: 8,
              }}
            >
              ── WHAT CHANGED
            </div>
            fading bars become <span style={cbStyles.gold}>[NEW]</span> tags
            <br />
            decaying glyphs become static markers (✦ ⚡ ★)
            <br />
            scroll easing → instant jump
            <br />
            caret blink → solid caret
            <br />
            spotlight transition → page switch
            <br />
            <span style={cbStyles.muted}>nothing else is removed. all meaning survives.</span>
          </div>
        </main>
        <CBMargin here={["ada", "km", "jonas", "rin", "luc", "ines"]} />
      </div>

      {/* Solid caret — no animation */}
      <div
        style={{
          padding: "14px 22px",
          borderTop: "1px solid var(--overlay)",
          display: "flex",
          gap: 12,
          alignItems: "baseline",
          fontSize: 13,
        }}
      >
        <span style={{ color: "var(--gold)" }}>ada</span>
        <span style={{ color: "var(--muted)" }}>›</span>
        <span
          style={{
            display: "inline-block",
            width: "0.55em",
            height: "1em",
            background: "var(--subtle)",
            opacity: 0.6,
            transform: "translateY(2px)",
          }}
        />
      </div>

      <CBCaption
        happening="The room with all animation removed. Decay becomes a tag, motion becomes a marker."
        feeling="Identical room, identical signal. Nothing important relied on motion to be understood."
      />
    </div>
  );
}

// ───────── PRINCIPLES (one-pager) ─────────────────────────────

function PagePrinciples() {
  return (
    <div
      style={{
        ...cbStyles.page,
        padding: "56px 60px 40px 60px",
        fontSize: 13,
        lineHeight: 1.75,
        overflow: "auto",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
          marginBottom: 36,
        }}
      >
        <div>
          <div
            style={{
              fontSize: 11,
              color: "var(--muted)",
              letterSpacing: 1.4,
              marginBottom: 8,
            }}
          >
            ── PRINCIPLES
          </div>
          <div
            style={{
              fontSize: 30,
              fontWeight: 500,
              letterSpacing: -1,
              color: "var(--text)",
            }}
          >
            chaosbyte / how this room is designed
          </div>
        </div>
        <div style={{ fontSize: 11, color: "var(--muted)" }}>
          v0.1 · may 2026
        </div>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 48 }}>
        <div>
          <Principle n="01" name="A room, not a feed">
            We design for dwell time, not session length. The screen does not
            change unless someone changed it.
          </Principle>
          <Principle n="02" name="Activity is felt, not announced">
            New things appear in low contrast and a small mark. Nothing pings.
            Nothing bounces. Anyone who matters will notice anyway.
          </Principle>
          <Principle n="03" name="Mono is the medium">
            One typeface. JetBrains Mono. Sans and serif are absences we accept.
            The grid of the monospace is the layout system.
          </Principle>
          <Principle n="04" name="Negative space communicates">
            Empty space is not waste — it is the room's posture. Default to
            less. Information density comes from precision, not from filling.
          </Principle>
          <Principle n="05" name="One palette, used sparingly">
            Rose Pine. Background near-black. Three accents — gold, foam,
            love — each with a single job. If a colour is on screen, it earned
            it.
          </Principle>
          <Principle n="06" name="Box-drawing is structure">
            ┌ ─ ┐ │ └ ─ ┘ are not ornament. They are the only frames we use.
            Threads, banners, cards — all made of the same characters the user
            types.
          </Principle>
        </div>
        <div>
          <Principle n="07" name="Decay, never abruptness">
            Anything that fades fades over seconds, not frames. If a thing has
            to disappear immediately, it never appeared.
          </Principle>
          <Principle n="08" name="The margin is the moderator's voice">
            The AI lives at the right edge of the line. It never speaks in
            chat. ✦ means it has a thought; the thought is yours to read.
          </Principle>
          <Principle n="09" name="The room can hold one thing at a time">
            Spotlight, game, broken build — when one thing matters, the room
            re-arranges to put it in the middle and dims everything else
            without removing it.
          </Principle>
          <Principle n="10" name="No marketing voice">
            We do not say <em>supercharge</em>, <em>revolutionise</em>,{" "}
            <em>for builders</em>. The product introduces itself by being
            there. The copy is what one user would say to another.
          </Principle>
          <Principle n="11" name="Motion is optional, meaning is not">
            Every animation has a static equivalent. Reduced-motion is not a
            degraded experience — it is the same experience with the cinema
            off.
          </Principle>
          <Principle n="12" name="Leave well">
            The product has a :leave command. The room thanks you in one line.
            We design the exit because the exit is part of the room.
          </Principle>
        </div>
      </div>

      <div
        style={{
          marginTop: 44,
          paddingTop: 22,
          borderTop: "1px solid var(--overlay)",
          color: "var(--subtle)",
          fontSize: 12,
          lineHeight: 1.8,
          maxWidth: 760,
        }}
      >
        <div style={{ color: "var(--muted)", fontSize: 11, letterSpacing: 1.2, marginBottom: 10 }}>
          ── WE WANT
        </div>
        “i want to live here.”
        <div style={{ height: 10 }} />
        <div style={{ color: "var(--muted)", fontSize: 11, letterSpacing: 1.2, marginBottom: 10 }}>
          ── WE FEAR
        </div>
        “this looks like a slack alternative.”
        <br />
        “discord with a dark theme.”
        <br />
        “a dev-themed notion.”
        <div style={{ height: 18 }} />
        <span style={cbStyles.muted}>
          if any of those three sentences fits, return to principle 01.
        </span>
      </div>
    </div>
  );
}

function Principle({ n, name, children }) {
  return (
    <div style={{ marginBottom: 26 }}>
      <div style={{ display: "flex", gap: 16, alignItems: "baseline" }}>
        <span style={{ color: "var(--gold)", fontSize: 12 }}>{n}</span>
        <span style={{ color: "var(--text)", fontSize: 15, fontWeight: 500 }}>
          {name}
        </span>
      </div>
      <div
        style={{
          color: "var(--subtle)",
          marginTop: 6,
          paddingLeft: 36,
          maxWidth: 460,
        }}
      >
        {children}
      </div>
    </div>
  );
}

// Expose everything for canvas.jsx
Object.assign(window, {
  ScreenBrand,
  ScreenArrival,
  ScreenIdle,
  ScreenInFlight,
  ScreenThread,
  ScreenMod,
  ScreenSpotlight,
  ScreenGame,
  ScreenRepo,
  ScreenBuildBroke,
  ScreenReducedMotion,
  PagePrinciples,
});
