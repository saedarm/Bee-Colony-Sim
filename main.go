package main

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/saedarm/bee-colony/colony"
	"github.com/saedarm/bee-colony/renderer"
	"github.com/saedarm/bee-colony/terrain"
)

const (
	screenWidth  = 1280
	screenHeight = 800
	gridSize     = 80
	tickRate     = 0.5 // seconds between algorithm steps
)

func main() {
	rl.InitWindow(screenWidth, screenHeight, "Bee Colony - ABC Algorithm Visualizer")
	rl.SetTargetFPS(60)

	// State
	presets := colony.Presets()
	presetNames := []string{"Balanced", "Plenty", "Famine", "Needle", "Swarm"}
	currentPreset := 0
	showSetup := true

	var sim *colony.Colony
	var terr *terrain.Terrain
	var rend *renderer.Renderer
	var tickTimer float32
	phase := "Initializing"

	// Setup screen loop
	for !rl.WindowShouldClose() {
		if showSetup {
			rl.BeginDrawing()
			rl.ClearBackground(rl.NewColor(15, 15, 25, 255))

			// Title
			rl.DrawText("BEE COLONY", screenWidth/2-180, 80, 48, rl.Gold)
			rl.DrawText("Artificial Bee Colony Algorithm Visualizer", screenWidth/2-220, 140, 18, rl.LightGray)

			// Preset selection
			rl.DrawText("Select a preset experiment:", screenWidth/2-130, 220, 18, rl.White)

			for i, name := range presetNames {
				cfg := presets[name]
				y := int32(280 + i*90)
				boxColor := rl.NewColor(40, 40, 60, 255)
				textColor := rl.LightGray

				if i == currentPreset {
					boxColor = rl.NewColor(80, 60, 20, 255)
					textColor = rl.Gold
				}

				// Check mouse hover
				mouseY := rl.GetMouseY()
				if mouseY >= y && mouseY < y+80 && rl.GetMouseX() >= 200 && rl.GetMouseX() <= screenWidth-200 {
					boxColor = rl.NewColor(60, 50, 30, 255)
					if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
						currentPreset = i
					}
				}

				rl.DrawRectangle(200, y, int32(screenWidth-400), 80, boxColor)
				rl.DrawRectangleLines(200, y, int32(screenWidth-400), 80, rl.NewColor(textColor.R, textColor.G, textColor.B, 100))

				rl.DrawText(fmt.Sprintf("[%d] %s", i+1, name), 220, y+10, 22, textColor)

				desc := presetDescription(name, cfg)
				rl.DrawText(desc, 220, y+38, 14, rl.NewColor(180, 180, 180, 200))
			}

			// Launch button
			launchY := int32(280 + len(presetNames)*90 + 20)
			launchColor := rl.Gold
			if rl.GetMouseY() >= launchY && rl.GetMouseY() < launchY+50 && rl.GetMouseX() >= screenWidth/2-100 && rl.GetMouseX() <= screenWidth/2+100 {
				launchColor = rl.Yellow
				if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
					showSetup = false
				}
			}
			rl.DrawRectangle(screenWidth/2-100, launchY, 200, 50, rl.NewColor(40, 40, 10, 255))
			rl.DrawRectangleLines(screenWidth/2-100, launchY, 200, 50, launchColor)
			rl.DrawText("Release the Swarm", screenWidth/2-82, launchY+16, 18, launchColor)

			// Keyboard shortcuts
			for i := range presetNames {
				if rl.IsKeyPressed(int32(rl.KeyOne) + int32(i)) {
					currentPreset = i
				}
			}
			if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeySpace) {
				showSetup = false
			}

			rl.EndDrawing()

			if !showSetup {
				// Initialize simulation
				cfg := presets[presetNames[currentPreset]]
				terr = terrain.New(cfg.TerrainType, gridSize)
				sim = colony.NewColony(cfg, terr)
				rend = renderer.New()
				rend.BuildTerrainMesh(terr)
				tickTimer = 0
			}
			continue
		}

		// === SIMULATION LOOP ===
		dt := rl.GetFrameTime()

		// Algorithm tick
		tickTimer += dt
		if tickTimer >= tickRate {
			tickTimer -= tickRate

			// Run one ABC generation
			phase = "Employed Bees"
			sim.Step()

			// Cycle phase display (visual only, the Step does all three)
			gen := sim.Generation
			switch gen % 3 {
			case 0:
				phase = "Employed Bees"
			case 1:
				phase = "Onlooker Bees"
			case 2:
				phase = "Scout Bees"
			}
		}

		// Animate bees smoothly
		sim.UpdateAnimation(dt)

		// Camera
		rend.UpdateCamera(dt)

		// Keyboard controls
		if rl.IsKeyPressed(rl.KeyR) {
			// Restart with same preset
			cfg := presets[presetNames[currentPreset]]
			terr = terrain.New(cfg.TerrainType, gridSize)
			sim = colony.NewColony(cfg, terr)
			rend = renderer.New()
			rend.BuildTerrainMesh(terr)
			tickTimer = 0
		}

		// Switch presets with number keys
		for i := range presetNames {
			if rl.IsKeyPressed(int32(rl.KeyOne) + int32(i)) {
				currentPreset = i
				cfg := presets[presetNames[currentPreset]]
				terr = terrain.New(cfg.TerrainType, gridSize)
				sim = colony.NewColony(cfg, terr)
				rend = renderer.New()
				rend.BuildTerrainMesh(terr)
				tickTimer = 0
			}
		}

		// Back to setup
		if rl.IsKeyPressed(rl.KeyEscape) {
			showSetup = true
			continue
		}

		// Toggle auto-orbit
		if rl.IsKeyPressed(rl.KeyO) {
			rend.AutoOrbit = !rend.AutoOrbit
		}

		// === DRAW ===
		rl.BeginDrawing()
		rl.ClearBackground(rl.NewColor(10, 10, 20, 255))

		rl.BeginMode3D(rend.Camera)

		rend.DrawTerrain()
		rend.DrawHive(terr)
		rend.DrawFoods(sim.Foods, terr)
		rend.DrawBees(sim.Bees, terr)

		// Grid floor reference
		rl.DrawGrid(20, 2.0)

		rl.EndMode3D()

		rend.DrawHUD(sim, presetNames[currentPreset], phase)

		rl.EndDrawing()
	}

	rl.CloseWindow()
}

func presetDescription(name string, cfg colony.Config) string {
	terrName := "Unknown"
	switch cfg.TerrainType {
	case terrain.Rastrigin:
		terrName = "Rastrigin (spiky, many local optima)"
	case terrain.Ackley:
		terrName = "Ackley (sharp global min, bumpy plateau)"
	case terrain.Rosenbrock:
		terrName = "Rosenbrock (narrow curved valley)"
	case terrain.PerlinNoise:
		terrName = "Perlin Noise (organic rolling hills)"
	case terrain.RandomPeaks:
		terrName = "Random Peaks (scattered gaussians)"
	}

	return fmt.Sprintf("%d bees | %d food sources | abandon limit %d | %s",
		cfg.NumBees, cfg.NumFoods, cfg.AbandonLimit, terrName)
}
