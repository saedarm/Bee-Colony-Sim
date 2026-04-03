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
	gridSize     = 100
	tickRate     = 0.4
)

func main() {
	rl.InitWindow(screenWidth, screenHeight, "Bee Colony - ABC Algorithm Visualizer")
	rl.SetTargetFPS(60)

	presets := colony.Presets()
	presetNames := []string{"Balanced", "Plenty", "Famine", "Needle", "Swarm"}
	currentPreset := 0
	showSetup := true

	var sim *colony.Colony
	var terr *terrain.Terrain
	var rend *renderer.Renderer
	var tickTimer float32
	phase := "Initializing"

	for !rl.WindowShouldClose() {
		if showSetup {
			rl.BeginDrawing()
			rl.ClearBackground(rl.NewColor(12, 12, 22, 255))

			rl.DrawText("BEE COLONY", screenWidth/2-180, 80, 48, rl.Gold)
			rl.DrawText("Artificial Bee Colony Algorithm Visualizer", screenWidth/2-220, 140, 18, rl.LightGray)
			rl.DrawText("Select a preset experiment:", screenWidth/2-130, 220, 18, rl.White)

			for i, name := range presetNames {
				cfg := presets[name]
				y := int32(270 + i*80)
				boxColor := rl.NewColor(35, 35, 55, 255)
				textColor := rl.LightGray
				if i == currentPreset {
					boxColor = rl.NewColor(70, 55, 15, 255)
					textColor = rl.Gold
				}
				mouseY := rl.GetMouseY()
				if mouseY >= y && mouseY < y+70 && rl.GetMouseX() >= 200 && rl.GetMouseX() <= screenWidth-200 {
					boxColor = rl.NewColor(55, 45, 25, 255)
					if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
						currentPreset = i
					}
				}
				rl.DrawRectangle(200, y, int32(screenWidth-400), 70, boxColor)
				rl.DrawRectangleLines(200, y, int32(screenWidth-400), 70, rl.NewColor(textColor.R, textColor.G, textColor.B, 80))
				rl.DrawText(fmt.Sprintf("[%d] %s", i+1, name), 220, y+8, 22, textColor)
				rl.DrawText(presetDescription(cfg), 220, y+36, 13, rl.NewColor(170, 170, 170, 200))
			}

			launchY := int32(270 + len(presetNames)*80 + 15)
			launchColor := rl.Gold
			if rl.GetMouseY() >= launchY && rl.GetMouseY() < launchY+50 && rl.GetMouseX() >= screenWidth/2-110 && rl.GetMouseX() <= screenWidth/2+110 {
				launchColor = rl.Yellow
				if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
					showSetup = false
				}
			}
			rl.DrawRectangle(screenWidth/2-110, launchY, 220, 50, rl.NewColor(40, 35, 10, 255))
			rl.DrawRectangleLines(screenWidth/2-110, launchY, 220, 50, launchColor)
			rl.DrawText("Release the Swarm", screenWidth/2-85, launchY+16, 18, launchColor)

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
				cfg := presets[presetNames[currentPreset]]
				terr = terrain.New(cfg.TerrainType, gridSize)
				sim = colony.NewColony(cfg, terr)
				rend = renderer.New()
				rend.BuildTerrainMesh(terr)
				tickTimer = 0
			}
			continue
		}

		// === SIMULATION ===
		dt := rl.GetFrameTime()

		tickTimer += dt
		if tickTimer >= tickRate {
			tickTimer -= tickRate
			sim.Step()
			switch sim.Generation % 3 {
			case 0:
				phase = "Employed Bees"
			case 1:
				phase = "Onlooker Bees"
			case 2:
				phase = "Scout Bees"
			}
		}

		sim.UpdateAnimation(dt)
		rend.UpdateCamera(dt)
		rend.UpdateParticles(dt)

		// Keyboard
		if rl.IsKeyPressed(rl.KeyR) {
			rend.Unload()
			cfg := presets[presetNames[currentPreset]]
			terr = terrain.New(cfg.TerrainType, gridSize)
			sim = colony.NewColony(cfg, terr)
			rend = renderer.New()
			rend.BuildTerrainMesh(terr)
			tickTimer = 0
		}
		for i := range presetNames {
			if rl.IsKeyPressed(int32(rl.KeyOne) + int32(i)) {
				rend.Unload()
				currentPreset = i
				cfg := presets[presetNames[currentPreset]]
				terr = terrain.New(cfg.TerrainType, gridSize)
				sim = colony.NewColony(cfg, terr)
				rend = renderer.New()
				rend.BuildTerrainMesh(terr)
				tickTimer = 0
			}
		}
		if rl.IsKeyPressed(rl.KeyEscape) {
			showSetup = true
			continue
		}

		// === DRAW ===
		rl.BeginDrawing()
		rl.ClearBackground(rl.NewColor(15, 20, 35, 255))

		rl.BeginMode3D(rend.Camera)
		rend.DrawTerrain()
		rend.DrawHive(terr)
		rend.DrawFoods(sim.Foods, terr)
		rend.DrawBees(sim.Bees, terr)
		rend.DrawParticles()
		rl.EndMode3D()

		rend.DrawHUD(sim, presetNames[currentPreset], phase)

		rl.EndDrawing()
	}

	if rend != nil {
		rend.Unload()
	}
	rl.CloseWindow()
}

func presetDescription(cfg colony.Config) string {
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
