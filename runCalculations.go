package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func newRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/run40ktestsim", initialiseCalculation).Methods("GET")
	return r
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	r := newRouter()
	err := http.ListenAndServe(":22222", r)
	if err != nil {
		panic(err.Error())
	}
}

func initialiseCalculation(w http.ResponseWriter, r *http.Request) {
	var weapons []WeaponProfile
	var targets []TargetProfile
	/*weapons = append(weapons, WeaponProfile{
		Name:     "Chainswords",
		Attacks:  `24`,
		BSWS:     3,
		Strength: `4`,
		AP:       1,
		Damage:   `1`,
	})
	weapons = append(weapons, WeaponProfile{
		Name:     "Heavy Melee",
		Attacks:  `6`,
		BSWS:     3,
		Strength: `8`,
		AP:       2,
		Damage:   `2`,
	})
	weapons = append(weapons, WeaponProfile{
		Name:     "Close Combat",
		Attacks:  `6`,
		BSWS:     3,
		Strength: `4`,
		AP:       0,
		Damage:   `1`,
	}) */
	weapons = append(weapons, WeaponProfile{
		Name:     "Demon Hammer",
		Attacks:  `5 + D3`,
		BSWS:     3,
		Strength: `8 + D3`,
		AP:       2,
		Damage:   `2`,
	})
	targets = append(targets, TargetProfile{
		Name:             "Rhino",
		Toughness:        9,
		Save:             3,
		InvulnerableSave: 0,
		Wounds:           10,
		FNP:              0,
	})
	targets = append(targets, TargetProfile{
		Name:             "Land Raider",
		Toughness:        12,
		Save:             2,
		InvulnerableSave: 0,
		Wounds:           16,
		FNP:              0,
	})
	targets = append(targets, TargetProfile{
		Name:             "Elite Squad",
		Toughness:        6,
		Save:             2,
		InvulnerableSave: 4,
		Wounds:           3,
		FNP:              0,
	})
	targets = append(targets, TargetProfile{
		Name:             "Infantry",
		Toughness:        4,
		Save:             4,
		InvulnerableSave: 0,
		Wounds:           1,
		FNP:              0,
	})
	var output []Output
	for _, weapon := range weapons {
		for _, target := range targets {
			output = append(output, runCalculations(weapon, target, NewCalculationOptions(func(co *CalculationOptions) {
				//co.sustainedHits = 1
				co.lethalHits = true
				co.critHit = 5
				co.rerollWounds = true
				co.woundsToReroll = 0
				co.devWounds = true
			})))
		}
	}
	responseBytes, err := json.Marshal(output)
	fmt.Println(string(responseBytes))
	checkErr(err)
	w.Write(responseBytes)
}

func calculateInput(input string) float64 {
	inputSlice := strings.Split(strings.ToLower(input), "+")
	value := float64(0)
	for _, element := range inputSlice {
		element = strings.Trim(element, " ")
		reDice := regexp.MustCompile(`(\d?d\d+)`)
		reFlat := regexp.MustCompile(`(\d+)`)
		if reDice.Match([]byte(element)) {
			reMultiplier := regexp.MustCompile(`^\d+`)
			multiplier := float64(1)
			if reMultiplier.Match([]byte(element)) {
				multiplierFound, err := strconv.Atoi(strings.Join(reMultiplier.FindAllString(element, 1), ""))
				if err != nil {
					printError(element)
				} else {
					multiplier = float64(multiplierFound)
				}
				element, _ = strings.CutPrefix(element, strconv.Itoa(int(multiplier)))
			}
			value += convertDiceToAverage(element) * multiplier
		} else if reFlat.Match([]byte(element)) {
			val, err := strconv.Atoi(element)
			if err != nil {
				printError(element)
			} else {
				value += float64(val)
			}
		}

	}
	return value
}

func printError(element string) {
	fmt.Println("Dodgy value got through:", element)
}

func convertDiceToAverage(dice string) float64 {
	valueStripped, _ := strings.CutPrefix(dice, "d")
	value, err := strconv.Atoi(valueStripped)
	if err != nil {
		printError(strconv.Itoa(value))
	} else {
		value += 1
		return float64(float64(value) / float64(2))
	}
	return 0
}

