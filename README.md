# Bee Colony - ABC Algorithm Visualizer

A 3D visualization of the **Artificial Bee Colony (ABC)** optimization algorithm built with Go and [raylib-go](https://github.com/gen2brain/raylib-go).

Watch employed, onlooker, and scout bees swarm across mathematical landscapes, picking up geometric food sources as the colony converges on optimal solutions.

## The Algorithm

The ABC algorithm (Karaboga, 2005) simulates honey bee foraging with three phases per generation:

1. **Employed Bees** — Each employed bee exploits a known food source, generating neighbor solutions. If the neighbor is better, the bee moves there. Otherwise, the source's trial counter increments.

2. **Onlooker Bees** — Onlookers use roulette-wheel selection (probability proportional to fitness) to choose which food source to investigate. Better sources attract more onlookers. This is the exploitation mechanism.

3. **Scout Bees** — When a food source's trial counter exceeds the abandonment limit (or nectar runs out), the assigned bee becomes a scout and flies off to discover a completely random new source. This is the exploration mechanism.

The tension between exploitation (onlookers piling onto good sources) and exploration (scouts discovering new territory) is what makes the emergent behavior fascinating to watch.

## What You'll See

- **3D terrain** generated from mathematical benchmark functions (Rastrigin, Ackley, Rosenbrock) or procedural noise
- **Gold bees** (employed) flying purposefully to assigned food sources
- **Blue bees** (onlookers) drifting toward the most promising sources
- **Red bees** (scouts) zipping erratically when sources are exhausted
- **Geometric food sources** (cubes, pyramids, octahedra, spheres, cylinders) that shrink and fade as nectar depletes
- **A central hive** where bees originate
- **Live HUD** showing generation count, best fitness, active/exhausted sources

## Presets

| Preset | Bees | Food | Abandon Limit | Terrain | Story |
|--------|------|------|---------------|---------|-------|
| Balanced | 30 | 10 | 20 | Rastrigin | Even exploration/exploitation |
| Plenty | 15 | 25 | 30 | Perlin Noise | Few bees, lots of food |
| Famine | 40 | 5 | 10 | Ackley | Big colony, scarce resources |
| Needle | 25 | 15 | 25 | Rastrigin | One great source among many mediocre |
| Swarm | 60 | 12 | 15 | Random Peaks | Massive colony spectacle |

## Prerequisites

### Windows
- [Go 1.22+](https://go.dev/dl/)
- C compiler: [MinGW-w64](https://www.mingw-w64.org/) or [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)

### macOS
- Go 1.22+
- Xcode or Command Line Tools (`xcode-select --install`)

### Linux
```bash
# Ubuntu/Debian
sudo apt-get install libgl1-mesa-dev libxi-dev libxcursor-dev libxrandr-dev libxinerama-dev libwayland-dev libxkbcommon-dev

# Fedora
sudo dnf install mesa-libGL-devel libXi-devel libXcursor-devel libXrandr-devel libXinerama-devel wayland-devel libxkbcommon-devel
```

## Build & Run

```bash
git clone https://github.com/saedarm/bee-colony.git
cd bee-colony
go mod tidy
go run .
```

First build takes a few minutes (raylib C source compiles with the bindings).

## Controls

| Key/Action | Effect |
|------------|--------|
| **1-5** | Switch preset (works on setup screen and during sim) |
| **Enter/Space** | Launch simulation from setup screen |
| **R** | Restart with same preset (re-roll randomness) |
| **O** | Toggle auto-orbit camera |
| **Right-click drag** | Manual camera orbit |
| **Scroll wheel** | Zoom in/out |
| **Escape** | Back to setup screen |

## Project Structure

```
bee-colony/
├── main.go              # Entry point, setup screen, simulation loop
├── colony/
│   └── colony.go        # ABC algorithm: bees, food sources, three-phase step
├── terrain/
│   └── terrain.go       # Fitness landscapes: Rastrigin, Ackley, Perlin, etc.
├── renderer/
│   └── renderer.go      # Raylib 3D rendering: terrain mesh, bees, food, HUD
├── go.mod
└── README.md
```

## Blog Post Ideas

- "Watching Bees Think: Visualizing the ABC Algorithm in 3D with Go and Raylib"
- Walk through each preset as a narrative experiment
- GIFs of swarm behavior on different landscapes
- Discussion of exploration vs. exploitation tradeoffs
- Comparison: what happens with high vs. low abandonment limits?

## License

MIT

## Author

S.A. Routh — [GitHub](https://github.com/saedarm) | [Medium](https://medium.com/@s.a.routh)
