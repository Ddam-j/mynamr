package registry

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/Ddam-j/mynamr/internal/rules"
)

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	store := &Store{db: db}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) ListRules() ([]rules.RuleSpec, error) {
	rows, err := s.db.Query(`
		SELECT name, description, enabled, source, spec, recognizer, matcher, selector, renderer
		FROM rules
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]rules.RuleSpec, 0)
	for rows.Next() {
		var rule rules.RuleSpec
		var enabled bool
		if err := rows.Scan(
			&rule.Name,
			&rule.Description,
			&enabled,
			&rule.Source,
			&rule.Spec,
			&rule.Recognizer,
			&rule.Matcher,
			&rule.Selector,
			&rule.Renderer,
		); err != nil {
			return nil, err
		}
		rule.Enabled = enabled
		result = append(result, rule)
	}

	return result, rows.Err()
}

func (s *Store) Rule(name string) (rules.RuleSpec, error) {
	var rule rules.RuleSpec
	var enabled bool

	err := s.db.QueryRow(`
		SELECT name, description, enabled, source, spec, recognizer, matcher, selector, renderer
		FROM rules
		WHERE name = ?
	`, name).Scan(
		&rule.Name,
		&rule.Description,
		&enabled,
		&rule.Source,
		&rule.Spec,
		&rule.Recognizer,
		&rule.Matcher,
		&rule.Selector,
		&rule.Renderer,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return rules.RuleSpec{}, errors.New("unknown rule \"" + name + "\"")
	}
	if err != nil {
		return rules.RuleSpec{}, err
	}

	rule.Enabled = enabled
	return rule, nil
}

func (s *Store) AddRule(rule rules.RuleSpec) error {
	if err := rules.ValidateRuleName(rule.Name); err != nil {
		return err
	}
	exists, err := s.ruleExists(rule.Name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("rule %q already exists", rule.Name)
	}
	if rule.Description == "" {
		return errors.New("rule description is required")
	}
	if rule.Source == "" {
		rule.Source = "user"
	}

	_, err = s.db.Exec(`
		INSERT INTO rules (name, description, enabled, source, spec, recognizer, matcher, selector, renderer)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rule.Name, rule.Description, rule.Enabled, rule.Source, rule.Spec, rule.Recognizer, rule.Matcher, rule.Selector, rule.Renderer)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) ListRuleNames() ([]string, error) {
	rows, err := s.db.Query(`SELECT name FROM rules ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, name)
	}

	return result, rows.Err()
}

func (s *Store) ListEnabledRules() ([]rules.RuleSpec, error) {
	rows, err := s.db.Query(`
		SELECT name, description, enabled, source, spec, recognizer, matcher, selector, renderer
		FROM rules
		WHERE enabled = 1
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]rules.RuleSpec, 0)
	for rows.Next() {
		var rule rules.RuleSpec
		var enabled bool
		if err := rows.Scan(
			&rule.Name,
			&rule.Description,
			&enabled,
			&rule.Source,
			&rule.Spec,
			&rule.Recognizer,
			&rule.Matcher,
			&rule.Selector,
			&rule.Renderer,
		); err != nil {
			return nil, err
		}
		rule.Enabled = enabled
		result = append(result, rule)
	}

	return result, rows.Err()
}

func (s *Store) UpdateRule(name string, description *string, enabled *bool, spec, recognizer, matcher, selector, renderer *string) error {
	updates := make([]string, 0, 2)
	args := make([]any, 0, 7)

	if description != nil {
		if *description == "" {
			return errors.New("rule description is required")
		}
		updates = append(updates, "description = ?")
		args = append(args, *description)
	}
	if enabled != nil {
		updates = append(updates, "enabled = ?")
		args = append(args, *enabled)
	}
	if spec != nil {
		updates = append(updates, "spec = ?")
		args = append(args, *spec)
	}
	if recognizer != nil {
		updates = append(updates, "recognizer = ?")
		args = append(args, *recognizer)
	}
	if matcher != nil {
		updates = append(updates, "matcher = ?")
		args = append(args, *matcher)
	}
	if selector != nil {
		updates = append(updates, "selector = ?")
		args = append(args, *selector)
	}
	if renderer != nil {
		updates = append(updates, "renderer = ?")
		args = append(args, *renderer)
	}
	if len(updates) == 0 {
		return errors.New("rule update requires at least one change")
	}

	args = append(args, name)
	result, err := s.db.Exec(
		fmt.Sprintf("UPDATE rules SET %s, updated_at = CURRENT_TIMESTAMP WHERE name = ?", strings.Join(updates, ", ")),
		args...,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("unknown rule %q", name)
	}
	return nil
}

