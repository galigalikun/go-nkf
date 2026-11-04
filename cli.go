package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

const helpText = `Usage: go-nkf [options] [file ...]
Without files, read standard input. Use - for standard input, -- to end options.
Output goes to standard output; the default output encoding is ISO-2022-JP.

  -j / -s / -e            Output ISO-2022-JP / Shift_JIS / EUC-JP
  -w / -w8 / -w80        Output UTF-8 without / with / without BOM
  -w16[BL][0]            Output UTF-16 (default BE, BOM; 0 omits BOM)
  -w32[BL][0]            Output UTF-32 (default BE, BOM; 0 omits BOM)
  -J / -S / -E / -W      Assume ISO-2022-JP / Shift_JIS / EUC-JP / UTF-8 input
  -W16[BL] / -W32[BL]    Assume UTF-16 / UTF-32 input (default BE)
  --ic=NAME / --oc=NAME   Specify input / output encoding
  -g / --guess=1         Print encoding only
  -g2 / --guess[=2]      Print encoding, BOM and newline information
  -Lu / -Lw / -Lm        Convert newlines to LF / CRLF / CR
  -d / -c / -L0          Convert to LF / CRLF / preserve newlines
  -x / -X               Preserve / widen half-width kana (default: widen)
  -t                     Copy input bytes without conversion
  --help / --version     Show help / version

Short options can be combined, for example -SwLu.
Supported names: ASCII, UTF-8, UTF-8-BOM, Shift_JIS (CP932, Windows-31J),
EUC-JP, ISO-2022-JP, UTF-16[BE|LE][-BOM], UTF-32[BE|LE][-BOM].
This implements a subset of nkf options. Unsupported options are errors.
`

type cliOptions struct {
	input        codeset
	output       codeset
	guess        int
	newline      string
	preserveKana bool
	passthrough  bool
	help         bool
	version      bool
	files        []string
}

func parseOptions(args []string) (cliOptions, error) {
	opts := cliOptions{output: codeset{name: "ISO-2022-JP"}}
	endOptions := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if endOptions || arg == "-" || !strings.HasPrefix(arg, "-") {
			opts.files = append(opts.files, arg)
			continue
		}
		if arg == "--" {
			endOptions = true
			continue
		}
		if strings.HasPrefix(arg, "--") {
			name, value, hasValue := strings.Cut(arg, "=")
			if name == "--ic" || name == "--oc" {
				if !hasValue && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					value = args[i]
				}
				if value == "" {
					return opts, fmt.Errorf("%s requires an encoding name", name)
				}
				c, err := lookupCodeset(value)
				if err != nil {
					return opts, err
				}
				if name == "--ic" {
					opts.input = c
				} else {
					opts.output = c
				}
				continue
			}
			if name == "--guess" {
				switch value {
				case "", "2":
					opts.guess = 2
				case "1":
					opts.guess = 1
				default:
					return opts, fmt.Errorf("invalid guess mode %q: use 1 or 2", value)
				}
				if hasValue && value == "" {
					return opts, errors.New("--guess requires 1 or 2 after =")
				}
				continue
			}
			if hasValue {
				return opts, fmt.Errorf("unsupported option %q", arg)
			}
			switch name {
			case "--help":
				opts.help = true
				return opts, nil
			case "--version":
				opts.version = true
				return opts, nil
			case "--jis":
				opts.output = codeset{name: "ISO-2022-JP"}
			case "--sjis":
				opts.output = codeset{name: "Shift_JIS"}
			case "--euc":
				opts.output = codeset{name: "EUC-JP"}
			case "--jis-input":
				opts.input = codeset{name: "ISO-2022-JP"}
			case "--sjis-input":
				opts.input = codeset{name: "Shift_JIS"}
			case "--euc-input":
				opts.input = codeset{name: "EUC-JP"}
			default:
				return opts, fmt.Errorf("unsupported option %q (see --help)", arg)
			}
			continue
		}
		if err := parseShortOptions(arg[1:], &opts); err != nil {
			return opts, err
		}
	}
	if opts.passthrough && opts.guess != 0 {
		return opts, errors.New("-t cannot be combined with --guess or -g")
	}
	return opts, nil
}

func parseShortOptions(arg string, opts *cliOptions) error {
	for i := 0; i < len(arg); i++ {
		switch arg[i] {
		case 'j', 'n':
			opts.output = codeset{name: "ISO-2022-JP"}
		case 's':
			opts.output = codeset{name: "Shift_JIS"}
		case 'e':
			opts.output = codeset{name: "EUC-JP"}
		case 'J':
			opts.input = codeset{name: "ISO-2022-JP"}
		case 'S':
			opts.input = codeset{name: "Shift_JIS"}
		case 'E':
			opts.input = codeset{name: "EUC-JP"}
		case 'w', 'W':
			c, consumed := parseUnicodeOption(arg[i+1:], arg[i] == 'w')
			if arg[i] == 'w' {
				opts.output = c
			} else {
				opts.input = c
			}
			i += consumed
		case 'g':
			opts.guess = 1
			if i+1 < len(arg) && (arg[i+1] == '1' || arg[i+1] == '2') {
				i++
				opts.guess = int(arg[i] - '0')
			}
		case 'L':
			i++
			if i == len(arg) {
				return errors.New("-L requires u, w, m or 0")
			}
			switch arg[i] {
			case 'u':
				opts.newline = "\n"
			case 'w':
				opts.newline = "\r\n"
			case 'm':
				opts.newline = "\r"
			case '0':
				opts.newline = ""
			default:
				return fmt.Errorf("invalid newline mode -L%c", arg[i])
			}
		case 'd':
			opts.newline = "\n"
		case 'c':
			opts.newline = "\r\n"
		case 'x':
			opts.preserveKana = true
		case 'X':
			opts.preserveKana = false
		case 't':
			opts.passthrough = true
		case 'v':
			opts.version = true
		default:
			return fmt.Errorf("unsupported option -%c (see --help)", arg[i])
		}
	}
	return nil
}

