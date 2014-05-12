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

var devnull, _err = os.Create(os.DevNull)
var logger = log.New(devnull, "", log.LstdFlags)

type Faction uint8
var NEITHER, REBELS, EMPIRE Faction = 0, 1, 2

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
    faction Faction
    skill int
    attack int
    defense int
    hull int
    shields int
    focusTokens int
    evadeTokens int
    lockedOnto *Ship
    hasHowlrunnerReroll bool
    providesHowlrunnerReroll bool
    isDestroyed bool
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
    logger.Printf("=== %s is attacking %s ===\n", ship, target)
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
	    logger.Printf("No focus but %s (PS %d) has a reroll from Howlrunner\n", ship.name, ship.skill)
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
	    logger.Printf("%s (PS %d) rerolls blank from Howlrunner\n", ship.name, ship.skill)
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

    // all damage to shields first
    // apply regular hits first
    shieldDamage := int(math.Min(float64(attackResults.hits), float64(target.shields)))
    logger.Println("Took", shieldDamage, "hits on shields")
    attackResults.hits -= shieldDamage
    target.shields -= shieldDamage
    // then apply crits
    if attackResults.hits == 0 && target.shields > 0 {
	shieldDamage := int(math.Min(float64(attackResults.crits), float64(target.shields)))
	logger.Println("Took", shieldDamage, "crits on shields")
	attackResults.crits -= shieldDamage
	target.shields -= shieldDamage
    }

    // apply damage to hull
    if attackResults.hits > 0 {
	logger.Println("Took", attackResults.hits, "hull damage")
	target.hull -= attackResults.hits
    }

    // apply crits to hull, 7/33 chance of direct hit
    for crit := 0; crit < attackResults.crits; crit++ {
	if uint8(rand.Int31n(33)) < 7 {
	    logger.Println("Direct Hit!")
	    target.hull -= 2
	} else {
	    logger.Println("Suffered other crit")
	    target.hull--
	}
    }

    if target.hull < 1 {
	logger.Println(target.name, "was destroyed!")
	target.isDestroyed = true
    }
}

type Squadron [](*Ship)
func (s Squadron) Len() int { return len(s) }
func (s Squadron) Less(i, j int) bool { return s[i].skill < s[j].skill }
func (s Squadron) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func NewSquadron(l[](*Ship)) *Squadron {
    s := Squadron(l)
    sort.Sort(sort.Reverse(s))
    return &s
}

type Match struct {
    rebelList *Squadron
    empireList *Squadron
}

type MatchResult struct {
    winner Faction
    shipsRemaining int
}

func (m MatchResult) String() (s string) {
    switch f := m.winner; f {
    case NEITHER:
	s = "draw"
    case REBELS:
	s = fmt.Sprintf("Rebels with %d ships remaining", m.shipsRemaining)
    case EMPIRE:
	s = fmt.Sprintf("Empire with %d ships remaining", m.shipsRemaining)
    }
    return
}

func (match *Match) PerformCombatRound(performAction func(*Ship)) *MatchResult {
    // returns match result if one side wins at the end, nil otherwise
    logger.Println("=== New Combat Round ===")

    // In descending pilot skill...
    for ps := 12; ps > -1; ps-- {
	// For later use: check if Howlrunner is alive
	howlrunnerRerollAvailable := false
	for _, ship := range(*match.empireList) {
	    if ship.providesHowlrunnerReroll && !ship.isDestroyed {
		howlrunnerRerollAvailable = true
	    }
	}

	// Gather eligible ships (not destroyed)
	combatants := make([](*Ship), 0, len(*match.rebelList) + len(*match.empireList))
	for _, ship := range(*match.rebelList) {
	    if ship.skill == ps && !ship.isDestroyed {
		combatants = append(combatants, ship)
	    }
	}

	for _, ship := range(*match.empireList) {
	    if ship.skill == ps && !ship.isDestroyed {
		if !ship.providesHowlrunnerReroll {
		    ship.hasHowlrunnerReroll = howlrunnerRerollAvailable
		}
		combatants = append(combatants, ship)
	    }
	}

	// for simplicity's sake, combatants shoot at highest surviving PS in opposing list
	// since we aren't resolving crit effects, initiative doesn't matter
	for _, ship := range(combatants) {
	    var enemy_list *Squadron
	    var target *Ship

	    switch ship.faction {
	    case REBELS:
		enemy_list = match.empireList
	    case EMPIRE:
		enemy_list = match.rebelList
	    }
	    for _, enemy_ship := range(*enemy_list) {
		if !enemy_ship.isDestroyed {
		    target = enemy_ship
		    break
		}
	    }

	    performAction(ship)

	    // We don't break immediately if no targets are available because a draw could occur
	    if target != nil {
		ship.Attack(target)
	    }
	}
    }

    nRebelsRemaining, nImperialsRemaining := 0, 0
    for _, ship := range(*match.rebelList) {
	if !ship.isDestroyed {
	    nRebelsRemaining++
	}
    }

    for _, ship := range(*match.empireList) {
	if !ship.isDestroyed {
	    nImperialsRemaining++
	}
    }

    switch {
    case nRebelsRemaining > 0 && nImperialsRemaining > 0:
	return nil
    case nRebelsRemaining > 0:
	result := new(MatchResult)
	result.winner = REBELS
	result.shipsRemaining = nRebelsRemaining
	return result
    case nImperialsRemaining > 0:
	result := new(MatchResult)
	result.winner = EMPIRE
	result.shipsRemaining = nImperialsRemaining
	return result
    default:
	result := new(MatchResult)
	result.winner = NEITHER
	return result
    }
}

