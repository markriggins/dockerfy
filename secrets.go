package main

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
    "sync"
)

// Mutex to protect critical regions when performing file system related activities
// with possible race conditions such as checking for the existance of directory
// before creating it.
var fileSysCreateMutex sync.Mutex

// http://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang/21067803#21067803
// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return err
}

//
// return a list of secrets file names from the --secrets-files option and env var  $SECRETS_FILES
//
// Secrets can come from (in order of precedence):
// 1) --secrets-file <file> (one or more times)
// 2) $SECRETS_FILES
func getSecretsFileNames() []string {

	var secretsFileNames []string
	//:= make([]string, len(secretsFilesFlag)+len(strings.Split(os.Getenv("SECRETS_FILES"), ":"))+1)

	// backward compatibility for OLD env var
	if os.Getenv("SECRETS_FILE") != "" {
		log.Println("Warning $SECRETS_FILE is deprecated, use $SECRETS_FILES instead")
		secretsFileNames = append(secretsFileNames, os.Getenv("SECRETS_FILE"))
	}
	if os.Getenv("SECRETS_FILES") != "" {
		secretsFileNames = append(secretsFileNames, strings.Split(os.Getenv("SECRETS_FILES"), ":")...)
	}
	// Command line options override the environment, so we process those LAST
	for _, secretsFileName := range secretsFilesFlag {
		secretsFileNames = append(secretsFileNames, strings.Split(secretsFileName, ":")...)
	}
	// Allow template substitutions in file names.   Works for {{ .Env.VAR }}, but not for {{ .Secret.VAR }}
	for i, secretsFileName := range secretsFileNames {
		secretsFileNames[i] = string_template_eval(secretsFileName)
	}

	return secretsFileNames
}

//
// return a map of secrets
//
func getSecrets() map[string]string {

	secrets := make(map[string]string)

	for _, secretsFileName := range getSecretsFileNames() {

		secretsFile, err := os.Open(secretsFileName)
		if err != nil {
			log.Fatalf("Error opening secrets file '%s':%s", secretsFileName, err)
		}
		if verboseFlag {
			log.Printf("Loading secrets from: %s:", secretsFileName)
		}
		defer secretsFile.Close()
		bSecretsFile := bufio.NewReader(secretsFile)

		if strings.HasSuffix(secretsFileName, ".env") {
			for {
				line_bytes, isPrefix, err := bSecretsFile.ReadLine()
				line := string(line_bytes)
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Fatalf("Error reading secrets file '%s':%s", secretsFileName, err)
				}
				if isPrefix {
					log.Fatal("Error secrets file too long: ", secretsFileName)
				}
				if strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) < 2 {
					continue
				}
				key, value := parts[0], strings.Trim(strings.TrimSpace(parts[1]), `'"`)
				secrets[key] = value
				if debugFlag {
					log.Printf("loaded secret: %s", key)
				}
			}
		} else if strings.HasSuffix(secretsFileName, ".json") {
			jsonData, err := ioutil.ReadAll(secretsFile)
			if err != nil {
				log.Fatalf("Error reading JSON secrets file '%s':%s", secretsFileName, err)
			}
			err = json.Unmarshal(jsonData, &secrets)
			if err != nil {
				log.Fatalf("Error reading JSON secrets file '%s':%s", secretsFileName, err)
			}
			for key, value := range secrets {
				secrets[key] = value
				if debugFlag {
					log.Printf("loaded secret: %s", key)
				}
			}
		} else {
			log.Fatalf("Unknown file extension '%s' must end with .env or .json\n", secretsFileName)
		}
		log.Println("")
	}
	return secrets
}

