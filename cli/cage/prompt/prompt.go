package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/loilo-inc/canarycage/v5/env"
)

type Prompter struct {
	Reader *bufio.Reader
}

func NewPrompter(stdin io.Reader) *Prompter {
	return &Prompter{Reader: bufio.NewReader(stdin)}
}

func (s *Prompter) Confirm(
	name string,
	value string,
) error {
	fmt.Fprintf(os.Stderr, "please confirm [%s]: ", name)
	if text, err := s.Reader.ReadString('\n'); err != nil {
		return fmt.Errorf("failed to read from stdin: %w", err)
	} else if text[:len(text)-1] != value {
		return fmt.Errorf("%s is not matched. expected: %s", name, value)
	}
	return nil
}

func (s *Prompter) ConfirmTask(
	envars *env.Envars,
) error {
	return s.confirmStackChange(envars, false)
}

func (s *Prompter) ConfirmService(
	envars *env.Envars,
) error {
	return s.confirmStackChange(envars, true)
}

func (s *Prompter) confirmStackChange(
	envars *env.Envars,
	service bool,
) error {
	if err := s.Confirm("region", envars.Region); err != nil {
		return err
	}
	if err := s.Confirm("cluster", envars.Cluster); err != nil {
		return err
	}
	if service {
		if err := s.Confirm("service", envars.Service); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "confirm changes:\n")
	fmt.Fprintf(os.Stderr, "[region]: %s\n", envars.Region)
	fmt.Fprintf(os.Stderr, "[cluster]: %s\n", envars.Cluster)
	if service {
		fmt.Fprintf(os.Stderr, "[service]: %s\n", envars.Service)
	}
	if err := s.Confirm("yes", "yes"); err != nil {
		return err
	}
	return nil
}