func (match *Match) Play(c chan MatchResult, performAction func(*Ship)) {
    var result *MatchResult
    for result == nil {
	result = match.PerformCombatRound(performAction)
    }
    c <- *result
}

type AggregateResults struct {
    rebelWins int
    empireWins int
    draws int
}
func (a AggregateResults) String() string {
    return fmt.Sprintf("Rebel wins: %d\nEmpire wins: %d\nDraws: %d", a.rebelWins, a.empireWins, a.draws)
}

func main() {
    rand.Seed(time.Now().UnixNano())

    focusAction := func (ship *Ship) {
	ship.Focus()
    }

    ch := make(chan MatchResult)

    nIterations := 1000
    results := new(AggregateResults)

    for i := 0; i < nIterations; i++ {
	luke := &Ship{faction: REBELS, name: "Luke Skywalker", skill: 8, attack: 3, defense: 2, hull: 3, shields: 2}
	porkins := &Ship{faction: REBELS, name: "Jek Porkins", skill: 7, attack: 3, defense: 2, hull: 3, shields: 2}
	rookie1 := &Ship{faction: REBELS, name: "Rookie Pilot", skill: 8, attack: 3, defense: 2, hull: 3, shields: 2}
	rookie2 := &Ship{faction: REBELS, name: "Rookie Pilot", skill: 7, attack: 3, defense: 2, hull: 3, shields: 2}

	// must be spelling this wrong
	rebels := NewSquadron([](*Ship){
	    luke,
	    porkins,
	    rookie1,
	    rookie2,
	})

	howlrunner := &Ship{faction: EMPIRE, name: "Howlrunner", skill: 8, attack: 2, defense: 3, hull: 3, providesHowlrunnerReroll: true}
	academy1 := &Ship{faction: EMPIRE, name: "Mauler Mithel", skill: 7, attack: 2, defense: 3, hull: 3}
	academy2 := &Ship{faction: EMPIRE, name: "Alpha Squadron Pilot", skill: 8, attack: 3, defense: 3, hull: 3}
	academy3 := &Ship{faction: EMPIRE, name: "Alpha Squadron Pilot", skill: 7, attack: 3, defense: 3, hull: 3}
	academy4 := &Ship{faction: EMPIRE, name: "Academy Pilot", skill: 1, attack: 2, defense: 3, hull: 3}
	academy5 := &Ship{faction: EMPIRE, name: "Academy Pilot", skill: 1, attack: 2, defense: 3, hull: 3}

	imps := NewSquadron([](*Ship){
	    howlrunner,
	    academy1,
	    academy2,
	    academy3,
	    academy4,
	    academy5,
	})

	match := &Match{rebels, imps}
	go match.Play(ch, focusAction)
    }

    resultsReceived := 0
    for resultsReceived < nIterations {
	var result MatchResult
	select {
	case result = <-ch:
	    resultsReceived++
	    switch result.winner {
	    case REBELS:
		results.rebelWins++
	    case EMPIRE:
		results.empireWins++
	    default:
		results.draws++
	    }
	}
    }

    fmt.Println(*results)
}
