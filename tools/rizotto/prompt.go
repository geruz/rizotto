package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// maxAttempts limits how many times a question is repeated after a wrong answer.
const maxAttempts = 5

var (
	errNoInput         = errors.New("no answer on stdin, pass the value as a flag instead")
	errTooManyAttempts = errors.New("too many invalid answers")
)

// prompter asks the author the questions of make-project. One reader is shared by
// every question so that buffered input is not lost between them.
type prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func newPrompter(in io.Reader, out io.Writer) *prompter {
	return &prompter{in: bufio.NewReader(in), out: out}
}

// ask prints "question [hint]: " and returns the trimmed answer, or fallback when
// the author just hits enter.
func (p *prompter) ask(question string, hint string, fallback string) (string, error) {
	suffix := ": "
	if hint != "" {
		suffix = " [" + hint + "]: "
	}

	_, _ = fmt.Fprint(p.out, question+suffix)

	line, err := p.in.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read answer: %w", err)
	}

	answer := strings.TrimSpace(line)
	if answer != "" {
		return answer, nil
	}

	if fallback == "" && errors.Is(err, io.EOF) {
		_, _ = fmt.Fprintln(p.out)

		return "", errNoInput
	}

	return fallback, nil
}

// askValid repeats the question until normalize accepts the answer.
func (p *prompter) askValid(
	question string,
	defaultValue string,
	normalize func(string) (string, error),
) (string, error) {
	for range maxAttempts {
		answer, err := p.ask(question, defaultValue, defaultValue)
		if err != nil {
			return "", err
		}

		value, err := normalize(answer)
		if err == nil {
			return value, nil
		}

		_, _ = fmt.Fprintln(p.out, "  "+err.Error())
	}

	return "", errTooManyAttempts
}

func (p *prompter) askYesNo(question string, defaultValue bool) (bool, error) {
	hint, fallback := "y/N", "n"
	if defaultValue {
		hint, fallback = "Y/n", "y"
	}

	answer, err := p.ask(question, hint, fallback)
	if err != nil {
		return false, err
	}

	value, ok := parseYesNo(answer)
	if !ok {
		return defaultValue, nil
	}

	return value, nil
}

func parseYesNo(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes", "true", "1", "д", "да":
		return true, true
	case "n", "no", "false", "0", "н", "нет":
		return false, true
	default:
		return false, false
	}
}