func runCalculations(weapon WeaponProfile, target TargetProfile, opts *CalculationOptions) Output {
	var diceSides float64 = 6
	var lethals float64 = 0
	var devs float64 = 0
	weapAttacks := calculateInput(weapon.Attacks)
	weapStrength := calculateInput(weapon.Strength)
	weapDamage := calculateInput(weapon.Damage)
	var output Output = *NewOutput((func(o *Output) { o.Name = weapon.Name + " vs " + target.Name }))
	hits, critHits := calculateExpectedSuccesses(weapAttacks, calculateHitRoll(weapon.BSWS), diceSides, adjustmodifier(opts.hitModifier), opts.critHit)
	if opts.rerollHits {
		var rerolledHits float64
		var rerolledcritHits float64
		if opts.hitsToReroll > 0 {
			rerolledHits, rerolledcritHits = calculateExpectedSuccesses(weapAttacks*(1/diceSides), calculateHitRoll(weapon.BSWS), diceSides, adjustmodifier(opts.hitModifier), opts.critHit)
		} else if opts.hitsToReroll < 0 {
			rerolledHits, rerolledcritHits = calculateExpectedSuccesses(weapAttacks-critHits, calculateHitRoll(weapon.BSWS), diceSides, adjustmodifier(opts.hitModifier), opts.critHit)
			hits = 0
		} else if opts.hitsToReroll == 0 {
			rerolledHits, rerolledcritHits = calculateExpectedSuccesses(weapAttacks-hits-critHits, calculateHitRoll(weapon.BSWS), diceSides, adjustmodifier(opts.hitModifier), opts.critHit)
		}
		hits += rerolledHits
		critHits += rerolledcritHits
		output.RerolledHits = rerolledHits
		output.RerolledCritHits = rerolledcritHits
	}

	if opts.lethalHits {
		lethals = critHits
	} else {
		hits += critHits
	}
	if opts.sustainedHits > 0 {
		hits += critHits * opts.sustainedHits
	}
	output.Hits = hits
	wounds, critWounds := calculateExpectedSuccesses(hits, calculateWoundRoll(weapStrength, target.Toughness), diceSides, adjustmodifier(opts.woundModifier), opts.critWound)
	if opts.rerollWounds {
		var rerolledWounds float64
		var rerolledcritWounds float64
		if opts.woundsToReroll > 0 {
			rerolledWounds, rerolledcritWounds = calculateExpectedSuccesses(hits*(1/diceSides), calculateWoundRoll(weapStrength, target.Toughness), diceSides, adjustmodifier(opts.woundModifier), opts.critWound)
		} else if opts.woundsToReroll < 0 {
			rerolledWounds, rerolledcritWounds = calculateExpectedSuccesses(hits-critWounds, calculateWoundRoll(weapStrength, target.Toughness), diceSides, adjustmodifier(opts.woundModifier), opts.critWound)
			wounds = 0
		} else if opts.woundsToReroll == 0 {
			rerolledWounds, rerolledcritWounds = calculateExpectedSuccesses(hits-wounds-critWounds, calculateWoundRoll(weapStrength, target.Toughness), diceSides, adjustmodifier(opts.woundModifier), opts.critWound)
		}
		wounds += rerolledWounds
		critWounds += rerolledcritWounds
		output.RerolledWounds = rerolledWounds
		output.RerolledWoundCrits = rerolledcritWounds
	}
	if opts.devWounds {
		devs = critWounds
	} else {
		wounds += critWounds
	}
	wounds += lethals
	unsuccessfulSaves, _ := calculateExpectedSuccesses(wounds, calculateSave(target, weapon.AP, false, opts.saveModifier), diceSides, float64(0), float64(0))
	unsuccessfulSaves = wounds - unsuccessfulSaves
	damage := (unsuccessfulSaves + devs) * weapDamage

	if opts.sustainedHits > 0 {
		output.SustainedHits = critHits * opts.sustainedHits
	}
	if opts.lethalHits {
		output.LethalHits = lethals
	}
	output.Wounds = wounds
	if opts.devWounds {
		output.DevWounds = devs * weapDamage
	}
	output.FailedSaves = unsuccessfulSaves
	output.Damage = damage
	if target.FNP > 0 {
		fnp := target.FNP
		if target.FNP < 2 {
			fnp = 2
		}
		fnp, _ = calculateExpectedSuccesses(damage, fnp, diceSides, float64(0), float64(0))
		damage -= fnp
		output.PainNotFelt = fnp
		output.Damage = damage
	}
	if target.Wounds > 0 {
		modelsKilled := float64(0)
		if damage < target.Wounds {
			modelsKilled = damage / target.Wounds
		} else {
			woundsRemaining := target.Wounds
			for damage > 0 {
				if damage < weapDamage {
					if damage > woundsRemaining {
						modelsKilled += 1
					} else {
						modelsKilled += damage / woundsRemaining
					}
					damage = 0
				} else {
					damage -= weapDamage
					woundsRemaining -= weapDamage
					if woundsRemaining <= 0 {
						modelsKilled += 1
						woundsRemaining = target.Wounds
					}
				}
			}
		}
		output.ModelsKilled = modelsKilled
	}
	return output
}

func calculateHitRoll(bsws float64) float64 {
	if bsws > 6 {
		bsws = 6
	} else if bsws < 2 {
		bsws = 2
	}
	return bsws
}

func calculateWoundRoll(strength float64, toughness float64) float64 {
	var woundTarget float64 = 4
	switch {
	case strength >= toughness*2:
		woundTarget = 2
	case strength > toughness:
		woundTarget = 3
	case toughness >= strength*2:
		woundTarget = 6
	case toughness > strength:
		woundTarget = 5
	}
	if woundTarget > 6 {
		woundTarget = 6
	} else if woundTarget < 2 {
		woundTarget = 2
	}
	return woundTarget
}

