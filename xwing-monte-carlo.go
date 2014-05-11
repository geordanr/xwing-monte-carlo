// Simplified Monte Carlo simulation of an X-Wing battle.
package main

import (
    "fmt"
    "log"
    "math"
    "math/rand"
    "os"
    "sort"
    "time"
)

var logger = log.New(os.Stderr, "", log.LstdFlags)

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

func (results *DiceResults) RerollOneBlank(rollfunc func() DieResult) *DiceResults {
    if results.blanks == 0 {
	return results
    }
    results.blanks--
    new_results := Roll(1, rollfunc)
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

func (results *DiceResults) RerollOneBlankOrFocus(rollfunc func() DieResult) *DiceResults {
    if results.blanks == 0  && results.focuses == 0 {
	return results
    }
    if results.blanks > 0 {
	return results.RerollOneBlank(rollfunc)
    }
    results.focuses--
    new_results := Roll(1, rollfunc)
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
    name string
    skill int
    attack int
    defense int
    hull int
    shields int
    focusTokens int
    evadeTokens int
    lockedOnto *Ship
    hasHowlrunnerReroll bool
}

func (ship Ship) String() string {
    return fmt.Sprintf("<Ship %s skill=%d attack=%d defense=%d hull=%d shields=%d>", ship.name, ship.skill, ship.attack, ship.defense, ship.hull, ship.shields)
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

func (ship *Ship) AcquireTargetLock(target *Ship) *Ship {
    ship.lockedOnto = target
    return ship
}

func (ship *Ship) SpendTargetLock() *Ship {
    ship.lockedOnto = nil
    return ship
}

func (ship *Ship) Attack(target *Ship) {
    logger.Println("=== Attack! ===")
    attackResults := Roll(ship.attack, AttackDie)
    logger.Println(attackResults)
    if ship.focusTokens == 0 {
	// No focus tokens, so it's okay to reroll eyeballs
	if attackResults.blanks > 0 && ship.lockedOnto == target {
	    logger.Println("No focus but we have a target lock, reroll all misses")
	    attackResults.RerollBlanksAndFocuses(AttackDie)
	    ship.SpendTargetLock()
	    logger.Println(attackResults)
	} else if ship.hasHowlrunnerReroll && (attackResults.blanks > 0 || attackResults.focuses > 0) {
	    logger.Println("No focus but we have a reroll from Howlrunner")
	    attackResults.RerollOneBlankOrFocus(AttackDie)
	    logger.Println(attackResults)
	}
    } else if attackResults.blanks > 0 {
	if ship.lockedOnto == target {
	    logger.Println("We have a target lock and focus, reroll only blanks")
	    attackResults.RerollBlanks(AttackDie)
	    ship.SpendTargetLock()
	    logger.Println(attackResults)
	} else if ship.hasHowlrunnerReroll {
	    logger.Println("Reroll blank from Howlrunner")
	    attackResults.RerollOneBlank(AttackDie)
	    logger.Println(attackResults)
	}
    }
    if attackResults.focuses > 0 && ship.focusTokens > 0 {
	logger.Println("Burning focus")
	attackResults.SpendFocus("attack")
	ship.focusTokens--
    }
    logger.Println("Final attack results:", attackResults)

    totalHits := attackResults.hits + attackResults.crits

    logger.Println("--- Defense! ---")
    defenseResults := Roll(target.defense, DefenseDie)
    logger.Println(defenseResults)
    if defenseResults.evades >= totalHits {
	logger.Println("Naturally evaded all hits")
	return
    }

    if defenseResults.focuses > 0 && target.focusTokens > 0 {
	logger.Println("Burning focus")
	defenseResults.SpendFocus("defense")
	target.focusTokens--
    }
    if defenseResults.evades >= totalHits {
	logger.Println("Evaded all hits after using focus")
	return
    }

    for ; target.evadeTokens > 0; target.evadeTokens-- {
	logger.Println("Spending evade token...")
	defenseResults.evades++
	if defenseResults.evades >= totalHits {
	    logger.Println("Evaded all hits after burning evade")
	    return
	}
    }

    // cancel hits before crits
    hitsCanceled := int(math.Min(float64(attackResults.hits), float64(defenseResults.evades)))
    logger.Println("Canceled", hitsCanceled, "hits")
    attackResults.hits -= hitsCanceled
    defenseResults.evades -= hitsCanceled

    critsCanceled := int(math.Min(float64(attackResults.crits), float64(defenseResults.evades)))
    logger.Println("Canceled", critsCanceled, "crits")
    attackResults.crits -= critsCanceled
    defenseResults.evades -= critsCanceled

    logger.Println("Damage sustained:", attackResults)
}

type Squadron [](*Ship)

func (s Squadron) Len() int { return len(s) }
func (s Squadron) Less(i, j int) bool { return s[i].skill < s[j].skill }
func (s Squadron) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func main() {
    rand.Seed(time.Now().Unix())
    xwing := &Ship{"X-Wing", 8, 3, 2, 3, 2, 0, 0, nil, false}
    logger.Println(xwing)
    tiefighter := &Ship{"TIE Fighter", 8, 2, 3, 3, 0, 0, 0, nil, true}
    logger.Println(tiefighter)

    logger.Println("=== Unmodified both")
    xwing.Attack(tiefighter)
    xwing.CleanUp()
    tiefighter.CleanUp()

    logger.Println("=== TL+F attack vs. Focus defense")
    xwing.Focus()
    xwing.AcquireTargetLock(tiefighter)
    tiefighter.Focus()
    xwing.Attack(tiefighter)
    xwing.CleanUp()
    tiefighter.CleanUp()

    logger.Println("=== Unmodified attack vs. Evade")
    tiefighter.Evade()
    xwing.Attack(tiefighter)
    xwing.CleanUp()
    tiefighter.CleanUp()

    logger.Println("=== Focus vs. Evade")
    xwing.Focus()
    tiefighter.Evade()
    xwing.Attack(tiefighter)
    xwing.CleanUp()
    tiefighter.CleanUp()


    logger.Println("=== TIE Attack ===")
    logger.Println("=== Howlrunner only")
    tiefighter.Attack(xwing)
    xwing.CleanUp()
    tiefighter.CleanUp()

    logger.Println("=== Howlrunner with Focus")
    tiefighter.Focus()
    tiefighter.Attack(xwing)
    xwing.CleanUp()
    tiefighter.CleanUp()

    luke := &Ship{"X-Wing", 8, 3, 2, 3, 2, 0, 0, nil, false}
    porkins := &Ship{"X-Wing", 7, 3, 2, 3, 2, 0, 0, nil, false}
    rookie1 := &Ship{"X-Wing", 2, 3, 2, 3, 2, 0, 0, nil, false}
    rookie2 := &Ship{"X-Wing", 2, 3, 2, 3, 2, 0, 0, nil, false}

    // must be spelling this wrong
    s := Squadron([](*Ship){rookie1, porkins, rookie2, luke})
    log.Println(s)
    sort.Sort(sort.Reverse(s))
    log.Println(s)

}
