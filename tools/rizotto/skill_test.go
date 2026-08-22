package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shoenig/test/must"
)

func Test_runSkill_PrintsTheProjectSkill(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	err := runSkill(nil, out)
	must.NoError(t, err)

	printed := out.String()
	must.StrHasPrefix(t, "# rizotto", printed)
	must.StrContains(t, printed, "rizotto service skill")
	must.StrContains(t, printed, "rizotto controller skill")
	must.StrContains(t, printed, "server.go")
}

func Test_runSkill_TakesNoArguments(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	err := runSkill([]string{"whatever"}, out)
	must.ErrorIs(t, err, errUsage)
	must.StrContains(t, err.Error(), `"rizotto service skill"`)
	must.StrContains(t, err.Error(), `"rizotto controller skill"`)
}

func Test_runService_PrintsTheServiceSkill(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	err := runService([]string{skillCommand}, strings.NewReader(""), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrHasPrefix(t, "# Services", printed)
	must.StrContains(t, printed, "bind-method")
	must.StrContains(t, printed, "RegisterServer")
	must.StrContains(t, printed, "bb.NewNotFoundError")
	must.StrContains(t, printed, "task migrate-up")
}

func Test_runController_PrintsTheControllerSkill(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}

	err := runController([]string{skillCommand}, strings.NewReader(""), out)
	must.NoError(t, err)

	printed := out.String()
	must.StrHasPrefix(t, "# Controllers", printed)
	must.StrContains(t, printed, "gateway.JSONMethod")
	must.StrContains(t, printed, `in:"path=order_id"`)
	must.StrContains(t, printed, "PrivateArea")
	must.StrContains(t, printed, "RouteTable()")
	must.StrContains(t, printed, "openapi.yml")
}

func Test_printSkill_UnknownSkill(t *testing.T) {
	t.Parallel()

	err := printSkill(&bytes.Buffer{}, "repository")
	must.ErrorIs(t, err, errUnknownSkill)
}
