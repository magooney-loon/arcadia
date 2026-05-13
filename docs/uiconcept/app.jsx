/* App root — routing + Tweaks */

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "accent": "citrus",
  "density": "compact",
  "showLiveDot": true,
  "showStatusBar": true
}/*EDITMODE-END*/;

const ACCENT_PRESETS = {
  citrus:  { accent: "oklch(0.86 0.17 128)", dim: "oklch(0.72 0.14 128 / 0.18)", glow: "oklch(0.86 0.17 128 / 0.28)" },
  cyan:    { accent: "oklch(0.84 0.13 200)", dim: "oklch(0.72 0.12 200 / 0.18)", glow: "oklch(0.84 0.13 200 / 0.28)" },
  amber:   { accent: "oklch(0.84 0.16 75)",  dim: "oklch(0.72 0.13 75 / 0.18)",  glow: "oklch(0.84 0.16 75 / 0.28)" },
  magenta: { accent: "oklch(0.76 0.18 330)", dim: "oklch(0.68 0.15 330 / 0.18)", glow: "oklch(0.76 0.18 330 / 0.28)" },
};

const App = () => {
  const [view, setView] = useState("overview");
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);

  // Apply tweaks to CSS variables
  useEffect(() => {
    const preset = ACCENT_PRESETS[t.accent] || ACCENT_PRESETS.citrus;
    document.documentElement.style.setProperty("--accent", preset.accent);
    document.documentElement.style.setProperty("--accent-dim", preset.dim);
    document.documentElement.style.setProperty("--accent-glow", preset.glow);
  }, [t.accent]);

  useEffect(() => {
    document.documentElement.style.setProperty("--row-padding", t.density === "comfortable" ? "12px" : "8px");
  }, [t.density]);

  const views = {
    overview:   <OverviewView/>,
    blocks:     <BlocksView/>,
    txs:        <TxsView/>,
    transfers:  <TransfersView/>,
    traces:     <TracesView/>,
    crosschain: <CrossChainView/>,
    fx:         <FXView/>,
    agents:     <AgentsView/>,
    jobs:       <JobsView/>,
    graph:      <GraphView/>,
  };

  return (
    <div>
      <div className="app" data-screen-label={"Arcadia · " + view}>
        <Logo/>
        <Topbar headBlock={STATS.block_height} tps={STATS.tps} blockMs={STATS.block_time_ms}/>
        <Sidebar active={view} onNav={setView}/>
        <main className="main">
          {views[view]}
        </main>
        {t.showStatusBar && <StatusBar/>}
      </div>

      <TweaksPanel title="Tweaks">
        <TweakSection label="Accent">
          <TweakRadio
            label="Activity color"
            value={t.accent}
            onChange={(v) => setTweak("accent", v)}
            options={["citrus", "cyan", "amber", "magenta"]}
          />
        </TweakSection>

        <TweakSection label="Layout">
          <TweakRadio
            label="Density"
            value={t.density}
            onChange={(v) => setTweak("density", v)}
            options={["compact", "comfortable"]}
          />
          <TweakToggle
            label="Status bar"
            value={t.showStatusBar}
            onChange={(v) => setTweak("showStatusBar", v)}
          />
        </TweakSection>

        <TweakSection label="Jump to view">
          <TweakSelect
            label="Section"
            value={view}
            options={["overview","blocks","txs","transfers","traces","crosschain","fx","agents","jobs","graph"]}
            onChange={setView}
          />
        </TweakSection>
      </TweaksPanel>
    </div>
  );
};

ReactDOM.createRoot(document.getElementById("root")).render(<App/>);
