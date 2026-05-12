/* canvas.jsx — assembles all artboards inside the design canvas */

const {
  DesignCanvas, DCSection, DCArtboard,
  ScreenBrand, ScreenArrival, ScreenIdle, ScreenInFlight, ScreenThread,
  ScreenMod, ScreenSpotlight, ScreenGame, ScreenRepo, ScreenBuildBroke,
  ScreenReducedMotion, PagePrinciples,
} = window;

const ROOM_W = 1180;
const ROOM_H = 760;

function CBApp() {
  return (
    <DesignCanvas>
      <DCSection id="brand" title="01 · brand mark"
        subtitle="wordmark · monogram · ascii banner · palette · type · manifesto">
        <DCArtboard id="brand" label="A · the mark" width={1180} height={820}>
          <ScreenBrand />
        </DCArtboard>
      </DCSection>

      <DCSection id="states" title="room · everyday states"
        subtitle="02 arrival · 03 idle room · 04 a message just landed · 11 reduced-motion">
        <DCArtboard id="arrival" label="02 · the arrival" width={ROOM_W} height={ROOM_H}>
          <ScreenArrival />
        </DCArtboard>
        <DCArtboard id="idle" label="03 · the idle room" width={ROOM_W} height={ROOM_H}>
          <ScreenIdle />
        </DCArtboard>
        <DCArtboard id="inflight" label="04 · a message in flight" width={ROOM_W} height={ROOM_H}>
          <ScreenInFlight />
        </DCArtboard>
        <DCArtboard id="reduced" label="11 · reduced-motion mode" width={ROOM_W} height={ROOM_H}>
          <ScreenReducedMotion />
        </DCArtboard>
      </DCSection>

      <DCSection id="structure" title="room · structure of attention"
        subtitle="05 thread · 06 mod margin">
        <DCArtboard id="thread" label="05 · a thread as structure" width={ROOM_W} height={ROOM_H}>
          <ScreenThread />
        </DCArtboard>
        <DCArtboard id="mod" label="06 · the mod margin" width={ROOM_W} height={ROOM_H}>
          <ScreenMod />
        </DCArtboard>
      </DCSection>

      <DCSection id="events" title="room · when something is happening"
        subtitle="07 spotlight · 08 a game in chat · 09 a repo gathers reactions · 10 a build broke">
        <DCArtboard id="spotlight" label="07 · the spotlight" width={ROOM_W} height={ROOM_H}>
          <ScreenSpotlight />
        </DCArtboard>
        <DCArtboard id="game" label="08 · a game in chat" width={ROOM_W} height={ROOM_H}>
          <ScreenGame />
        </DCArtboard>
        <DCArtboard id="repo" label="09 · a repo with gravity" width={ROOM_W} height={ROOM_H}>
          <ScreenRepo />
        </DCArtboard>
        <DCArtboard id="broke" label="10 · a build broke" width={ROOM_W} height={ROOM_H}>
          <ScreenBuildBroke />
        </DCArtboard>
      </DCSection>

      <DCSection id="principles" title="principles"
        subtitle="one page · distilled from the mockups">
        <DCArtboard id="principles" label="12 · how this room is designed" width={1180} height={960}>
          <PagePrinciples />
        </DCArtboard>
      </DCSection>
    </DesignCanvas>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<CBApp />);
