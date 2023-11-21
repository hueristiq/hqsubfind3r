package configuration

import (
	"os"
	"path/filepath"

	"dario.cat/mergo"
	"github.com/hueristiq/hqgolog"
	"github.com/hueristiq/xsubfind3r/pkg/scraper/sources"
	"github.com/logrusorgru/aurora/v3"
	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Version string       `yaml:"version"`
	Sources []string     `yaml:"sources"`
	Keys    sources.Keys `yaml:"keys"`
}

func (cfg *Configuration) Write(path string) (err error) {
	var file *os.File

	directory := filepath.Dir(path)
	identation := 4

	if _, err = os.Stat(directory); os.IsNotExist(err) {
		if directory != "" {
			if err = os.MkdirAll(directory, os.ModePerm); err != nil {
				return
			}
		}
	}

	file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return
	}

	defer file.Close()

	enc := yaml.NewEncoder(file)
	enc.SetIndent(identation)
	err = enc.Encode(&cfg)

	return
}

const (
	NAME    string = "xsubfind3r"
	VERSION string = "0.5.0"
)

var (
	BANNER = aurora.Sprintf(
		aurora.BrightBlue(`
                _      __ _           _ _____      
__  _____ _   _| |__  / _(_)_ __   __| |___ / _ __ 
\ \/ / __| | | | '_ \| |_| | '_ \ / _`+"`"+` | |_ \| '__|
 >  <\__ \ |_| | |_) |  _| | | | | (_| |___) | |   
/_/\_\___/\__,_|_.__/|_| |_|_| |_|\__,_|____/|_| 
                                             %s

                   %s`).Bold(),
		aurora.BrightRed("v"+VERSION).Bold(),
		aurora.BrightYellow("with <3 by Hueristiq Open Source").Italic(),
	)
	UserDotConfigDirectoryPath = func() (userDotConfig string) {
		var err error

		userDotConfig, err = os.UserConfigDir()
		if err != nil {
			hqgolog.Fatal().Msg(err.Error())
		}

		return
	}()
	projectRootDirectoryName = NAME
	ProjectRootDirectoryPath = filepath.Join(UserDotConfigDirectoryPath, projectRootDirectoryName)
	configurationFileName    = "config.yaml"
	ConfigurationFilePath    = filepath.Join(ProjectRootDirectoryPath, configurationFileName)
)

func CreateUpdate(path string) (err error) {
	var cfg Configuration

	defaultConfig := Configuration{
		Version: VERSION,
		Sources: sources.List,
		Keys: sources.Keys{
			Bevigil:   []string{},
			BuiltWith: []string{},
			Chaos:     []string{},
			Fullhunt:  []string{},
			GitHub:    []string{},
			Intelx:    []string{},
			Shodan:    []string{},
			URLScan:   []string{},
		},
	}

	_, err = os.Stat(path)

	switch {
	case err != nil && os.IsNotExist(err):
		cfg = defaultConfig

		if err = cfg.Write(path); err != nil {
			return
		}
	case err != nil:
		return
	default:
		cfg, err = Read(path)
		if err != nil {
			return
		}

		if cfg.Version != VERSION || len(cfg.Sources) != len(sources.List) {
			if err = mergo.Merge(&cfg, defaultConfig); err != nil {
				return
			}

			cfg.Version = VERSION
			cfg.Sources = sources.List

			if err = cfg.Write(path); err != nil {
				return
			}
		}
	}

	return
}

func Read(path string) (cfg Configuration, err error) {
	var file *os.File

	file, err = os.Open(path)
	if err != nil {
		return
	}

	defer file.Close()

	if err = yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return
	}

	return
}
