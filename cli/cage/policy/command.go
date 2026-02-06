package policy

import (
	"encoding/json"
	"io"
)

type Command struct {
	Writer io.Writer
	Short  bool
}

func NewCommand(writer io.Writer, short bool) *Command {
	return &Command{Writer: writer, Short: short}
}

func (c *Command) Run() error {
	doc := DefaultDocument()
	var (
		out []byte
		err error
	)
	if !c.Short {
		out, err = json.MarshalIndent(doc, "", "  ")
	} else {
		out, err = json.Marshal(doc)
	}
	if err != nil {
		return err
	}
	_, err = c.Writer.Write(append(out, '\n'))
	return err
}
