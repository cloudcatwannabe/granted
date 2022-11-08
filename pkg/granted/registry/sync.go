package registry

import (
	"github.com/common-fate/clio"
	grantedConfig "github.com/common-fate/granted/pkg/config"
	"github.com/urfave/cli/v2"
)

var SyncCommand = cli.Command{
	Name:        "sync",
	Description: "Pull the latest change from remote origin and sync aws profiles in aws config files. For more click here https://github.com/common-fate/rfds/discussions/2",
	Action: func(c *cli.Context) error {

		if err := SyncProfileRegistries(); err != nil {
			return err
		}

		return nil
	},
}

func SyncProfileRegistries() error {
	gConf, err := grantedConfig.Load()
	if err != nil {
		return err
	}

	if len(gConf.ProfileRegistryURLS) < 1 {
		clio.Warn("granted registry not configured. Try adding a git repository with 'granted registry add <https://github.com/your-org/your-registry.git>'")
	}

	awsConfigPath, err := getDefaultAWSConfigLocation()
	if err != nil {
		return err
	}

	configFile, err := loadAWSConfigFile()
	if err != nil {
		return err
	}

	// if the config file contains granted generated content then remove it
	if err := removeAutogeneratedProfiles(configFile, awsConfigPath); err != nil {
		return err
	}

	for index, repoURL := range gConf.ProfileRegistryURLS {
		u, err := parseGitURL(repoURL)
		if err != nil {
			return err
		}

		repoDirPath, err := getRegistryLocation(u)
		if err != nil {
			return err
		}

		if err = gitPull(repoDirPath, false); err != nil {
			return err
		}

		if err = parseClonedRepo(repoDirPath, repoURL); err != nil {
			return err
		}

		var r Registry
		_, err = r.Parse(repoDirPath)
		if err != nil {
			return err
		}

		isFirstSection := false
		if index == 0 {
			isFirstSection = true
		}

		if err := Sync(r, repoURL, repoDirPath, isFirstSection); err != nil {
			return err
		}
	}

	return nil
}

func Sync(r Registry, repoURL string, repoDirPath string, isFirstSection bool) error {
	clio.Debugf("syncing %s \n", repoURL)

	awsConfigPath, err := getDefaultAWSConfigLocation()
	if err != nil {
		return err
	}

	awsConfigFile, err := loadAWSConfigFile()
	if err != nil {
		return err
	}

	clonedFile, err := loadClonedConfigs(r, repoDirPath)
	if err != nil {
		return err
	}

	err = generateNewRegistrySection(awsConfigFile, clonedFile, repoURL, isFirstSection)
	if err != nil {
		return err
	}

	awsConfigFile.SaveTo(awsConfigPath)

	clio.Debug("Changes saved to aws config file.")

	clio.Infof("successfully synced registry %s", repoURL)

	return nil
}