func calculateSave(target TargetProfile, ap float64, cover bool, modifier float64) float64 {
	modifiedSave := target.Save + ap + (modifier * -1)
	if cover {
		modifiedSave = applyCover(modifiedSave, ap)
	}
	if modifiedSave > target.InvulnerableSave && target.InvulnerableSave > 1 {
		return (target.InvulnerableSave)
	}
	if modifiedSave < 2 {
		modifiedSave = 2
	}
	return (modifiedSave)
}

func applyCover(modifiedSave float64, ap float64) float64 {
	if modifiedSave <= 3 && ap <= 0 {
		return modifiedSave
	}
	return modifiedSave - 1
}

func adjustmodifier(modifier float64) float64 {
	if modifier > 1 {
		return float64(1)
	} else if modifier < -1 {
		return float64(-1)
	} else {
		return modifier
	}
}

func calculateExpectedSuccesses(rolls float64, rollTarget float64, diceSides float64, modifier float64, criticalTarget float64) (float64, float64) {
	modifier *= -1
	if criticalTarget == 0 {
		return rolls * ((diceSides - (rollTarget + modifier - 1)) * (1 / diceSides)), float64(0)
	} else if criticalTarget <= rollTarget+modifier {
		return float64(0), rolls * ((diceSides - (criticalTarget - 1)) * (1 / diceSides))
	}
	return rolls * ((diceSides - (rollTarget + modifier - 1) - (diceSides - (criticalTarget - 1))) * (1 / diceSides)), rolls * ((diceSides - (criticalTarget - 1)) * (1 / diceSides))
}

type Inputs struct {
	Weapons WeaponProfile `json:"weapons"`
	Targets TargetProfile `json:"targets"`
}

type WeaponProfile struct {
	Name     string  `json:"name"`
	Attacks  string  `json:"attacks"`
	BSWS     float64 `json:"bsws"`
	Strength string  `json:"strength"`
	AP       float64 `json:"ap"`
	Damage   string  `json:"damage"`
}

type TargetProfile struct {
	Name             string  `json:"name"`
	Toughness        float64 `json:"toughness"`
	Save             float64 `json:"save"`
	InvulnerableSave float64 `json:"invulnerable_save"`
	Wounds           float64 `json:"wounds"`
	FNP              float64 `json:"fnp"`
}

type CalculationOptions struct {
	critHit    float64
	critWound  float64
	rerollHits bool
	// -1 Non Crits | 0 All Rolls | 1 1's
	hitsToReroll float64
	rerollWounds bool
	// -1 Non Crits | 0 All Rolls | 1 1's
	woundsToReroll float64
	sustainedHits  float64
	lethalHits     bool
	devWounds      bool
	hitModifier    float64
	woundModifier  float64
	saveModifier   float64
	cover          bool
}

func NewCalculationOptions(optsFn func(co *CalculationOptions)) *CalculationOptions {
	calcOptions := CalculationOptions{
		critHit:        6,
		critWound:      6,
		rerollHits:     false,
		rerollWounds:   false,
		hitsToReroll:   1,
		woundsToReroll: 1,
		sustainedHits:  0,
		lethalHits:     false,
		devWounds:      false,
		hitModifier:    0,
		woundModifier:  0,
		saveModifier:   0,
		cover:          false,
	}
	if optsFn != nil {
		optsFn(&calcOptions)
	}
	return &calcOptions
}

type Output struct {
	Name               string  `json:"name"`
	Hits               float64 `json:"hits"`
	Wounds             float64 `json:"wounds"`
	FailedSaves        float64 `json:"failed_saves"`
	Damage             float64 `json:"damage"`
	ModelsKilled       float64 `json:"models_killed"`
	RerolledHits       float64 `json:"rerolled_hits"`
	RerolledCritHits   float64 `json:"rerolled_crit_hits"`
	RerolledWounds     float64 `json:"rerolled_wounds"`
	RerolledWoundCrits float64 `json:"rerolled_wound_crits"`
	SustainedHits      float64 `json:"sustained_hits"`
	LethalHits         float64 `json:"lethal_hits"`
	DevWounds          float64 `json:"dev_wounds"`
	PainNotFelt        float64 `json:"pain_not_felt"`
}

func NewOutput(outputFn func(o *Output)) *Output {
	output := Output{
		Name:               "Unknown",
		Hits:               0,
		Wounds:             0,
		FailedSaves:        0,
		Damage:             0,
		ModelsKilled:       0,
		RerolledHits:       0,
		RerolledCritHits:   0,
		RerolledWounds:     0,
		RerolledWoundCrits: 0,
		SustainedHits:      0,
		LethalHits:         0,
		DevWounds:          0,
		PainNotFelt:        0,
	}
	if outputFn != nil {
		outputFn(&output)
	}
	return &output
}
