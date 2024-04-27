package env

import (
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func MustLoadEnvFile(envName string) {
	envFiles := []string{} // by default .env
	if envName != "" {
		envFiles = append(envFiles, envName)
	}

	for {
		err := godotenv.Load(envFiles...)
		if err != nil {
			panic(err)
		}

		if len(envFiles) > 1 { // only one import level
			return
		}

		extends := os.Getenv("IMPORT_ENV_FILES")
		if extends != "" {
			extends = path.Join(path.Dir(envName), extends)
			envFiles = strings.Split(extends, ",")
			envFiles = append(envFiles, envName)
		} else {
			break
		}
	}
}

func MustGetEnumValue[T ~string](key string, values []T) T {
	value := os.Getenv(key)
	if value == "" {
		panic("Missing environment variable: " + key)
	}

	for _, v := range values {
		if strings.EqualFold(strings.ToLower(string(v)), strings.ToLower(value)) {
			return v
		}
	}

	panic("Invalid value for environment variable: " + key)
}

func MustGetStringValue(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("Missing environment variable: " + key)
	}

	return value
}

func MustGetStringsArrayValue(key string, separator string) []string {
	value := os.Getenv(key)
	if value == "" {
		panic("Missing environment variable: " + key)
	}

	values := strings.Split(value, separator)

	return values
}

func GetStringValue(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func MustGetIntValue(key string) int {
	value := MustGetStringValue(key)

	converted, err := strconv.Atoi(value)
	if err != nil {
		panic("Invalid value for environment variable: " + key)
	}

	return converted
}

func GetIntValue(key string, defaultValue int) int {
	value := GetStringValue(key, "")
	if value == "" {
		return defaultValue
	}

	converted, err := strconv.Atoi(value)
	if err != nil {
		panic("Invalid value for environment variable: " + key)
	}

	return converted
}

func MustGetURLValue(key string) string {
	value := MustGetStringValue(key)
	if value == "" {
		panic("Missing environment variable: " + key)
	}

	_, err := url.Parse(value)
	if err != nil {
		panic("Invalid value for environment variable: " + key + ". Value is not a valid url")
	}

	return value
}

func GetURLValue(key string, defaultURL string) string {
	value := GetStringValue(key, defaultURL)

	_, err := url.Parse(value)
	if err != nil {
		panic("Invalid value for environment variable: " + key + ". Value is not a valid url")
	}

	return value
}

func MustGetBoolValue(key string) bool {
	value := MustGetStringValue(key)

	converted, err := strconv.ParseBool(value)
	if err != nil {
		panic("Invalid value for environment variable: " + key)
	}

	return converted
}

// GetBoolValue tries to get a value from env, and convert it to true or false via strconv.ParseBool.
// If the value is not present, defaultValue is returned.
func GetBoolValue(key string, defaultValue bool) bool {
	envValue := os.Getenv(key)

	converted, err := strconv.ParseBool(envValue)
	if err != nil {
		return defaultValue
	}

	return converted
}