// Note that secrets files are typically readable only the root user, and node programs and python programs
// may benefit from the illusion that there is a single SECRETS_FILE (combining all keys)
//
// If this command is running under a different user account, then copy the SECRETS_FILES
// and --secret-files into the home directory .secrets (as an emphermeral file)
// and make them readable by the by the user account in case the application wants
// to read the file directly, instead of just using a template to alter a config file.
//
// Set the cmd.Env['SECRETS_FILES'] to point to the new, readable copy
func copySecretsFiles(cmd *exec.Cmd) error {

	// If there is no Credential.Uid or its root, then there is no need to copy the secrets files
	// because the root user can always read them, but on the otherhand, the combined file is useful
	// so lets just do it.

	// if cmd.SysProcAttr == nil || cmd.SysProcAttr.Credential == nil || cmd.SysProcAttr.Credential.Uid == 0 {
	//  return nil
	// }

	// If we are not running in a container, then do not copy secrets files because the copies
	// will not be ephemeral
	if _, err := os.Stat("/.dockerenv"); os.IsNotExist(err) {
		return nil
	}

    fileSysCreateMutex.Lock()
    defer fileSysCreateMutex.Unlock()

	cmdUid := os.Getuid()
	cmdGid := os.Getgid()
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Credential != nil {
		cmdUid = int(cmd.SysProcAttr.Credential.Uid)
		cmdGid = int(cmd.SysProcAttr.Credential.Gid)
	}
	if cmdUser, err := user.LookupId(strconv.Itoa(cmdUid)); err != nil {
		return err
	} else {

		// Pass new SECRETS_FILES to the cmd.Env

		cmd.Env = make([]string, len(os.Environ())+2, len(os.Environ())+2)
		envCount := 0
		// Copy all ENV vars, except SECRETS_FILE.*
		for _, envLine := range os.Environ() {
			if !strings.HasPrefix(envLine, "SECRETS_FILE") {
				cmd.Env[envCount] = envLine
				envCount++
			}
		}

		secretsDir := cmdUser.HomeDir + "/.secrets/"

		cmd.Env[envCount] = "SECRETS_FILE=" + secretsDir + "combined_secrets.json"
		envCount++

		// Rebase all individual secrets-files paths to secretsDir
		secretsFileNames := getSecretsFileNames()
		newSecretsFileNames := make([]string, len(secretsFileNames))

		for i, secretsFileName := range getSecretsFileNames() {
			newSecretsFileNames[i] = secretsDir + filepath.Base(secretsFileName)
		}
		cmd.Env[envCount] = "SECRETS_FILES=" + strings.Join(newSecretsFileNames, ":")
		envCount++

		// If we have not already done so, then copy all the secrets files to secretsDir
		// and write the combined_secrets.json file

		if _, err := os.Stat(secretsDir); err == nil {
			// we've already created it for some other Cmd run as this cmdUser
			return nil
		}

		// Create ~/.secrets dir for the cmdUser
		if _, err := os.Stat(secretsDir); os.IsNotExist(err) {
			// path/to/whatever does not exist
			if err := os.Mkdir(secretsDir, 0700); err != nil {
				return err
			}
			if err := os.Chown(secretsDir, cmdUid, cmdGid); err != nil {
				return err
			}
		}

		// Create a combined secrets file with all secrets
		if jsonData, err := json.MarshalIndent(getSecrets(), "", "    "); err == nil {
			if err := ioutil.WriteFile(secretsDir+"combined_secrets.json", jsonData, 0400); err != nil {
				return err
			}
			if err := os.Chown(secretsDir+"combined_secrets.json", cmdUid, cmdGid); err != nil {
				return err
			}
		} else {
			return err
		}

		// Copy all the individual secrets files into secretsDir
		for _, secretsFileName := range getSecretsFileNames() {
			copyName := secretsDir + "/" + filepath.Base(secretsFileName)
			if err := copyFileContents(secretsFileName, copyName); err != nil {
				return err
			}
			if err := os.Chown(copyName, cmdUid, cmdGid); err != nil {
				return err
			}
			if err := os.Chmod(copyName, 0400); err != nil {
				return err
			}
		}

	}

	return nil
}
