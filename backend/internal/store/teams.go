package store

import "database/sql"

type Team struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	GroupID           string `json:"group_id"`
	FlagURL           string `json:"flag_url"`
	FIFARanking       int    `json:"fifa_ranking"`
	ManagerName       string `json:"manager_name"`
	PreTournamentForm string `json:"pre_tournament_form"` // JSON-encoded array of last 5 matches
	FixtureSrcID      string `json:"fixture_src_id"`
}

// UpdateTeamGroup sets group_id for an existing team. Used by bootstrap to
// backfill the group from match data (football-data.org's /teams endpoint
// doesn't include group info — it lives on each /matches record as `group`).
// No-op if the team doesn't exist.
func (s *Store) UpdateTeamGroup(code, groupID string) error {
	_, err := s.db.Exec(`UPDATE teams SET group_id = ? WHERE code = ?`, groupID, code)
	return err
}

func (s *Store) UpsertTeam(t Team) error {
	_, err := s.db.Exec(
		`INSERT INTO teams (code, name, group_id, flag_url, fifa_ranking, manager_name, pre_tournament_form, fixture_src_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(code) DO UPDATE SET
		   name=excluded.name, group_id=excluded.group_id, flag_url=excluded.flag_url,
		   fifa_ranking=excluded.fifa_ranking, manager_name=excluded.manager_name,
		   pre_tournament_form=excluded.pre_tournament_form, fixture_src_id=excluded.fixture_src_id`,
		t.Code, t.Name, t.GroupID, t.FlagURL, t.FIFARanking, t.ManagerName, t.PreTournamentForm, t.FixtureSrcID,
	)
	return err
}

func (s *Store) GetTeam(code string) (Team, error) {
	var t Team
	var group, flag, manager, form, fixID sql.NullString
	var ranking sql.NullInt64
	row := s.db.QueryRow(`SELECT code, name, group_id, flag_url, fifa_ranking, manager_name, pre_tournament_form, fixture_src_id FROM teams WHERE code = ?`, code)
	if err := row.Scan(&t.Code, &t.Name, &group, &flag, &ranking, &manager, &form, &fixID); err != nil {
		return Team{}, err
	}
	t.GroupID, t.FlagURL, t.ManagerName, t.PreTournamentForm, t.FixtureSrcID =
		group.String, flag.String, manager.String, form.String, fixID.String
	t.FIFARanking = int(ranking.Int64)
	return t, nil
}

// GetTeamByFixtureSrcID looks up a team by the upstream fixture-source ID
// (football-data.org's numeric team ID, stored as a string in fixture_src_id).
// This is used as a fallback when the TLA returned by the /matches endpoint
// doesn't match any team.code — football-data.org's /matches endpoint
// occasionally returns a different TLA than /teams for the same team
// (e.g. Curaçao: CUW from /teams, CUR from /matches).
func (s *Store) GetTeamByFixtureSrcID(srcID string) (Team, error) {
	var t Team
	var group, flag, manager, form, fixID sql.NullString
	var ranking sql.NullInt64
	row := s.db.QueryRow(`SELECT code, name, group_id, flag_url, fifa_ranking, manager_name, pre_tournament_form, fixture_src_id FROM teams WHERE fixture_src_id = ?`, srcID)
	if err := row.Scan(&t.Code, &t.Name, &group, &flag, &ranking, &manager, &form, &fixID); err != nil {
		return Team{}, err
	}
	t.GroupID, t.FlagURL, t.ManagerName, t.PreTournamentForm, t.FixtureSrcID =
		group.String, flag.String, manager.String, form.String, fixID.String
	t.FIFARanking = int(ranking.Int64)
	return t, nil
}

func (s *Store) ListTeams() ([]Team, error) {
	rows, err := s.db.Query(`SELECT code, name, group_id, flag_url, fifa_ranking, manager_name, pre_tournament_form, fixture_src_id FROM teams ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Team
	for rows.Next() {
		var t Team
		var group, flag, manager, form, fixID sql.NullString
		var ranking sql.NullInt64
		if err := rows.Scan(&t.Code, &t.Name, &group, &flag, &ranking, &manager, &form, &fixID); err != nil {
			return nil, err
		}
		t.GroupID, t.FlagURL, t.ManagerName, t.PreTournamentForm, t.FixtureSrcID =
			group.String, flag.String, manager.String, form.String, fixID.String
		t.FIFARanking = int(ranking.Int64)
		out = append(out, t)
	}
	return out, rows.Err()
}
