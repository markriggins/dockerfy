package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)

type TemplateContext struct {
	env, secrets map[string]string
}

func (c *TemplateContext) Env() map[string]string {
	if c.env != nil {
		return c.env
	}
	c.env = make(map[string]string)

	for _, i := range os.Environ() {
		sep := strings.Index(i, "=")
		c.env[i[0:sep]] = i[sep+1:]
	}
	return c.env
}

//
// Make secrets available under .Secret.VARNAME in the TemplateContext
//
func (c *TemplateContext) Secret() map[string]string {
	if c.secrets != nil {
		return c.secrets
	}
	c.secrets = make(map[string]string)

	// Secrets can come from (in order of precedence):
	// 1) -secrets <file> (one or more times)
	// 2) $SECRETS_FILE

	secretsFileNames := secretsFlag[:]
	if os.Getenv("SECRETS_FILE") != "" {
		secretsFileNames = append(secretsFileNames, os.Getenv("SECRETS_FILE"))
	}

	for _, secretsFileName := range secretsFileNames {

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
				c.secrets[key] = value
				if verboseFlag {
					log.Printf(" %s", key)
				}
			}
		} else if strings.HasSuffix(secretsFileName, ".json") {
			jsonData, err := ioutil.ReadAll(secretsFile)
			if err != nil {
				log.Fatalf("Error reading JSON secrets file '%s':%s", secretsFileName, err)
			}
			var secrets map[string]string
			err = json.Unmarshal(jsonData, &secrets)
			if err != nil {
				log.Fatalf("Error reading JSON secrets file '%s':%s", secretsFileName, err)
			}
			for key, value := range secrets {
				c.secrets[key] = value
				if verboseFlag {
					log.Printf(" %s", key)
				}
			}
		} else {
			log.Fatalf("Unknown file extension '%s' must end with .env or .json\n", secretsFileName)
		}
		log.Println("")
	}

	return c.secrets
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func contains(item map[string]string, key string) bool {
	if _, ok := item[key]; ok {
		return true
	}
	return false
}

func defaultValue(args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("default called with no values!")
	}

	if len(args) > 0 {
		if args[0] != nil {
			return args[0].(string), nil
		}
	}

	if len(args) > 1 {
		if args[1] == nil {
			return "", fmt.Errorf("default called with nil default value!")
		}

		if _, ok := args[1].(string); !ok {
			return "", fmt.Errorf("default is not a string value. hint: surround it w/ double quotes.")
		}

		return args[1].(string), nil
	}

	return "", fmt.Errorf("default called with no default value")
}

func parseUrl(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		log.Fatalf("unable to parse url %s: %s", rawurl, err)
	}
	return u
}

func add(arg1, arg2 int) int {
	return arg1 + arg2
}

//
// Execute the string_template under the TemplateContext, and
// return the result as a string
//
func string_template_eval(string_template string) string {
	var result bytes.Buffer
	t := template.New("String Template")

	t, err := t.Parse(string_template)
	if err != nil {
		log.Fatalf("unable to parse template: %s", err)
	}

	err = t.Execute(&result, &TemplateContext{})
	if err != nil {
		log.Fatalf("template error: %s\n", err)
	}

	return result.String()
}

//
// Execute the template at templatePath under the TemplateContext and write it to destPath
//
func generateFile(templatePath, destPath string) bool {
	tmpl := template.New(filepath.Base(templatePath)).Funcs(template.FuncMap{
		"contains": contains,
		"exists":   exists,
		"split":    strings.Split,
		"replace":  strings.Replace,
		"default":  defaultValue,
		"parseUrl": parseUrl,
		"atoi":     strconv.Atoi,
		"add":      add,
	})

	if len(delims) > 0 {
		tmpl = tmpl.Delims(delims[0], delims[1])
	}
	tmpl, err := tmpl.ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("unable to parse template: %s", err)
	}

	dest := os.Stdout
	if destPath != "" {
		dest, err = os.Create(destPath)
		if err != nil {
			log.Fatalf("unable to create %s", err)
		}
		defer dest.Close()
		log.Printf("Template %s --> %s\n", templatePath, destPath)
	}

	err = tmpl.ExecuteTemplate(dest, filepath.Base(templatePath), &TemplateContext{})
	if err != nil {
		log.Fatalf("template error: %s\n", err)
	}

	if fi, err := os.Stat(destPath); err == nil {
		if err := dest.Chmod(fi.Mode()); err != nil {
			log.Fatalf("unable to chmod temp file: %s\n", err)
		}
		if err := dest.Chown(int(fi.Sys().(*syscall.Stat_t).Uid), int(fi.Sys().(*syscall.Stat_t).Gid)); err != nil {
			log.Fatalf("unable to chown temp file: %s\n", err)
		}
	}

	return true
}
