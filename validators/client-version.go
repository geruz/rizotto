package validators

import (
	"github.com/Masterminds/semver/v3"
)

func IsVersionEqualOrLower(currentVersion, targetVersion string) (bool, error) {
	c, err := semver.NewConstraint("<= " + targetVersion)
	if err != nil {
		return false, err
	}

	v, err := semver.NewVersion(currentVersion)
	if err != nil {
		return false, err
	}

	return c.Check(v), nil
}

func IsVersionLower(currentVersion, targetVersion string) (bool, error) {
	c, err := semver.NewConstraint("< " + targetVersion)
	if err != nil {
		return false, err
	}

	v, err := semver.NewVersion(currentVersion)
	if err != nil {
		return false, err
	}

	return c.Check(v), nil
}
