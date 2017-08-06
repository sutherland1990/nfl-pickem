package main

import (
	"log"
	"math/rand"
	"time"

	nflpickem "github.com/ameske/nfl-pickem"
	"github.com/ameske/nfl-pickem/sqlite3"
	"github.com/spf13/cobra"
)

var testWeeks uint
var testWeek uint
var testYear uint
var testThur, testSunEarly, testSunLate, testSunNight, testMon bool

func init() {
	TestCmd.AddCommand(setupCommand)
	TestCmd.AddCommand(generateCommand)

	setupCommand.AddCommand(generateResultsCommand)

	// Game/User/Pick setup (year/weeks)
	setupCommand.Flags().UintVarP(&testWeeks, "weeks", "w", 0, "number of weeks to generate fake data")

	// Randomize pick selections (year/week)

	// Randomize game results (year/week)
	generateCommand.AddCommand(generateResultsCommand)
	generateResultsCommand.Flags().UintVarP(&testWeek, "week", "w", 0, "week to generate game results for")
	generateResultsCommand.Flags().UintVarP(&testYear, "year", "y", 0, "year to genearate game results for")
	generateResultsCommand.Flags().BoolVarP(&testThur, "thur", "t", false, "generate thursday game result")
	generateResultsCommand.Flags().BoolVarP(&testSunEarly, "sune", "e", false, "generate sunday early game result")
	generateResultsCommand.Flags().BoolVarP(&testSunLate, "sunl", "l", false, "generate sunday late game result")
	generateResultsCommand.Flags().BoolVarP(&testSunNight, "sunn", "n", false, "generate sunday night game result")
	generateResultsCommand.Flags().BoolVarP(&testMon, "mon", "m", false, "generate monday game result")
}

var TestCmd = &cobra.Command{
	Use:   "testing",
	Short: "manipulate a test db instance",
	Long:  "manipulate a test db instance",
}

var generateCommand = &cobra.Command{
	Use:   "generate",
	Short: "generate fake data for a db instance",
	Long:  "generate fake data for a db instance",
}