func (s *Store) RemoveRule(name string) error {
	result, err := s.db.Exec(`DELETE FROM rules WHERE name = ?`, name)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("unknown rule %q", name)
	}
	return nil
}

func (s *Store) init() error {
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS registry_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}

	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS rules (
			name TEXT PRIMARY KEY,
			description TEXT NOT NULL,
			enabled BOOLEAN NOT NULL,
			source TEXT NOT NULL,
			spec TEXT NOT NULL DEFAULT '',
			recognizer TEXT NOT NULL DEFAULT '',
			matcher TEXT NOT NULL DEFAULT '',
			selector TEXT NOT NULL DEFAULT '',
			renderer TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}

	for _, column := range []string{"spec", "recognizer", "matcher", "selector", "renderer"} {
		if err := s.ensureRuleColumn(column); err != nil {
			return err
		}
	}

	if err := s.migrateLegacyBuiltinSource(); err != nil {
		return err
	}
	if err := s.migrateSeededDefinitions(); err != nil {
		return err
	}

	seedVersion, err := s.metaValue("seed_version")
	if err != nil {
		return err
	}
	if seedVersion == "1" {
		return nil
	}

	return s.seedBuiltins("1")
}

func (s *Store) migrateLegacyBuiltinSource() error {
	for _, rule := range rules.Builtins() {
		if _, err := s.db.Exec(`
			UPDATE rules
			SET source = 'seeded'
			WHERE name = ? AND source = 'builtin'
		`, rule.Name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) migrateSeededDefinitions() error {
	for _, rule := range rules.Builtins() {
		if _, err := s.db.Exec(`
			UPDATE rules
			SET spec = CASE WHEN spec = '' THEN ? ELSE spec END,
				recognizer = CASE WHEN recognizer = '' THEN ? ELSE recognizer END,
				matcher = CASE WHEN matcher = '' THEN ? ELSE matcher END,
				selector = CASE WHEN selector = '' THEN ? ELSE selector END,
				renderer = CASE WHEN renderer = '' THEN ? ELSE renderer END
			WHERE name = ? AND source = 'seeded'
		`, rule.Spec, rule.Recognizer, rule.Matcher, rule.Selector, rule.Renderer, rule.Name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) seedBuiltins(version string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, rule := range rules.Builtins() {
		exists, err := ruleExistsTx(tx, rule.Name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		if _, err := tx.Exec(`
			INSERT INTO rules (name, description, enabled, source, spec, recognizer, matcher, selector, renderer)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, rule.Name, rule.Description, rule.Enabled, rule.Source, rule.Spec, rule.Recognizer, rule.Matcher, rule.Selector, rule.Renderer); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO registry_meta (key, value, updated_at)
		VALUES ('seed_version', ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, version); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) metaValue(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM registry_meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *Store) ruleExists(name string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM rules WHERE name = ?)`, name).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func ruleExistsTx(tx *sql.Tx, name string) (bool, error) {
	var exists bool
	err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM rules WHERE name = ?)`, name).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *Store) ensureRuleColumn(column string) error {
	if _, err := s.db.Exec(fmt.Sprintf(`ALTER TABLE rules ADD COLUMN %s TEXT NOT NULL DEFAULT ''`, column)); err != nil {
		if strings.Contains(err.Error(), "duplicate column name") {
			return nil
		}
		return err
	}
	return nil
}
