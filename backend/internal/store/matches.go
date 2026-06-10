package store

import "database/sql"

type Match struct {
	ID            string
	HomeTeamCode  string
	AwayTeamCode  string
	KickoffUTC    string
	Stage         string
	Venue         string
	FixtureSrcID  string
	HomeScore     *int
	AwayScore     *int
	ResultFetched string
}

func (s *Store) UpsertMatch(m Match) error {
	_, err := s.db.Exec(
		`INSERT INTO matches (id, home_team_code, away_team_code, kickoff_utc, stage, venue, fixture_src_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   home_team_code=excluded.home_team_code, away_team_code=excluded.away_team_code,
		   kickoff_utc=excluded.kickoff_utc, stage=excluded.stage,
		   venue=excluded.venue, fixture_src_id=excluded.fixture_src_id`,
		m.ID, m.HomeTeamCode, m.AwayTeamCode, m.KickoffUTC, m.Stage, m.Venue, m.FixtureSrcID,
	)
	return err
}

func (s *Store) GetMatch(id string) (Match, error) {
	return s.scanMatch(s.db.QueryRow(`SELECT id, home_team_code, away_team_code, kickoff_utc, stage, venue, fixture_src_id, home_score, away_score, result_fetched_at FROM matches WHERE id = ?`, id))
}

func (s *Store) ListMatches() ([]Match, error) {
	rows, err := s.db.Query(`SELECT id, home_team_code, away_team_code, kickoff_utc, stage, venue, fixture_src_id, home_score, away_score, result_fetched_at FROM matches ORDER BY kickoff_utc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Match
	for rows.Next() {
		m, err := s.scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) SetMatchResult(id string, home, away int, fetchedAt string) error {
	_, err := s.db.Exec(
		`UPDATE matches SET home_score = ?, away_score = ?, result_fetched_at = ? WHERE id = ?`,
		home, away, fetchedAt, id,
	)
	return err
}

type rowScanner interface{ Scan(dest ...any) error }

func (s *Store) scanMatch(r rowScanner) (Match, error) {
	var m Match
	var venue, fixID, fetched sql.NullString
	var home, away sql.NullInt64
	if err := r.Scan(&m.ID, &m.HomeTeamCode, &m.AwayTeamCode, &m.KickoffUTC, &m.Stage, &venue, &fixID, &home, &away, &fetched); err != nil {
		return Match{}, err
	}
	m.Venue, m.FixtureSrcID, m.ResultFetched = venue.String, fixID.String, fetched.String
	if home.Valid {
		v := int(home.Int64)
		m.HomeScore = &v
	}
	if away.Valid {
		v := int(away.Int64)
		m.AwayScore = &v
	}
	return m, nil
}
