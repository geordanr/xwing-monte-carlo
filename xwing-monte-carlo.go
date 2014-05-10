// Simplified Monte Carlo simulation of an X-Wing battle.
package main

import (
    "math"
    "math/rand"
    "fmt"
    "time"
)

type DieResult uint8
var BLANK, FOCUS, HIT, CRIT, EVADE DieResult = 0, 1, 2, 3, 4

type DiceResults struct {
    hits int
    crits int
    evades int
    focuses int
    blanks int
}

func (results DiceResults) String() string {
    return fmt.Sprintf("<DiceResult %d hits, %d crits, %d evades, %d focuses, %d blanks>", results.hits, results.crits, results.evades, results.focuses, results.blanks)
}

func Roll(numDice int, rollfunc func() DieResult) (results DiceResults) {
    for i := 0; i < numDice; i++ {
	switch result := rollfunc(); result {
	case BLANK:
	    results.blanks++
	case FOCUS:
	    results.focuses++
	case HIT:
	    results.hits++
	case CRIT:
	    results.crits++
	case EVADE:
	    results.evades++
	}
    }
    return
}

func (results *DiceResults) Add(other DiceResults) *DiceResults {
    results.blanks += other.blanks
    results.focuses += other.focuses
    results.hits += other.hits
    results.crits += other.crits
    results.evades += other.evades
    return results
}

func (results *DiceResults) RerollBlanks(rollfunc func() DieResult) *DiceResults {
    if results.blanks == 0 {
	return results
    }
    numToReroll := results.blanks
    results.blanks = 0
    new_results := Roll(numToReroll, rollfunc)
    results.Add(new_results)
    return results
}

func (results *DiceResults) RerollBlanksAndFocuses(rollfunc func() DieResult) *DiceResults {
    if results.blanks == 0  && results.focuses == 0 {
	return results
    }
    numToReroll := results.blanks + results.focuses
    results.blanks = 0
    results.focuses = 0
    new_results := Roll(numToReroll, rollfunc)
    results.Add(new_results)
    return results
}

func (results *DiceResults) SpendFocus(onWhat string) *DiceResults {
    switch onWhat {
    case "attack":
	results.hits += results.focuses
    case "defense":
	results.evades += results.focuses
    }
    results.focuses = 0
    return results
}

func (results *DiceResults) SpendEvade() *DiceResults {
    results.evades++
    return results
}

func AttackDie() DieResult {
    face := DieResult(uint8(rand.Int31n(8)))
    switch {
    case face < 2:
	return BLANK
    case face < 4:
	return FOCUS
    case face < 7:
	return HIT
    default:
	return CRIT
    }
}

func DefenseDie() DieResult {
    face := DieResult(uint8(rand.Int31n(8)))
    switch {
    case face < 3:
	return BLANK
    case face < 5:
	return FOCUS
    default:
	return EVADE
    }
}


type Ship struct {
    skill int
    attack int
    defense int
    hull int
    shields int
    focusTokens int
    evadeTokens int
    hasTargetLock bool
}

func (ship *Ship) CleanUp() *Ship {
    ship.focusTokens = 0
    ship.evadeTokens = 0
    return ship
}

func (ship *Ship) Focus() *Ship {
    ship.focusTokens++
    return ship
}

func (ship *Ship) Evade() *Ship {
    ship.evadeTokens++
    return ship
}

func (ship *Ship) AcquireTargetLock() *Ship {
    // yeah it should be on someone but eh
    ship.hasTargetLock = true
    return ship
}

func (ship *Ship) SpendTargetLock() *Ship {
    ship.hasTargetLock = false
    return ship
}

func (ship *Ship) Attack(target *Ship) {
    fmt.Println("=== Attack! ===")
    attackResults := Roll(ship.attack, AttackDie)
    fmt.Println(attackResults)
    if ship.focusTokens == 0 && attackResults.blanks > 0 && ship.hasTargetLock {
	fmt.Println("No focus but we have a target lock, reroll all misses")
	attackResults.RerollBlanksAndFocuses(AttackDie)
	ship.SpendTargetLock()
	fmt.Println(attackResults)
    } else if attackResults.blanks > 0 && ship.hasTargetLock {
	fmt.Println("We have a target lock and focus, reroll only blanks")
	attackResults.RerollBlanks(AttackDie)
	ship.SpendTargetLock()
	fmt.Println(attackResults)
    }
    if attackResults.focuses > 0 && ship.focusTokens > 0 {
	fmt.Println("Burning focus")
	attackResults.SpendFocus("attack")
	ship.focusTokens--
    }
    fmt.Println("Final attack results:", attackResults)

    totalHits := attackResults.hits + attackResults.crits

    fmt.Println("--- Defense! ---")
    defenseResults := Roll(target.defense, DefenseDie)
    fmt.Println(defenseResults)
    if defenseResults.evades >= totalHits {
	fmt.Println("Naturally evaded all hits")
	return
    }

    if defenseResults.focuses > 0 && target.focusTokens > 0 {
	fmt.Println("Burning focus")
	defenseResults.SpendFocus("defense")
	target.focusTokens--
    }
    if defenseResults.evades >= totalHits {
	fmt.Println("Evaded all hits after using focus")
	return
    }

    for ; target.evadeTokens > 0; target.evadeTokens-- {
	fmt.Println("Spending evade token...")
	defenseResults.evades++
	if defenseResults.evades >= totalHits {
	    fmt.Println("Evaded all hits after burning evade")
	    return
	}
    }

    // cancel hits before crits
    hitsCanceled := int(math.Min(float64(attackResults.hits), float64(defenseResults.evades)))
    fmt.Println("Canceled", hitsCanceled, "hits")
    attackResults.hits -= hitsCanceled
    defenseResults.evades -= hitsCanceled

    critsCanceled := int(math.Min(float64(attackResults.crits), float64(defenseResults.evades)))
    fmt.Println("Canceled", critsCanceled, "crits")
    attackResults.crits -= critsCanceled
    defenseResults.evades -= critsCanceled

    fmt.Println("Damage sustained:", attackResults)
}


func main() {
    rand.Seed(time.Now().Unix())

    xwing := Ship{8, 3, 2, 3, 2, 0, 0, false}
    tiefighter := Ship{8, 2, 3, 3, 0, 0, 0, false}

    fmt.Println("=== Unmodified both")
    xwing.Attack(&tiefighter)
    xwing.CleanUp()
    tiefighter.CleanUp()

    fmt.Println("=== TL+F attack vs. Focus defense")
    xwing.Focus()
    xwing.AcquireTargetLock()
    tiefighter.Focus()
    xwing.Attack(&tiefighter)
    xwing.CleanUp()
    tiefighter.CleanUp()

    fmt.Println("=== Unmodified attack vs. Evade")
    tiefighter.Evade()
    xwing.Attack(&tiefighter)
    xwing.CleanUp()
    tiefighter.CleanUp()

    fmt.Println("=== Focus vs. Evade")
    xwing.Focus()
    tiefighter.Evade()
    xwing.Attack(&tiefighter)
    xwing.CleanUp()
    tiefighter.CleanUp()
}
