package logging

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("apertura %s: %v", path, err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func records(t *testing.T, path string) []map[string]any {
	t.Helper()
	lines := readLines(t, path)
	out := make([]map[string]any, 0, len(lines))
	for i, line := range lines {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("%s riga %d non è JSON valido: %v (%s)", path, i+1, err, line)
		}
		out = append(out, rec)
	}
	return out
}

func TestFilesAreSeparated(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "help-alarm.log")
	alertPath := filepath.Join(dir, "help-alarm-alerts.log")

	l, err := New(Options{Path: logPath, AlertPath: alertPath, Level: "debug"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	l.Console.Info("avvio help-alarm", "keywords", "aiuto")
	l.Console.Info("trascrizione", "testo", "per favore aiuto")
	l.Alerts.Info("aiuto rilevato, luce accesa", "keyword", "aiuto")

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	normale := records(t, logPath)
	if len(normale) != 2 {
		t.Fatalf("il log normale ha %d righe, attese 2: %+v", len(normale), normale)
	}
	for _, rec := range normale {
		if rec["msg"] == "aiuto rilevato, luce accesa" {
			t.Error("il messaggio di allarme è finito nel log dei messaggi normali")
		}
	}

	allarmi := records(t, alertPath)
	if len(allarmi) != 1 {
		t.Fatalf("il log allarmi ha %d righe, attese 1: %+v", len(allarmi), allarmi)
	}
	if allarmi[0]["msg"] != "aiuto rilevato, luce accesa" {
		t.Errorf("msg = %v", allarmi[0]["msg"])
	}
	if allarmi[0]["keyword"] != "aiuto" {
		t.Errorf("keyword = %v", allarmi[0]["keyword"])
	}
	if _, ok := allarmi[0]["time"]; !ok {
		t.Error("il record di allarme non ha il timestamp")
	}
}

func TestAlertsFallBackToMainFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "only.log")

	// AlertPath vuota: entrambi i logger scrivono sullo stesso file.
	l, err := New(Options{Path: logPath, Level: "info"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Console.Info("avvio")
	l.Alerts.Info("aiuto rilevato, luce accesa")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if got := len(records(t, logPath)); got != 2 {
		t.Errorf("il file contiene %d righe, attese 2", got)
	}
}

func TestNewAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "existing.log")
	if err := os.WriteFile(logPath, []byte("{\"preesistente\":true}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	l, err := New(Options{Path: logPath, Level: "info"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Console.Info("nuovo")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	recs := records(t, logPath)
	if len(recs) != 2 {
		t.Fatalf("il file ha %d righe, attese 2 (append, non truncate)", len(recs))
	}
	if recs[1]["msg"] != "nuovo" {
		t.Errorf("l'ultima riga non è quella appena scritta: %+v", recs[1])
	}
}

func TestNewCreatesMissingDirectories(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "sub", "a.log")
	alertPath := filepath.Join(dir, "logs", "sub", "b.log")

	l, err := New(Options{Path: logPath, AlertPath: alertPath, Level: "info"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Console.Info("avvio")
	l.Alerts.Info("allarme")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(records(t, logPath)) != 1 || len(records(t, alertPath)) != 1 {
		t.Error("i file non sono stati scritti nelle directory create")
	}
}

func TestInvalidLevel(t *testing.T) {
	dir := t.TempDir()
	_, err := New(Options{Path: filepath.Join(dir, "a.log"), Level: "verboso"})
	if err == nil {
		t.Error("New ha accettato un livello di log inesistente")
	}
}

func TestLevelFiltersDebug(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "a.log")

	l, err := New(Options{Path: logPath, Level: "info"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Console.Debug("rumoroso", "dettaglio", 1)
	l.Console.Info("visibile")
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	recs := records(t, logPath)
	if len(recs) != 1 {
		t.Fatalf("il log ha %d righe, attesa 1: %+v", len(recs), recs)
	}
	if recs[0]["msg"] != "visibile" {
		t.Errorf("msg = %v", recs[0]["msg"])
	}
}
