package ui

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var ptermMu sync.Mutex

type Session struct {
	out         io.Writer
	err         io.Writer
	in          io.Reader
	outTTY      bool
	errTTY      bool
	inputTTY    bool
	interactive bool
}

type Table struct {
	Title   string
	Columns []string
	Rows    [][]string
	Writer  io.Writer
}

type TaskOptions struct {
	UseErrorWriter bool
	SuccessText    string
	FailureText    string
}

func New(cmd *cobra.Command) *Session {
	out := cmd.OutOrStdout()
	err := cmd.ErrOrStderr()
	in := cmd.InOrStdin()

	outTTY := isTerminalWriter(out)
	errTTY := isTerminalWriter(err)
	inputTTY := isTerminalReader(in)

	return &Session{
		out:         out,
		err:         err,
		in:          in,
		outTTY:      outTTY,
		errTTY:      errTTY,
		inputTTY:    inputTTY,
		interactive: inputTTY && outTTY,
	}
}

func PrintError(w io.Writer, err error) {
	if err == nil {
		return
	}

	styled := isTerminalWriter(w)
	withPtermState(styled, func() {
		pterm.Error.WithWriter(w).Printfln("%v", err)
	})
}

func (s *Session) Interactive() bool {
	return s.interactive
}

func (s *Session) Header(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}

	withPtermState(s.outTTY, func() {
		pterm.DefaultHeader.
			WithWriter(s.out).
			WithFullWidth(false).
			WithMargin(0).
			Printfln("%s", text)
	})
}

func (s *Session) Section(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}

	withPtermState(s.outTTY, func() {
		pterm.DefaultSection.
			WithWriter(s.out).
			WithTopPadding(0).
			WithBottomPadding(0).
			Printfln("%s", text)
	})
}

func (s *Session) Paragraph(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}

	withPtermState(s.outTTY, func() {
		pterm.Fprintln(s.out, text)
	})
}

func (s *Session) Blank() {
	withPtermState(s.outTTY, func() {
		pterm.Fprintln(s.out)
	})
}

func (s *Session) Infof(format string, args ...any) {
	s.prefixf(s.out, s.outTTY, pterm.Info, format, args...)
}

func (s *Session) Successf(format string, args ...any) {
	s.prefixf(s.out, s.outTTY, pterm.Success, format, args...)
}

func (s *Session) Warningf(format string, args ...any) {
	s.prefixf(s.err, s.errTTY, pterm.Warning, format, args...)
}

func (s *Session) Errorf(format string, args ...any) {
	s.prefixf(s.err, s.errTTY, pterm.Error, format, args...)
}

func (s *Session) KeyValues(title string, rows [][2]string) error {
	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, []string{row[0], row[1]})
	}

	return s.RenderTable(Table{
		Title:  title,
		Rows:   tableRows,
		Writer: s.out,
	})
}

func (s *Session) RenderTable(table Table) error {
	writer := table.Writer
	if writer == nil {
		writer = s.out
	}
	styled := s.outTTY
	if writer == s.err {
		styled = s.errTTY
	}

	if table.Title != "" {
		withPtermState(styled, func() {
			pterm.DefaultSection.
				WithWriter(writer).
				WithTopPadding(0).
				WithBottomPadding(0).
				Printfln("%s", table.Title)
		})
	}

	data := make(pterm.TableData, 0, len(table.Rows)+1)
	if len(table.Columns) > 0 {
		data = append(data, sanitizeRow(table.Columns))
	}
	for _, row := range table.Rows {
		data = append(data, sanitizeRow(row))
	}

	withPtermState(styled, func() {
		printer := pterm.DefaultTable.
			WithWriter(writer).
			WithData(data).
			WithHasHeader(len(table.Columns) > 0).
			WithLeftAlignment()

		if !styled {
			printer = printer.
				WithSeparator("  ").
				WithHeaderRowSeparator("").
				WithRowSeparator("")
		}

		_ = printer.Render()
	})

	return nil
}

func (s *Session) PromptSelect(label string, options []string, defaultValue string) (string, error) {
	if s.interactive && sameWriter(s.out, os.Stdout) {
		var result string
		var err error
		withDefaultOutput(s.out, s.outTTY, func() {
			result, err = pterm.DefaultInteractiveSelect.
				WithOptions(slices.Clone(options)).
				WithDefaultOption(defaultValue).
				WithMaxHeight(len(options)).
				Show(label)
		})
		return result, err
	}

	withPtermState(s.outTTY, func() {
		pterm.DefaultSection.
			WithWriter(s.out).
			WithTopPadding(0).
			WithBottomPadding(0).
			Printfln("%s", label)
	})

	for idx, option := range options {
		withPtermState(s.outTTY, func() {
			pterm.Fprintln(s.out, fmt.Sprintf("%d. %s", idx+1, option))
		})
	}
	withPtermState(s.outTTY, func() {
		pterm.Fprint(s.out, "> ")
	})

	buf := make([]byte, 0)
	tmp := make([]byte, 1)
	for {
		n, readErr := s.in.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if tmp[0] == '\n' {
				break
			}
		}
		if readErr != nil {
			if len(buf) == 0 {
				return "", readErr
			}
			break
		}
	}

	return strings.TrimSpace(string(buf)), nil
}

