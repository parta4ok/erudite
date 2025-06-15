package envsettings

import (
	_ "embed"

	"github.com/joho/godotenv"
)

//go:embed .env
var envdata string

func Getenv(envName string) string {
	envMap, _ := godotenv.Unmarshal(envdata)
	return envMap[envName]
}