func parseUnicodeOption(suffix string, output bool) (codeset, int) {
	c := codeset{name: "UTF-8"}
	consumed := 0
	switch {
	case strings.HasPrefix(suffix, "8"):
		consumed = 1
		c.bom = output
	case strings.HasPrefix(suffix, "16"), strings.HasPrefix(suffix, "32"):
		c.name = "UTF-" + suffix[:2] + "BE"
		c.bom = output
		consumed = 2
		if len(suffix) > consumed && (suffix[consumed] == 'B' || suffix[consumed] == 'L') {
			c.name = "UTF-" + suffix[:2] + string(suffix[consumed]) + "E"
			consumed++
		}
	}
	if output && consumed > 0 && len(suffix) > consumed && suffix[consumed] == '0' {
		c.bom = false
		consumed++
	}
	return c, consumed
}

func runCLI(args []string, stdin io.Reader, stdout io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if opts.help {
		_, err := io.WriteString(stdout, helpText)
		return err
	}
	if opts.version {
		_, err := io.WriteString(stdout, "go-nkf (Go implementation; nkf option subset)\n")
		return err
	}
	files := opts.files
	if len(files) == 0 {
		files = []string{"-"}
	}
	var failures []error
	for _, name := range files {
		var data []byte
		if name == "-" {
			data, err = io.ReadAll(stdin)
		} else {
			data, err = os.ReadFile(name)
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
			continue
		}
		output, err := convertInput(data, opts)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
			continue
		}
		if opts.guess != 0 && len(files) > 1 {
			if _, err := fmt.Fprintf(stdout, "%s: ", name); err != nil {
				return err
			}
		}
		if n, err := stdout.Write(output); err != nil {
			return err
		} else if n != len(output) {
			return io.ErrShortWrite
		}
		// Multiple inputs form one output stream, so emit its BOM only once.
		opts.output.bom = false
	}
	return errors.Join(failures...)
}

func convertInput(data []byte, opts cliOptions) ([]byte, error) {
	if opts.passthrough {
		return data, nil
	}
	c := opts.input
	if c.name == "" {
		var err error
		c, err = detectCodeset(data)
		if err != nil {
			if opts.guess != 0 {
				return []byte("BINARY\n"), nil
			}
			return nil, err
		}
	} else if actual, ok := detectBOM(data); ok && len(c.marker()) > 0 {
		// A BOM determines endianness within the explicitly selected UTF family.
		if c.family() != actual.family() {
			return nil, fmt.Errorf("input BOM indicates %s, but input encoding is %s", actual.name, c.name)
		}
		c = actual
	} else {
		c.bom = false
	}
	decoded, err := decodeText(data, c)
	if err != nil {
		if opts.guess != 0 && opts.input.name == "" {
			return []byte("BINARY\n"), nil
		}
		return nil, err
	}
	if opts.guess != 0 {
		return []byte(formatDetection(c, decoded, opts.guess == 2)), nil
	}
	text := string(decoded)
	if !opts.preserveKana {
		text = widenKana(text)
	}
	if opts.newline != "" {
		text = strings.ReplaceAll(text, "\r\n", "\n")
		text = strings.ReplaceAll(text, "\r", "\n")
		text = strings.ReplaceAll(text, "\n", opts.newline)
	}
	return encodeText([]byte(text), opts.output)
}

func formatDetection(c codeset, decoded []byte, detail bool) string {
	name := c.name
	if strings.HasPrefix(name, "UTF-16") || strings.HasPrefix(name, "UTF-32") {
		name = name[:6]
		if detail {
			name += " " + c.name[6:]
		}
	}
	if detail {
		if c.bom {
			name += " (BOM)"
		}
		if newline := guessNewline(decoded); newline != "" {
			name += " (" + newline + ")"
		}
	}
	return name + "\n"
}

func widenKana(text string) string {
	var output, kana strings.Builder
	flush := func() {
		if kana.Len() > 0 {
			output.WriteString(norm.NFC.String(width.Widen.String(kana.String())))
			kana.Reset()
		}
	}
	for _, r := range text {
		if r >= 0xff61 && r <= 0xff9f {
			kana.WriteRune(r)
		} else {
			flush()
			output.WriteRune(r)
		}
	}
	flush()
	return output.String()
}