var generateResultsCommand = &cobra.Command{
	Use:   "results",
	Short: "generate fake results for a db instance",
	Long:  "generate fake results for a db instance",
	Run: func(cmd *cobra.Command, args []string) {
		if datastore == "" {
			log.Fatal("db flag is required")
		}

		if testYear == 0 || testWeek == 0 {
			log.Fatal("year and week required")
		}

		rand.Seed(time.Now().Unix())

		db, err := sqlite3.NewDatastore(datastore)
		if err != nil {
			log.Fatal(err)
		}

		// Get the week's games
		games, err := db.WeekGames(int(testYear), int(testWeek))
		if err != nil {
			log.Fatal(err)
		}

		log.Println(games)

		// Generate random scores and upate the game
		for _, g := range games {
			home := rand.Intn(64)
			away := rand.Intn(64)

			if verbose {
				log.Printf("UpdateGame(%v, %v, %v, %v, %v)\n", int(testWeek), int(testYear), g.Home.Nickname, home, away)
			}

			err := db.UpdateGame(int(testWeek), int(testYear), g.Home.Nickname, home, away)
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

var setupCommand = &cobra.Command{
	Use:   "setup",
	Short: "setup a test db instance with generated data",
	Long:  "setup a test db instance with generated data",
	Run: func(cmd *cobra.Command, args []string) {
		if testWeeks == 0 {
			log.Fatal("weeks must be set via command line")

		}

		if datastore == "" {
			log.Fatal("db flag is required")
		}

		db, err := sqlite3.NewDatastore(datastore)
		if err != nil {
			log.Fatal(err)
		}

		users, err := addTestUsers(db)
		if err != nil {
			log.Fatal(err)
		}

		next := nextNFLWeek(time.Now())

		err = db.AddYear(next.Year(), int(next.Unix()))
		if err != nil {
			log.Fatal(err)
		}

		for i := 0; i < int(testWeeks); i++ {
			err = db.AddWeek(next.Year(), i+1)
			if err != nil {
				log.Fatal(err)
			}

			err = addFakeGames(db, next)
			if err != nil {
				log.Fatal(err)
			}

			for _, u := range users {
				err = db.CreatePicks(u, next.Year(), i+1)
				if err != nil {
					log.Fatal(err)
				}
			}

			next = next.Add(time.Hour * 24 * 7)
		}
	},
}

var (
	teams = map[int]string{
		1:  "Ravens",
		2:  "Bengals",
		3:  "Browns",
		4:  "Steelers",
		5:  "Bears",
		6:  "Lions",
		7:  "Packers",
		8:  "Vikings",
		9:  "Texans",
		10: "Colts",
		11: "Jaguars",
		12: "Titans",
		13: "Falcons",
		14: "Panthers",
		15: "Saints",
		16: "Buccaneers",
		17: "Bills",
		18: "Dolphins",
		19: "Patriots",
		20: "Jets",
		21: "Cowboys",
		22: "Giants",
		23: "Eagles",
		24: "Redskins",
		25: "Broncos",
		26: "Chiefs",
		27: "Raiders",
		28: "Chargers",
		29: "Cardinals",
		30: "Rams",
		31: "49ers",
		32: "Seahawks",
	}
)

// nextNFLWeek calculates the time representing the start of the next possible week that can be used to immediately test.
//
// If the day is Sunday or Monday, the next week is the next Tuesday. If the day is Tuesday or Wednesday, we can use the current
// week to test. Otherwise, the week is the next Tuesday.
func nextNFLWeek(t time.Time) time.Time {
	var next time.Time
	switch t.Weekday() {
	case time.Sunday, time.Monday:
		next = nextDay(t, time.Tuesday)
	case time.Tuesday, time.Wednesday:
		next = nextDay(time.Date(t.Year(), t.Month(), t.Day()-7, t.Hour(), t.Minute(), t.Second(), 0, t.Location()), time.Tuesday)
	default:
		next = nextDay(t, time.Tuesday)
	}

	return next
}

// addFakeGames adds a fake schedule for the week represented by the start time
func addFakeGames(db nflpickem.Service, start time.Time) error {
	curTeam := 1

	// One game on Thursday
	thur := nextDay(start, time.Thursday)
	thur = time.Date(thur.Year(), thur.Month(), thur.Day(), 20, 30, 0, 0, thur.Location())
	err := db.AddGame(thur, teams[curTeam], teams[curTeam+1])
	if err != nil {
		return err
	}

	curTeam += 2

	// Nine games at 1:00 Sunday
	sunday := nextDay(start, time.Sunday)
	sunday = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 13, 0, 0, 0, sunday.Location())
	for i := 0; i < 9; i++ {
		err = db.AddGame(sunday, teams[curTeam], teams[curTeam+1])
		if err != nil {
			return err
		}

		curTeam += 2
	}

	// Three games at 4:00 Sunday
	sunday = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 16, 0, 0, 0, sunday.Location())
	for i := 0; i < 3; i++ {
		err = db.AddGame(sunday, teams[curTeam], teams[curTeam+1])
		if err != nil {
			return err
		}

		curTeam += 2
	}

	// One game at 4:25 Sunday
	sunday = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 16, 25, 0, 0, sunday.Location())
	err = db.AddGame(sunday, teams[curTeam], teams[curTeam+1])
	if err != nil {
		return err
	}

	curTeam += 2

	// One game on Sunday Night
	sunday = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 20, 30, 0, 0, sunday.Location())
	err = db.AddGame(sunday, teams[curTeam], teams[curTeam+1])
	if err != nil {
		return err
	}

	curTeam += 2

	// One game on Monday Night
	monday := nextDay(start, time.Monday)
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 20, 30, 0, 0, monday.Location())
	err = db.AddGame(monday, teams[curTeam], teams[curTeam+1])
	if err != nil {
		return err
	}

	return nil
}

// nextDay advances to the next instance of the given time.Weekday
func nextDay(now time.Time, day time.Weekday) time.Time {
	// We only want to go forwards, so use modular arith to force going ahead
	diff := int(day-now.Weekday()+7) % 7

	next := now.AddDate(0, 0, diff)
	next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())

	return next
}

// addTestUsers adds Alice and Bob to the given nflpickem.Service
func addTestUsers(db nflpickem.Service) ([]string, error) {
	err := db.AddUser("Alice", "Tester", "alice@gmail.com", "password", true)
	if err != nil {
		return nil, err
	}

	err = db.AddUser("Bob", "Tester", "bob@gmail.com", "password", true)
	if err != nil {
		return nil, err
	}

	return []string{"alice@gmail.com", "bob@gmail.com"}, nil
}