package content

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kelaia/vxaos/server/internal/rvdata"
)

// Options selects the data directory (client data with Scripts.rvdata2) and the
// content source directory.
type Options struct {
	DataDir    string
	ContentDir string
	Locale     string
}

// Result summarizes a build or check.
type Result struct {
	ScriptsPath  string
	VocabEntries int
	Quests       int
	Enums        int
	VocabSource  []byte
	QuestSource  []byte
	EnumSource   []byte
	ConfigSource []byte
}

// Check renders the generated scripts without writing them.
func Check(opts Options) (*Result, error) {
	return run(opts, false)
}

// Build renders the generated modules and replaces them inside
// Scripts.rvdata2. The previous archive is kept as Scripts.rvdata2.bak.
func Build(opts Options) (*Result, error) {
	return run(opts, true)
}

func run(opts Options, write bool) (*Result, error) {
	if opts.Locale == "" {
		opts.Locale = "ptBR"
	}
	if opts.DataDir == "" || opts.ContentDir == "" {
		return nil, fmt.Errorf("content: data and content directories are required")
	}
	scriptsPath := filepath.Join(opts.DataDir, "Scripts.rvdata2")
	scripts, err := rvdata.LoadScripts(scriptsPath)
	if err != nil {
		return nil, err
	}
	vocabPath := filepath.Join(opts.ContentDir, "locales", opts.Locale+".csv")
	vocab, err := LoadVocabCSV(vocabPath)
	if err != nil {
		return nil, err
	}
	vocabRuby, err := RenderVocab(vocab)
	if err != nil {
		return nil, err
	}
	questsPath := filepath.Join(opts.ContentDir, "quests.toml")
	quests, err := LoadQuests(questsPath)
	if err != nil {
		return nil, err
	}
	questRuby, err := RenderQuests(quests)
	if err != nil {
		return nil, err
	}
	enumsPath := filepath.Join(opts.ContentDir, "enums.toml")
	enums, err := LoadEnums(enumsPath)
	if err != nil {
		return nil, err
	}
	enumRuby, err := RenderEnums(enums)
	if err != nil {
		return nil, err
	}
	configsPath := filepath.Join(opts.ContentDir, "configs.toml")
	configs, err := LoadConfigs(configsPath)
	if err != nil {
		return nil, err
	}
	configRuby, err := RenderConfigs(configs)
	if err != nil {
		return nil, err
	}
	if !rvdata.SetScriptSource(scripts, "[VS] Enums", enumRuby) {
		return nil, fmt.Errorf("content: Scripts.rvdata2 has no [VS] Enums entry")
	}
	if !rvdata.SetScriptSource(scripts, "[VS] Configs", configRuby) {
		return nil, fmt.Errorf("content: Scripts.rvdata2 has no [VS] Configs entry")
	}
	if !rvdata.SetScriptSource(scripts, "[VS] Vocab", vocabRuby) {
		return nil, fmt.Errorf("content: Scripts.rvdata2 has no [VS] Vocab entry")
	}
	if !rvdata.SetScriptSource(scripts, "[VS] Quests", questRuby) {
		return nil, fmt.Errorf("content: Scripts.rvdata2 has no [VS] Quests entry")
	}
	result := &Result{
		ScriptsPath:  scriptsPath,
		VocabEntries: len(vocab),
		Quests:       len(quests),
		Enums:        len(enums.Enums),
		VocabSource:  vocabRuby,
		QuestSource:  questRuby,
		EnumSource:   enumRuby,
		ConfigSource: configRuby,
	}
	if !write {
		return result, nil
	}
	data, err := rvdata.EncodeScripts(scripts)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(scriptsPath, data); err != nil {
		return nil, err
	}
	// Never leave a build that our own reader cannot load.
	if _, err := rvdata.LoadScripts(scriptsPath); err != nil {
		return nil, fmt.Errorf("content: generated Scripts.rvdata2 is unreadable: %w", err)
	}
	return result, nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vxaos-content-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Keep one recoverable copy of the previous archive.
	if old, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(path+".bak", old, 0o644); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpName, path); err == nil {
		return nil
	}
	// Windows can refuse to replace an existing file; the backup is already in
	// place, so removing the destination is safe.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmpName, path)
}
