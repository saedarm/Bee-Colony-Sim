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

	paused := false
	simSpeed := float32(1.0)
	baseTick := float32(1.5) // much slower base tick

	for !rl.WindowShouldClose() {
		if showSetup {
			rl.BeginDrawing()
			rl.ClearBackground(rl.NewColor(12, 12, 22, 255))
			rl.DrawText("BEE COLONY", screenWidth/2-180, 60, 48, rl.Gold)
			rl.DrawText("Artificial Bee Colony Algorithm Visualizer", screenWidth/2-220, 120, 18, rl.LightGray)
			rl.DrawText("Select a preset experiment:", screenWidth/2-130, 190, 18, rl.White)

			for i, name := range presetNames {
				cfg := presets[name]
				y := int32(240 + i*75)
				boxC := rl.NewColor(35, 35, 55, 255)
				txtC := rl.LightGray
				if i == currentPreset {
					boxC = rl.NewColor(70, 55, 15, 255)
					txtC = rl.Gold
				}
				my := rl.GetMouseY()
				if my >= y && my < y+65 && rl.GetMouseX() >= 200 && rl.GetMouseX() <= screenWidth-200 {
					boxC = rl.NewColor(55, 45, 25, 255)
					if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
						currentPreset = i
					}
				}
				rl.DrawRectangle(200, y, int32(screenWidth-400), 65, boxC)
				rl.DrawRectangleLines(200, y, int32(screenWidth-400), 65, rl.NewColor(txtC.R, txtC.G, txtC.B, 80))
				rl.DrawText(fmt.Sprintf("[%d] %s", i+1, name), 220, y+6, 22, txtC)
				rl.DrawText(presetDesc(cfg), 220, y+34, 13, rl.NewColor(170, 170, 170, 200))
			}

			ly := int32(240 + len(presetNames)*75 + 12)
			lc := rl.Gold
			if rl.GetMouseY() >= ly && rl.GetMouseY() < ly+50 && rl.GetMouseX() >= screenWidth/2-110 && rl.GetMouseX() <= screenWidth/2+110 {
				lc = rl.Yellow
				if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
					showSetup = false
				}
			}
			rl.DrawRectangle(screenWidth/2-110, ly, 220, 50, rl.NewColor(40, 35, 10, 255))
			rl.DrawRectangleLines(screenWidth/2-110, ly, 220, 50, lc)
			rl.DrawText("Release the Swarm", screenWidth/2-85, ly+16, 18, lc)

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
				paused = false
				simSpeed = 1.0
			}
			continue
		}

		dt := rl.GetFrameTime()
		tickRate := baseTick / simSpeed

		// === INPUT ===
		if rl.IsKeyPressed(rl.KeySpace) {
			paused = !paused
		}
		if rl.IsKeyPressed(rl.KeyEqual) || rl.IsKeyPressed(rl.KeyKpAdd) {
			simSpeed *= 1.5
			if simSpeed > 10 {
				simSpeed = 10
			}
		}
		if rl.IsKeyPressed(rl.KeyMinus) || rl.IsKeyPressed(rl.KeyKpSubtract) {
			simSpeed /= 1.5
			if simSpeed < 0.1 {
				simSpeed = 0.1
			}
		}
		if rl.IsKeyPressed(rl.KeyT) {
			rend.ShowTrails = !rend.ShowTrails
		}
		if rl.IsKeyPressed(rl.KeyI) {
			rend.ShowFoodInfo = !rend.ShowFoodInfo
		}

		// === SIMULATION ===
		if !paused {
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
			sim.UpdateEvents(dt)
		}

		rend.UpdateCamera(dt)
		rend.UpdateParticles(dt) // particles still animate when paused for visual continuity

		tickProgress := tickTimer / tickRate
		if tickProgress > 1 {
			tickProgress = 1
		}

		// Restart
		if rl.IsKeyPressed(rl.KeyR) {
			rend.Unload()
			cfg := presets[presetNames[currentPreset]]
			terr = terrain.New(cfg.TerrainType, gridSize)
			sim = colony.NewColony(cfg, terr)
			rend = renderer.New()
			rend.BuildTerrainMesh(terr)
			tickTimer = 0
			paused = false
		}
		// Preset switch
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
				paused = false
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
		rend.DrawFoods(sim.Foods, terr, sim.AbandonLimit)
		rend.DrawBees(sim.Bees, terr)
		rend.DrawScoutPaths(sim.ScoutEvents, terr)
		rend.DrawParticles()
		rl.EndMode3D()

		rend.DrawHUD(sim, presetNames[currentPreset], phase, paused, simSpeed, tickProgress)
		rend.DrawFoodInfoOverlay(sim.Foods, terr, sim.AbandonLimit)

		rl.EndDrawing()
	}

	if rend != nil {
		rend.Unload()
	}
	rl.CloseWindow()
}

func presetDesc(cfg colony.Config) string {
	tn := "Unknown"
	switch cfg.TerrainType {
	case terrain.Rastrigin:
		tn = "Rastrigin (spiky, many local optima)"
	case terrain.Ackley:
		tn = "Ackley (sharp global min, bumpy plateau)"
	case terrain.Rosenbrock:
		tn = "Rosenbrock (narrow curved valley)"
	case terrain.PerlinNoise:
		tn = "Perlin Noise (organic rolling hills)"
	case terrain.RandomPeaks:
		tn = "Random Peaks (scattered gaussians)"
	}
	return fmt.Sprintf("%d bees | %d food | abandon:%d | %s", cfg.NumBees, cfg.NumFoods, cfg.AbandonLimit, tn)
}
