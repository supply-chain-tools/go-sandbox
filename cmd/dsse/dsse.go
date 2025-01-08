package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/supply-chain-tools/go-sandbox/gitrelease"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const usage = `Usage:
    dsse [command] [options] [dsse path]

COMMANDS
        sign
                Add signature to a DSSE.
        payload
                Show DSSE payload.
        pae
                Show PAE of DSSE payload.

SIGN OPTIONS
        --keyfile
                Path to SSH private key to sign DSSE with.

GLOBAL OPTIONS
        --help, -h
                Show help.
        --debug
                Output debug information.

Add signature to dsse.json
    $ dsse sign --keyfile ~/.ssh/id_ed25519 dsse.json

Show payload
    $ dsse payload dsse.json

Show PAE
    $ dsse pae dsse.json`

const (
	keyfilePathRegex      = "(~/|(../)+)?[a-zA-Z0-9-_/.]+"
	sshNamespace          = "dsse"
	defaultFilePermission = 0644
)

func main() {
	optionsAndArgs, err := parseOptionsAndArgs()
	if err != nil {
		print("Failed to parse input: ", err.Error(), "\n")
		os.Exit(1)
	}

	switch optionsAndArgs.command {
	case "sign":
		envelope, err := load(optionsAndArgs.dssePath)
		if err != nil {
			print("Failed to read DSSE: ", err.Error(), "\n")
			os.Exit(1)
		}

		err = signAndWrite(optionsAndArgs.dssePath, envelope, optionsAndArgs.keyfilePath)
		if err != nil {
			print("Failed to sign DSSE: ", err.Error(), "\n")
			os.Exit(1)
		}
	case "payload":
		envelope, err := load(optionsAndArgs.dssePath)
		if err != nil {
			print("Failed to read DSSE: ", err.Error(), "\n")
			os.Exit(1)
		}

		payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
		if err != nil {
			print("Failed to decode DSSE payload", err.Error(), "\n")
			os.Exit(1)
		}

		fmt.Println(string(payload))
	case "pae":
		envelope, err := load(optionsAndArgs.dssePath)
		if err != nil {
			print("Failed to read DSSE: ", err.Error(), "\n")
			os.Exit(1)
		}

		pae, err := getPAE(envelope)
		if err != nil {
			print("Failed to get PAE: ", err.Error(), "\n")
		}

		fmt.Printf(string(pae))
	default:
		print("Unknown command: ", optionsAndArgs.command, "\n")
		os.Exit(1)
	}
}

type OptionsAndArgs struct {
	keyfilePath string
	dssePath    string
	command     string
}

func parseOptionsAndArgs() (*OptionsAndArgs, error) {
	flag.Usage = func() {
		fmt.Println(usage)
	}

	flags := flag.NewFlagSet("all", flag.ExitOnError)
	var help, h, debugMode bool
	var keyfilePath string
	flags.BoolVar(&help, "help", false, "")
	flags.BoolVar(&h, "h", false, "")
	flags.BoolVar(&debugMode, "debug", false, "")
	flags.StringVar(&keyfilePath, "keyfile", "", "")

	if len(os.Args) < 3 {
		fmt.Println(usage)
		os.Exit(1)
	}

	command := os.Args[1]

	err := flags.Parse(os.Args[2:])
	if err != nil || help || h {
		fmt.Println(usage)
		os.Exit(1)
	}

	dssePath := flags.Args()[0]
	if dssePath == "" {
		print("DSSE path is required")
		os.Exit(1)
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if debugMode {
		opts.Level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	if command == "sign" {
		if keyfilePath == "" {
			return nil, fmt.Errorf("--keyfile must be specified")
		}

		match, _ := regexp.MatchString(keyfilePathRegex, keyfilePath)
		if !match {
			return nil, fmt.Errorf("invalid character in keyfile path '%s'", keyfilePath)
		}

		if strings.HasPrefix(keyfilePath, "-") || strings.Contains(keyfilePath, "--") {
			return nil, fmt.Errorf("invalid format of keyfile path '%s'", keyfilePath)
		}

		keyfilePath, err = filepath.Abs(keyfilePath)
		if err != nil {
			return nil, fmt.Errorf("unable to get absolute path of '%s': %w", keyfilePath, err)
		}

		_, err = os.Stat(keyfilePath)
		if err != nil {
			return nil, fmt.Errorf("unable to verify keyfile': %w", err)
		}
	}

	return &OptionsAndArgs{
		keyfilePath: keyfilePath,
		dssePath:    dssePath,
		command:     command,
	}, nil
}

func load(DSSEPath string) (*gitrelease.Envelope, error) {
	data, err := os.ReadFile(DSSEPath)
	if err != nil {
		return nil, err
	}

	envelope := gitrelease.Envelope{}
	err = json.Unmarshal(data, &envelope)
	if err != nil {
		return nil, err
	}

	return &envelope, nil
}

func getPAE(envelope *gitrelease.Envelope) ([]byte, error) {
	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return nil, err
	}

	preAuthenticationEncoding, err := gitrelease.PreAuthenticationEncoding(payload)
	if err != nil {
		return nil, err
	}

	return preAuthenticationEncoding, nil
}

func signAndWrite(DSSEPath string, envelope *gitrelease.Envelope, keyfilePath string) error {
	existing, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return err
	}

	err = printPayload(envelope)
	if err != nil {
		return err
	}

	signature, errors := sign(existing, keyfilePath)
	if len(errors) > 0 {
		allErrors := ""
		for _, err := range errors {
			allErrors += err.Error() + ";"
		}

		return fmt.Errorf("failed to sign file: %s", allErrors)
	}

	envelope.Signatures = append(envelope.Signatures, gitrelease.Signature{
		Sig: signature,
	})

	output, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	err = os.WriteFile(DSSEPath, output, defaultFilePermission)
	if err != nil {
		return err
	}

	return nil
}

func printPayload(envelope *gitrelease.Envelope) error {
	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return err
	}
	var t any
	err = json.Unmarshal(payload, &t)
	if err != nil {
		return err
	}

	formatted, err := json.MarshalIndent(t, "  ", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(formatted))

	return nil
}

func sign(payload []byte, keyfilePath string) (signedPayload string, errors []error) {
	preAuthenticationEncoding, err := gitrelease.PreAuthenticationEncoding(payload)
	if err != nil {
		errors = append(errors, fmt.Errorf("unable to create signature payload: %w", err))
		return "", errors
	}

	tmpFile, err := os.CreateTemp("", "payload")
	if err != nil {
		errors = append(errors, fmt.Errorf("unable to create temporary file: %w", err))
		return "", errors
	}

	defer func() {
		err := os.Remove(tmpFile.Name())
		if err != nil {
			errors = append(errors, fmt.Errorf("unable to delete temporary file %s: %w", tmpFile.Name(), err))
		}
	}()

	_, err = tmpFile.Write(preAuthenticationEncoding)
	if err != nil {
		errors = append(errors, fmt.Errorf("unable to write to temporary file: %w", err))
		return "", errors
	}
	err = tmpFile.Sync()
	if err != nil {
		errors = append(errors, fmt.Errorf("unable to sync temporary file: %w", err))
		return "", errors
	}

	command := []string{"ssh-keygen", "-Y", "sign", "-n", sshNamespace, "-f", keyfilePath, tmpFile.Name()}

	fmt.Printf("Executing command: %s\n", command)
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		errors = append(errors, err)
		return "", errors
	}

	signatureFile := tmpFile.Name() + ".sig"
	defer func() {
		err = os.Remove(tmpFile.Name() + ".sig")
		if err != nil {
			errors = append(errors, fmt.Errorf("unable to delete temporary signature %s: %w", signatureFile, err))
		}
	}()

	sig, err := os.ReadFile(signatureFile)
	if err != nil {
		errors = append(errors, fmt.Errorf("unable to read signature from %s: %w", signatureFile, err))
		return "", errors
	}

	signature := base64.StdEncoding.EncodeToString(sig)
	return signature, errors
}
