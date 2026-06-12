package odds

// aliases maps football-data.org canonical team names to the names used by
// the-odds-api. Only the five teams where the two providers disagree need
// entries; lookup falls through to identity for the other 43 WC teams.
//
// Regenerate by diffing the DB's teams.name column against the home_team /
// away_team strings in a live /v4/sports/soccer_fifa_world_cup/odds/ response.
var aliases = map[string]string{
	"Bosnia-Herzegovina": "Bosnia & Herzegovina",
	"Cape Verde Islands": "Cape Verde",
	"Congo DR":           "DR Congo",
	"Czechia":            "Czech Republic",
	"United States":      "USA",
}

func oddsAPIName(dbName string) string {
	if v, ok := aliases[dbName]; ok {
		return v
	}
	return dbName
}