func (s *Session) RunTask(text string, opts TaskOptions, fn func() error) error {
	writer := s.out
	styled := s.outTTY
	if opts.UseErrorWriter {
		writer = s.err
		styled = s.errTTY
	}

	if !styled || !sameWriter(writer, os.Stdout) && !sameWriter(writer, os.Stderr) {
		return fn()
	}

	var runErr error
	withPtermState(styled, func() {
		spinner, err := pterm.DefaultSpinner.
			WithWriter(writer).
			WithRemoveWhenDone(true).
			Start(text)
		if err != nil {
			runErr = fn()
			return
		}

		runErr = fn()
		if runErr != nil {
			failText := opts.FailureText
			if strings.TrimSpace(failText) == "" {
				failText = text
			}
			spinner.Fail(failText)
			return
		}

		successText := opts.SuccessText
		if strings.TrimSpace(successText) == "" {
			successText = text
		}
		spinner.Success(successText)
	})

	return runErr
}

func (s *Session) RenderHelp(cmd *cobra.Command) error {
	s.Header(cmd.CommandPath())

	description := strings.TrimSpace(cmd.Long)
	if description == "" {
		description = strings.TrimSpace(cmd.Short)
	}
	if description != "" {
		s.Paragraph(description)
		s.Blank()
	}

	if cmd.Runnable() {
		if err := s.KeyValues("Usage", [][2]string{{"Command", cmd.UseLine()}}); err != nil {
			return err
		}
		s.Blank()
	}

	if rows := commandRows(cmd); len(rows) > 0 {
		if err := s.RenderTable(Table{
			Title:   "Commands",
			Columns: []string{"Name", "Description"},
			Rows:    rows,
		}); err != nil {
			return err
		}
		s.Blank()
	}

	if rows := flagRows(cmd.LocalFlags()); len(rows) > 0 {
		if err := s.RenderTable(Table{
			Title:   "Flags",
			Columns: []string{"Flag", "Description", "Default"},
			Rows:    rows,
		}); err != nil {
			return err
		}
		s.Blank()
	}

	if rows := flagRows(cmd.InheritedFlags()); len(rows) > 0 {
		if err := s.RenderTable(Table{
			Title:   "Global Flags",
			Columns: []string{"Flag", "Description", "Default"},
			Rows:    rows,
		}); err != nil {
			return err
		}
		s.Blank()
	}

	if cmd.HasAvailableSubCommands() {
		s.Infof("Use %q for more information about a command.", cmd.CommandPath()+" [command] --help")
	}

	return nil
}

func commandRows(cmd *cobra.Command) [][]string {
	commands := cmd.Commands()
	rows := make([][]string, 0, len(commands))
	for _, sub := range commands {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		rows = append(rows, []string{sub.Name(), sub.Short})
	}
	return rows
}

func flagRows(flags *pflag.FlagSet) [][]string {
	if flags == nil {
		return nil
	}

	rows := [][]string{}
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden {
			return
		}

		name := "--" + flag.Name
		if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
			name = "-" + flag.Shorthand + ", " + name
		}
		rows = append(rows, []string{name, flag.Usage, flag.DefValue})
	})
	return rows
}

func (s *Session) prefixf(writer io.Writer, styled bool, printer pterm.PrefixPrinter, format string, args ...any) {
	withPtermState(styled, func() {
		printer.WithWriter(writer).Printfln(format, args...)
	})
}

func sanitizeRow(row []string) []string {
	sanitized := make([]string, 0, len(row))
	for _, cell := range row {
		value := strings.TrimSpace(cell)
		if value == "" {
			value = "-"
		}
		sanitized = append(sanitized, value)
	}
	return sanitized
}

func withDefaultOutput(writer io.Writer, styled bool, fn func()) {
	ptermMu.Lock()
	defer ptermMu.Unlock()

	oldRaw := pterm.RawOutput
	if styled {
		pterm.EnableStyling()
	} else {
		pterm.DisableStyling()
	}

	pterm.SetDefaultOutput(writer)
	fn()
	pterm.SetDefaultOutput(os.Stdout)

	if oldRaw {
		pterm.DisableStyling()
	} else {
		pterm.EnableStyling()
	}
}

func withPtermState(styled bool, fn func()) {
	ptermMu.Lock()
	defer ptermMu.Unlock()

	oldRaw := pterm.RawOutput
	if styled {
		pterm.EnableStyling()
	} else {
		pterm.DisableStyling()
	}

	fn()

	if oldRaw {
		pterm.DisableStyling()
	} else {
		pterm.EnableStyling()
	}
}

func isTerminalWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func isTerminalReader(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func sameWriter(a io.Writer, b io.Writer) bool {
	af, aok := a.(*os.File)
	bf, bok := b.(*os.File)
	if !aok || !bok {
		return false
	}
	return af == bf
}
